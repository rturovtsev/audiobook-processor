package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
)

type AudioFile struct {
	Path string
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
	for i, file := range files {
		if filepath.Ext(file.Name()) == ".mp3" {
			index := parseIndex(file.Name())
			audioFiles = append(audioFiles, AudioFile{Path: filepath.Join(inputDir, file.Name()), Index: index})
		}
	}
	return audioFiles, nil
}

func parseIndex(filename string) int {
	// ""
	return 0
}
