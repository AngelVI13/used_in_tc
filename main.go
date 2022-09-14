package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

func SearchFile(path string, pattern *regexp.Regexp) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("ERROR: Couldn't read file %s: %v", path, err)
		return false
	}

	result := pattern.Find(data)
	return result != nil
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

	files, _ := GetFilesFromDir(dir, fileType)
	for idx, file := range files {
		found := SearchFile(file, searchPattern)
		if found {
			log.Println(idx, file, " <<<<<<<<<<")
		} else {
			log.Println(idx, file)
		}
	}
}
