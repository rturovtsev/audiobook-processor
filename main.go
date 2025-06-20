package main

import (
	"encoding/json"
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
	"strings"
)

type AudioFile struct {
	Path     string
	NewName  string
	Index    int
	Duration float64
	Title    string
}

type FFProbeOutput struct {
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
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
		log.Fatalf("input directory is required")
	}

	// Validate input directory exists
	if _, err := os.Stat(*inputDir); os.IsNotExist(err) {
		log.Fatalf("input directory does not exist: %s", *inputDir)
	}

	fmt.Printf("Processing MP3 files in: %s\n", *inputDir)
	mp3Files, err := getMP3Files(*inputDir)
	if err != nil {
		log.Fatalf("error getting MP3 files: %v", err)
	}

	if len(mp3Files) == 0 {
		log.Fatalf("no MP3 files found in directory: %s", *inputDir)
	}

	fmt.Printf("Found %d MP3 files\n", len(mp3Files))

	fmt.Println("Updating metadata tags...")
	err = updateMetaTags(mp3Files, author, title)
	if err != nil {
		log.Fatalf("error updating metadata: %v", err)
	}

	fmt.Println("Merging files to M4B format...")
	err = mergeFilesToM4B(mp3Files, *author, *title)
	if err != nil {
		log.Fatalf("error merging files: %v", err)
	}

	fmt.Println("Audiobook processing completed successfully!")
}

func getMP3Files(inputDir string) ([]AudioFile, error) {
	files, err := os.ReadDir(inputDir)
	if err != nil {
		return nil, err
	}

	var audioFiles []AudioFile
	mp3Count := 0

	// First pass: count MP3 files and collect with original indices
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".mp3" {
			oldName := file.Name()
			oldPath := filepath.Join(inputDir, oldName)
			index, err := extractIndex(oldName)
			if err != nil {
				return nil, err
			}
			audioFiles = append(audioFiles, AudioFile{Path: oldPath, NewName: oldName, Index: index})
			mp3Count++
		}
	}

	// Sort by original index to maintain order
	sort.Sort(ByIndex(audioFiles))

	// Second pass: rename files with proper numbering
	for i, audioFile := range audioFiles {
		newName := generateNewName(i+1, mp3Count)
		newPath := filepath.Join(inputDir, newName)
		
		if audioFile.Path != newPath {
			err := renameFiles(audioFile.Path, newPath)
			if err != nil {
				return nil, err
			}
		}
		
		audioFiles[i].Path = newPath
		audioFiles[i].NewName = newName
		audioFiles[i].Index = i + 1
	}

	// Get duration and title for each file
	for i := range audioFiles {
		duration, err := getFileDuration(audioFiles[i].Path)
		if err != nil {
			return nil, fmt.Errorf("error getting duration for %s: %v", audioFiles[i].Path, err)
		}
		audioFiles[i].Duration = duration
		audioFiles[i].Title = getChapterTitle(audioFiles[i].Path)
	}

	return audioFiles, nil
}

func extractIndex(filename string) (int, error) {
	// Try to find numbers in the filename, not just at the beginning
	re := regexp.MustCompile(`\d+`)
	match := re.FindString(filename)

	if match == "" {
		return 0, fmt.Errorf("no number found in %s", filename)
	}

	index, err := strconv.Atoi(match)
	if err != nil {
		return 0, err
	}

	return index, nil
}

func generateNewName(index, totalFiles int) string {
	if totalFiles < 10 {
		return fmt.Sprintf("%d.mp3", index)
	} else if totalFiles < 100 {
		return fmt.Sprintf("%02d.mp3", index)
	} else if totalFiles < 1000 {
		return fmt.Sprintf("%03d.mp3", index)
	} else {
		return fmt.Sprintf("%04d.mp3", index)
	}
}

func renameFiles(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

func getFileDuration(filePath string) (float64, error) {
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", filePath)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("error running ffprobe: %v", err)
	}

	var probe FFProbeOutput
	err = json.Unmarshal(output, &probe)
	if err != nil {
		return 0, fmt.Errorf("error parsing ffprobe output: %v", err)
	}

	duration, err := strconv.ParseFloat(probe.Format.Duration, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing duration: %v", err)
	}

	return duration, nil
}

