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
	TabSize = 4
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
	usedInMethod string
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

func ProcessMatch(match []int, text string) SearchResult {
	var (
		start = match[0]
		end   = match[1]

		pretext  = text[:start]
		posttext = text[end:]
	)

	line := strings.Count(pretext, "\n") + 1

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
	matchTxt := (pretext[leftNewLineIdx+1:] +
		text[start:end] +
		posttext[:rightNewLineIdx])
	col := start - leftNewLineIdx

	// TODO: refactor this to another method
	// Search for the file line above the match that has a smaller indentation.
	// When found, try to process a method name from that line -> this is used to
	// continue searching for usages incase the match does not occur inside a test case file
	matchCol := -1
	usedInMethod := ""
	lines := strings.Split(pretext, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		textLine := lines[i]

		textCol := 0
		for idx, c := range textLine {
			if string(c) != " " {
				textCol = idx
				break
			}
		}

		// During the first iteration we find the indentation level of match
		if i == len(lines)-1 {
			matchCol = textCol
			continue
		}

		if textCol != matchCol-TabSize {
			continue
		}

		methodPattern, err := regexp.Compile(`def\s*(?P<name>.*?)\(`)
		if err != nil {
			log.Fatalf("Couldn't compile method declaration regex: %v", err)
		}
		nameIdx := methodPattern.SubexpIndex("name")

		match := methodPattern.FindStringSubmatch(textLine)
		if match == nil {
			break
		}

		usedInMethod = match[nameIdx]
		log.Println(textLine, "<<<", usedInMethod)
		break
	}

	return SearchResult{
		line:         line,
		col:          col,
		matchLineTxt: matchTxt,
		usedInMethod: usedInMethod,
	}
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
		searchResult := ProcessMatch(match, text)
		searchResult.file = path

		results = append(results, searchResult)
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

type SearchJob[T SearchTerm] struct {
	filepath string
	pattern  T
}

func worker[T SearchTerm](jobs <-chan SearchJob[T], results chan<- FileResult, done chan<- int) {
	numOfResults := 0
	for j := range jobs {
		found := SearchFile(j.filepath, j.pattern)
		if found == nil {
			continue
		}
		results <- *found
		numOfResults++
	}
	done <- numOfResults
}

func SearchInRepo[T SearchTerm](dir, fileType string, searchPattern T) []FileResult {
	files, err := GetFilesFromDir(dir, fileType)
	if err != nil {
		log.Fatalf("Couldn't get list of files for dir %s: %v", dir, err)
	}

	var (
		workerNum = 4
		jobNum    = len(files)
		jobs      = make(chan SearchJob[T], jobNum)
		results   = make(chan FileResult)
		done      = make(chan int, workerNum)
	)

	for i := 0; i < workerNum; i++ {
		go worker(jobs, results, done)
	}

	for _, file := range files {
		jobs <- SearchJob[T]{
			filepath: file,
			pattern:  searchPattern,
		}
	}
	close(jobs)

	fileResults := []FileResult{}
	resultCount := 0
	workersDone := 0
	resultsProcessed := 0
	for {
		select {
		case count := <-done:
			resultCount += count
			workersDone++
		case result := <-results:
			resultsProcessed++
			fileResults = append(fileResults, result)
		}

		if workersDone == workerNum && resultsProcessed == resultCount {
			break
		}
	}

	return fileResults

}

func SearchForUsagesInTc[T SearchTerm](dir, fileType string, searchPattern T) []string {
	testCases := []string{}
	nonTcMatches := []FileResult{}

	results := SearchInRepo(dir, fileType, searchPattern)
	for _, result := range results {
		fmt.Println(result)
		if result.tcId != "" {
			testCases = append(testCases, result.tcId)
		} else {
			nonTcMatches = append(nonTcMatches, result)
		}
	}

	// TODO: finish this by recursively continuing the search until it reaches a test case
	log.Println("Non TC results")
	for _, fileResult := range nonTcMatches {
		for _, searchResult := range fileResult.matches {
			log.Println(searchResult)
			if searchResult.usedInMethod != "" {
				log.Println(searchResult.usedInMethod)
			}
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
	// pattern := `\.outputHeater\.`
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
	log.Printf("Used in test cases (%d):", len(testCases))
	log.Println(testCases)

	log.Println("Elapsed time", time.Since(start).Seconds())
}
