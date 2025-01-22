package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

type AudioFile struct {
	Path  string
	Index int
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

}

func getMP3Files(inputDir string) ([]AudioFile, error) {
	files, err := os.ReadDir(inputDir)
	if err != nil {
		return nil, err
	}

	var audioFiles []AudioFile
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".mp3" {
			newName, index, err := parseIndex(file.Name(), len(files))
			if err != nil {
				return nil, err
			}
			audioFiles = append(audioFiles, AudioFile{Path: filepath.Join(inputDir, newName), Index: index})
		}
	}
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
