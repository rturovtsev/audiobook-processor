package main

import (
	"flag"
	"fmt"
	"github.com/bogem/id3v2/v2"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

type AudioFile struct {
	Path    string
	NewName string
	Index   int
}

type ByIndex []AudioFile

func (a ByIndex) Len() int           { return len(a) }
func (a ByIndex) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByIndex) Less(i, j int) bool { return a[i].Index < a[j].Index }

func main() {
	inputDir := flag.String("input", "", "Path to the directory with MP3 files")
	author := flag.String("author", "", "Author of the book")
	title := flag.String("title", "", "Title of the book")
	flag.Parse()

	if *inputDir == "" {
		log.Fatalf("imput directory is required")
	}

	mp3Files, err := getMP3Files(*inputDir)
	if err != nil {
		log.Fatalf("error getting MP3 files: %v", err)
	}

	updateMetaTags(mp3Files, author, title)

	err = mergeFilesToM4B(mp3Files, *author, *title)
	if err != nil {
		log.Fatalf("error merging files: %v", err)
	}
}

func getMP3Files(inputDir string) ([]AudioFile, error) {
	files, err := os.ReadDir(inputDir)
	if err != nil {
		return nil, err
	}

	var audioFiles []AudioFile
	var newName string
	var index int

	fileCount := len(files)
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".mp3" {
			oldName := file.Name()
			oldPath := filepath.Join(inputDir, oldName)
			newPath := filepath.Join(inputDir, newName)
			newName, index, err = parseIndex(oldName, fileCount)
			if err != nil {
				return nil, err
			}
			renameFiles(oldPath, newPath)
			audioFiles = append(audioFiles, AudioFile{Path: newPath, NewName: newName, Index: index})
		}
	}
	sort.Sort(ByIndex(audioFiles))

	return audioFiles, nil
}

func parseIndex(filename string, countAllFile int) (newName string, index int, err error) {
	re := regexp.MustCompile(`^\d+`)
	match := re.FindString(filename)

	if match == "" {
		return "", 0, fmt.Errorf("no number found in %s", filename)
	}

	index, err = strconv.Atoi(match)
	if err != nil {
		return "", 0, err
	}
	if countAllFile < 10 {
		newName = fmt.Sprintf("%d.mp3", index)
	} else if countAllFile < 100 {
		newName = fmt.Sprintf("02%d.mp3", index)
	} else if countAllFile < 1000 {
		newName = fmt.Sprintf("03%d.mp3", index)
	} else {
		newName = fmt.Sprintf("04%d.mp3", index)
	}

	return newName, index, nil
}

func renameFiles(oldPath, newPath string) {
	err := os.Rename(oldPath, newPath)
	if err != nil {
		log.Fatalf("error renaming file: %v", err)
	}
}

func updateMetaTags(files []AudioFile, author, title *string) {
	totalFiles := len(files)
	for _, file := range files {
		tag, err := id3v2.Open(file.Path, id3v2.Options{Parse: true})
		if err != nil {
			log.Fatalf("error opening tag for file %s: %v", file.Path, err)
		}
		defer tag.Close()

		// Очистка существующего исполнителя альбома
		tag.DeleteFrames("TPE2")

		// Установка номера трека
		trackNum := fmt.Sprintf("%s/%d", file.NewName, totalFiles)
		tag.AddTextFrame("TRCK", tag.DefaultEncoding(), trackNum)

		// Установка номера диска, если требуется
		tag.AddTextFrame("TPOS", tag.DefaultEncoding(), "1")

		// Удаление комментариев
		tag.DeleteFrames("COMM")

		// Установка имени автора и названия альбома, если предоставлены
		if author != nil && *author != "" {
			tag.AddTextFrame("TPE1", tag.DefaultEncoding(), *author)
		}

		if title != nil && *title != "" {
			tag.AddTextFrame("TALB", tag.DefaultEncoding(), *title)
		}

		// Сохранение изменений в файле
		if err = tag.Save(); err != nil {
			log.Fatalf("error saving tag for file %s: %v", file.Path, err)
		}
	}
}

func mergeFilesToM4B(files []AudioFile, author, title string) error {
	var args []string
	for _, file := range files {
		args = append(args, "-i", file.Path)
	}
	outputFile := filepath.Join(filepath.Dir(files[0].Path), "output.m4b")

	args = append(args, "-c:a", "aac", "-vn", "-ar", "44100", "-ab", "256k", "-f", "mp4", outputFile)

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
