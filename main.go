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
	return fmt.Sprintf("%d: %s", r.line, r.matchLineTxt)
}

type FileResult struct {
	file    string
	matches []SearchResult
	isTc    bool
	tcId    string
}

func (r FileResult) String() string {
	out := fmt.Sprintf("%s\n", r.file)

	for _, match := range r.matches {
		out += fmt.Sprintf("\t%s\n\n", match)
	}

	return out
}

type SearchTerm interface {
	string | *regexp.Regexp
}

func FindAllStringIndex[T SearchTerm](s string, pattern T) [][]int {
	if p, ok := any(pattern).(*regexp.Regexp); ok {
		return p.FindAllStringIndex(s, -1)
	}

	sub, ok := any(pattern).(string)
	if !ok {
		log.Fatalf(
			"Expected either a regexp.Regexp or a string but got neither: %v",
			pattern,
		)
	}

	results := make([][]int, 0)
	subLen := len(sub)
	currentIdx := 0
	currentTxt := s
	for {
		idx := strings.Index(currentTxt, sub)
		if idx == -1 {
			break
		}

		idx += currentIdx

		results = append(results, []int{idx, idx + subLen})

		currentTxt = currentTxt[idx+subLen:]
		currentIdx = idx + subLen
	}

	return results
}

func ProcessMatch(match []int, text string) (line, col int, matchTxt string) {
	var (
		start = match[0]
		end   = match[1]

		pretext  = text[:start]
		posttext = text[end:]
	)

	line = strings.Count(pretext, "\n") + 1

	leftNewLineIdx := strings.LastIndex(pretext, "\n")
	rightNewLineIdx := strings.Index(posttext, "\n")

	if leftNewLineIdx == -1 || rightNewLineIdx == -1 {
		log.Fatalf(
			"Couldn't find left newline or right newline for match %s: %d %d",
			text[start:end],
			leftNewLineIdx,
			rightNewLineIdx,
		)
	}

	// +1 is needed to ignore the preceding newline
	matchTxt = (pretext[leftNewLineIdx+1:] +
		text[start:end] +
		posttext[:rightNewLineIdx])
	col = start - leftNewLineIdx

	return line, col, matchTxt
}

func ProcessTc(text, path string) (isTc bool, tcId string) {
	tcPathPattern, err := regexp.Compile(`test_cases/.*?/test_.*?\.py`)
	if err != nil {
		log.Fatalf("Couldn't compile TC path pattern: %v", err)
	}
	isTc = tcPathPattern.FindString(path) != ""

	tcId = ""
	if isTc {
		tcIdPattern, err := regexp.Compile(`Polarion ID: (?P<id>[a-zA-Z0-9]+-\d+)`)
		if err != nil {
			log.Fatalf("Couldn't compile TC ID pattern: %v", err)
		}
		idIndex := tcIdPattern.SubexpIndex("id")
		match := tcIdPattern.FindStringSubmatch(text)
		if match != nil {
			tcId = match[idIndex]
		} else {
			log.Printf("ERROR: Couldn't find ID for TC %s", path)
		}
	}

	return isTc, tcId
}

func SearchFile[T SearchTerm](path string, pattern T) *FileResult {
	results := []SearchResult{}

	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("ERROR: Couldn't read file %s: %v", path, err)
		return nil
	}
	text := string(data)

	matches := FindAllStringIndex(text, pattern)
	for _, match := range matches {
		lineNum, colNum, matchLineText := ProcessMatch(match, text)

		results = append(results, SearchResult{
			file:         path,
			line:         lineNum,
			col:          colNum,
			matchLineTxt: matchLineText,
		})
	}

	if len(results) <= 0 {
		return nil
	}

	isTc, tcId := ProcessTc(text, path)
	return &FileResult{
		file:    path,
		matches: results,
		isTc:    isTc,
		tcId:    tcId,
	}
}

func SearchForUsagesInTc[T SearchTerm](dir, fileType string, searchPattern T) []string {
	testCases := []string{}
	files, _ := GetFilesFromDir(dir, fileType)
	for _, file := range files {
		found := SearchFile(file, searchPattern)
		// found := SearchFile(file, searchPatternStr)
		if found == nil {
			continue
		}

		fmt.Println(found)
		if found.tcId != "" {
			testCases = append(testCases, found.tcId)
		}
	}

	return testCases
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

	// TODO: use this flag
	// useRegex := true
	pattern := `\.outputHeater\.set_disconnected`
	searchPattern, err := regexp.Compile(pattern)
	if err != nil {
		log.Fatalf("Couldn't compile search pattern %s: %v", pattern, err)
	}
	// searchPatternStr := `.outputHeater.set_disconnected`

	fileType := ".py"
	dir := "/media/sf_shared/TestAutomation"

	log.Println(searchPattern, fileType, dir)

	start := time.Now()

	testCases := SearchForUsagesInTc(dir, fileType, searchPattern)
	log.Println("Used in test cases:")
	log.Println(testCases)

	log.Println("Elapsed time", time.Since(start).Seconds())
}
