package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	LogFile = "search.log"
)

func GetFilesFromDir(root string, fileType string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(path, fileType) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

type SearchResult struct {
	file         string
	line         int
	col          int
	matchLineTxt string
}

func (r SearchResult) String() string {
	return fmt.Sprintf("\n%s\n%s", r.file, r.matchLineTxt)
}

func SearchFile(path string, pattern *regexp.Regexp) []*SearchResult {
	results := []*SearchResult{}

	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("ERROR: Couldn't read file %s: %v", path, err)
		return []*SearchResult{}
	}
	text := string(data)

	matches := pattern.FindAllStringIndex(text, -1)
	for _, match := range matches {
		start := match[0]
		end := match[1]

		pretext := text[:start]
		posttext := text[end:]
		lineNum := strings.Count(pretext, "\n") + 1

		leftNewLineIdx := strings.LastIndex(pretext, "\n")
		rightNewLineIdx := strings.Index(posttext, "\n")

		matchLineText := pretext[leftNewLineIdx:] + text[start:end] + posttext[:rightNewLineIdx]
		colNum := start - leftNewLineIdx

		results = append(results, &SearchResult{
			file:         path,
			line:         lineNum,
			col:          colNum,
			matchLineTxt: matchLineText,
		})
		log.Println(matchLineText)
	}

	return results
}

func setupLogger(filename string) {
	// Delete old log file
	os.Remove(filename)
	// Set up logging to stdout and file
	logFile, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
}

func main() {
	setupLogger(LogFile)

	pattern := `\.outputHeater\.set_disconnected`
	searchPattern, err := regexp.Compile(pattern)
	if err != nil {
		log.Fatalf("Couldn't compile search pattern %s: %v", pattern, err)
	}

	fileType := ".py"
	dir := "/media/sf_shared/TestAutomation"

	log.Println(searchPattern, fileType, dir)

	start := time.Now()

	files, _ := GetFilesFromDir(dir, fileType)
	count := 0
	for _, file := range files {
		found := SearchFile(file, searchPattern)
		if len(found) == 0 {
			continue
		}

		for _, result := range found {
			count++
			log.Println(count, result)
		}
	}

	log.Println("Elapsed time", time.Since(start).Seconds())
}