func getChapterTitle(filePath string) string {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return generateChapterTitleFromFilename(filePath)
	}
	defer tag.Close()

	// Try to get title from ID3 tag
	titleFrames := tag.GetFrames("TIT2")
	if len(titleFrames) > 0 {
		if textFrame, ok := titleFrames[0].(id3v2.TextFrame); ok {
			title := strings.TrimSpace(textFrame.Text)
			if title != "" {
				return title
			}
		}
	}

	// Fallback to filename-based title
	return generateChapterTitleFromFilename(filePath)
}

func generateChapterTitleFromFilename(filePath string) string {
	filename := filepath.Base(filePath)
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	
	// Remove leading numbers and separators
	re := regexp.MustCompile(`^\d+[._\-\s]*`)
	name = re.ReplaceAllString(name, "")
	
	if name == "" {
		// Extract number for generic chapter name
		numRe := regexp.MustCompile(`^\d+`)
		match := numRe.FindString(filename)
		if match != "" {
			return fmt.Sprintf("Глава %s", match)
		}
		return "Глава"
	}
	
	return name
}

func createChaptersFile(files []AudioFile, author, title string) (string, error) {
	if len(files) == 0 {
		return "", fmt.Errorf("no files provided")
	}

	// Create chapters file in the same directory as the first file
	chaptersFile := filepath.Join(filepath.Dir(files[0].Path), "chapters.txt")
	
	file, err := os.Create(chaptersFile)
	if err != nil {
		return "", fmt.Errorf("error creating chapters file: %v", err)
	}
	defer file.Close()

	// Write FFMETADATA header
	_, err = file.WriteString(";FFMETADATA1\n")
	if err != nil {
		return "", err
	}

	// Write global metadata with additional Plex-friendly tags
	if title != "" {
		_, err = file.WriteString(fmt.Sprintf("title=%s\n", title))
		if err != nil {
			return "", err
		}
		_, err = file.WriteString(fmt.Sprintf("album=%s\n", title))
		if err != nil {
			return "", err
		}
	}

	if author != "" {
		_, err = file.WriteString(fmt.Sprintf("artist=%s\n", author))
		if err != nil {
			return "", err
		}
		_, err = file.WriteString(fmt.Sprintf("album_artist=%s\n", author))
		if err != nil {
			return "", err
		}
	}

	// Add audiobook-specific metadata
	_, err = file.WriteString("genre=Audiobook\n")
	if err != nil {
		return "", err
	}
	_, err = file.WriteString("media_type=audiobook\n")
	if err != nil {
		return "", err
	}

	_, err = file.WriteString("\n")
	if err != nil {
		return "", err
	}

	// Generate chapters with timestamps
	var currentTime float64 = 0

	for _, audioFile := range files {
		startTime := int64(currentTime * 1000) // Convert to milliseconds
		endTime := int64((currentTime + audioFile.Duration) * 1000)

		// Write chapter
		_, err = file.WriteString("[CHAPTER]\n")
		if err != nil {
			return "", err
		}

		_, err = file.WriteString("TIMEBASE=1/1000\n")
		if err != nil {
			return "", err
		}

		_, err = file.WriteString(fmt.Sprintf("START=%d\n", startTime))
		if err != nil {
			return "", err
		}

		_, err = file.WriteString(fmt.Sprintf("END=%d\n", endTime))
		if err != nil {
			return "", err
		}

		_, err = file.WriteString(fmt.Sprintf("title=%s\n\n", audioFile.Title))
		if err != nil {
			return "", err
		}

		currentTime += audioFile.Duration
	}

	return chaptersFile, nil
}

func createSimpleChaptersFile(files []AudioFile) (string, error) {
	if len(files) == 0 {
		return "", fmt.Errorf("no files provided")
	}

	// Create simple chapters file for Plex compatibility
	chaptersFile := filepath.Join(filepath.Dir(files[0].Path), "chapters_simple.txt")
	
	file, err := os.Create(chaptersFile)
	if err != nil {
		return "", fmt.Errorf("error creating simple chapters file: %v", err)
	}
	defer file.Close()

	// Generate chapters in simple format
	var currentTime float64 = 0

	for i, audioFile := range files {
		hours := int(currentTime / 3600)
		minutes := int((currentTime - float64(hours*3600)) / 60)
		seconds := currentTime - float64(hours*3600) - float64(minutes*60)

		// Write chapter in HH:MM:SS.sss format
		_, err = file.WriteString(fmt.Sprintf("CHAPTER%02d=%02d:%02d:%06.3f\n", i+1, hours, minutes, seconds))
		if err != nil {
			return "", err
		}

		_, err = file.WriteString(fmt.Sprintf("CHAPTER%02dNAME=%s\n", i+1, audioFile.Title))
		if err != nil {
			return "", err
		}

		currentTime += audioFile.Duration
	}

	return chaptersFile, nil
}

func updateMetaTags(files []AudioFile, author, title *string) error {
	totalFiles := len(files)
	for i, file := range files {
		tag, err := id3v2.Open(file.Path, id3v2.Options{Parse: true})
		if err != nil {
			return fmt.Errorf("error opening tag for file %s: %v", file.Path, err)
		}
		defer tag.Close()

		// Use UTF-8 encoding for proper cyrillic support
		utf8Encoding := id3v2.EncodingUTF8

		// Очистка существующего исполнителя альбома
		tag.DeleteFrames("TPE2")

		// Установка номера трека в формате "номер/общее количество"
		trackNum := fmt.Sprintf("%d/%d", i+1, totalFiles)
		tag.AddTextFrame("TRCK", utf8Encoding, trackNum)

		// Установка номера диска на 1
		tag.AddTextFrame("TPOS", utf8Encoding, "1")

		// Удаление комментариев (заметок)
		tag.DeleteFrames("COMM")

		// Установка имени автора и названия альбома, если предоставлены
		if author != nil && *author != "" {
			tag.AddTextFrame("TPE1", utf8Encoding, *author)
		}

		if title != nil && *title != "" {
			tag.AddTextFrame("TALB", utf8Encoding, *title)
		}

		// Сохранение изменений в файле
		if err = tag.Save(); err != nil {
			return fmt.Errorf("error saving tag for file %s: %v", file.Path, err)
		}
	}
	return nil
}

func mergeFilesToM4B(files []AudioFile, author, title string) error {
	if len(files) == 0 {
		return fmt.Errorf("no files to merge")
	}

	// Create chapters metadata file
	fmt.Println("Creating chapters metadata...")
	chaptersFile, err := createChaptersFile(files, author, title)
	if err != nil {
		return fmt.Errorf("error creating chapters file: %v", err)
	}
	defer os.Remove(chaptersFile) // Clean up temporary file

	// Also create a simple chapters file for better Plex compatibility
	simpleChaptersFile, err := createSimpleChaptersFile(files)
	if err != nil {
		return fmt.Errorf("error creating simple chapters file: %v", err)
	}
	defer os.Remove(simpleChaptersFile) // Clean up temporary file

	var args []string
	
	// Add input files
	for _, file := range files {
		args = append(args, "-i", file.Path)
	}

	// Add chapters metadata file
	args = append(args, "-i", chaptersFile)

	// Create output filename
	outputDir := filepath.Dir(files[0].Path)
	var outputFile string
	if author != "" && title != "" {
		outputFile = filepath.Join(outputDir, fmt.Sprintf("%s - %s.m4b", author, title))
	} else if title != "" {
		outputFile = filepath.Join(outputDir, fmt.Sprintf("%s.m4b", title))
	} else {
		outputFile = filepath.Join(outputDir, "audiobook.m4b")
	}

	// Create filter_complex for concatenation
	var filterComplex strings.Builder
	for i := 0; i < len(files); i++ {
		if i > 0 {
			filterComplex.WriteString(" ")
		}
		filterComplex.WriteString(fmt.Sprintf("[%d:a]", i))
	}
	filterComplex.WriteString(fmt.Sprintf(" concat=n=%d:v=0:a=1 [out]", len(files)))

	// FFmpeg arguments for M4B conversion with chapters
	args = append(args, 
		"-filter_complex", filterComplex.String(),
		"-map", "[out]",
		"-map_metadata", fmt.Sprintf("%d", len(files)), // Map metadata from chapters file
		"-c:a", "aac",    // Use AAC codec
		"-b:a", "128k",   // Audio bitrate
		"-ar", "44100",   // Sample rate
		"-f", "mp4",      // Output format
		"-brand", "M4B ", // Explicit M4B brand for better compatibility
		"-movflags", "+faststart", // Optimize for streaming
		"-metadata", "media_type=audiobook", // Mark as audiobook
		"-y",             // Overwrite output file
		outputFile)

	fmt.Printf("Creating M4B file with chapters: %s\n", outputFile)
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
