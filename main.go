package main

import (
	"fmt"
	"github.com/alexflint/go-arg"
	"github.com/charmbracelet/lipgloss"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var args struct {
	UseRegex bool   `arg:"-r,--regex" default:"false" help:"Flag that enables regex search"`
	FileType string `arg:"-t,--type" default:".py" help:"Filetypes to search (i.e. '.py')"`

	// If match is not inside a testcase -> search for usage of containing method.
	// How many levels of search to perform (trying to find a TC usage) before giving up
	Distance int `arg:"-d,--dist" default:"6" help:"Levels of recursive search"`

	LogFile string `arg:"-l,--log" default:"search.log" help:"Log filename"`
	Pattern string `arg:"positional,required" help:"Pattern to search for"`
	Dir     string `arg:"positional,required" help:"Directory to search in"`
}

const (
	MethodPatternStr = `def\s*(?P<name>.*?)\(`
)

var (
	// https://en.wikipedia.org/wiki/ANSI_escape_code
	// 8-bit color table shows where the numbers below come from
	ImportantStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("3")) // gold

	FilenameStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("6")) // green-blue

	MatchStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("5")) // magenta

	InfoStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")) // bright blue

	WarningStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("11")) // bright yellow

	ErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("9")) // bright red
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
	colEnd       int
	matchLineTxt string
	usedInMethod string
	isMethodDecl bool
}

func (r SearchResult) String() string {
	// Apply styling to line number and match
	out := MatchStyle.Render(fmt.Sprintf("%d: ", r.line))
	out += r.matchLineTxt[:r.col]
	out += MatchStyle.Render(r.matchLineTxt[r.col:r.colEnd])
	out += r.matchLineTxt[r.colEnd:]

	// return fmt.Sprintf("%d: %s", r.line, r.matchLineTxt)
	return out
}

type FileResult struct {
	file    string
	matches []SearchResult
	isTc    bool
	tcId    string
}

func (r FileResult) String() string {
	out := FilenameStyle.Render(fmt.Sprintf("%s", r.file))
	out += "\n"

	for _, match := range r.matches {
		out += fmt.Sprintf("\t%s\n\n", match)
	}

	return out
}

func (r *FileResult) RemoveMatch(idx int) {
	r.matches[idx] = r.matches[len(r.matches)-1]
	r.matches = r.matches[:len(r.matches)-1]
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
		errorTxt := fmt.Sprintf(
			"Expected either a regexp.Regexp or a string but got neither: %v",
			pattern,
		)
		log.Fatalf(ErrorStyle.Render(errorTxt))
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

		// The result indexes have to be absolute and not only relative to the current text
		results = append(results, []int{idx + currentIdx, idx + subLen + currentIdx})

		currentTxt = currentTxt[idx+subLen:]
		currentIdx += idx + subLen
	}

	return results
}

func MatchMethodName(s string) string {
	methodPattern, err := regexp.Compile(MethodPatternStr)
	if err != nil {
		errorTxt := fmt.Sprintf("Couldn't compile method declaration regex: %v", err)
		log.Fatalf(ErrorStyle.Render(errorTxt))
	}
	nameIdx := methodPattern.SubexpIndex("name")

	match := methodPattern.FindStringSubmatch(s)
	if match == nil {
		return ""
	}
	return match[nameIdx]

}

func GetContainingMethod(pretext string) string {
	// Search for closest method declaration above usage -> this is used to
	// continue searching for usages incase the match does not occur inside a test case file
	usedInMethod := ""
	lines := strings.Split(pretext, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		textLine := lines[i]

		// Don't consider empty lines or lines containing only whitespace
		if len(strings.TrimSpace(textLine)) == 0 {
			continue
		}

		methodName := MatchMethodName(textLine)
		if methodName == "" {
			continue
		}

		usedInMethod = methodName
		break
	}

	return usedInMethod
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
		errorTxt := fmt.Sprintf(
			"Couldn't find left newline or right newline for match %s: %d %d",
			text[start:end],
			leftNewLineIdx,
			rightNewLineIdx,
		)
		log.Fatalf(ErrorStyle.Render(errorTxt))
	}

	// +1 is needed to ignore the preceding newline
	matchTxt := (pretext[leftNewLineIdx+1:] +
		text[start:end] +
		posttext[:rightNewLineIdx])
	col := start - leftNewLineIdx - 1
	colEnd := end - leftNewLineIdx - 1

	usedInMethod := ""
	isMethodDecl := MatchMethodName(matchTxt) != ""
	// Only extract containing method if we don't have a method declaration in matchTxt
	if !isMethodDecl {
		usedInMethod = GetContainingMethod(pretext)
	}

	return SearchResult{
		line:         line,
		col:          col,
		colEnd:       colEnd,
		matchLineTxt: matchTxt,
		usedInMethod: usedInMethod,
		isMethodDecl: isMethodDecl,
	}
}

// NOTE: this is very project specific
func ProcessTc(text, path string) (isTc bool, tcId string) {
	tcPathPattern, err := regexp.Compile(`test_cases/.*?/test_.*?\.py`)
	if err != nil {
		errorTxt := fmt.Sprintf("Couldn't compile TC path pattern: %v", err)
		log.Fatalf(ErrorStyle.Render(errorTxt))
	}
	isTc = tcPathPattern.FindString(path) != ""

	tcId = ""
	if isTc {
		tcIdPattern, err := regexp.Compile(`Polarion ID: (?P<id>[a-zA-Z0-9]+-\d+)`)
		if err != nil {
			errorTxt := fmt.Sprintf("Couldn't compile TC ID pattern: %v", err)
			log.Fatalf(ErrorStyle.Render(errorTxt))
		}
		idIndex := tcIdPattern.SubexpIndex("id")
		match := tcIdPattern.FindStringSubmatch(text)
		if match != nil {
			tcId = match[idIndex]
		} else {
			errorTxt := fmt.Sprintf("ERROR: Couldn't find ID for TC %s", path)
			log.Printf(ErrorStyle.Render(errorTxt))
		}
	}

	return isTc, tcId
}

func SearchFile[T SearchTerm](path string, pattern T) *FileResult {
	results := []SearchResult{}

	data, err := os.ReadFile(path)
	if err != nil {
		errorTxt := fmt.Sprintf("ERROR: Couldn't read file %s: %v", path, err)
		log.Printf(ErrorStyle.Render(errorTxt))
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
		errorTxt := fmt.Sprintf("Couldn't get list of files for dir %s: %v", dir, err)
		log.Fatalf(errorTxt)
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

type TestCasesMap map[string]bool

func (m TestCasesMap) String() string {
	out := "["
	for k, _ := range m {
		out += fmt.Sprintf("%s, ", k)
	}
	out += "]"
	return out
}

func UpdateMap(v1, v2 map[string]bool) map[string]bool {
	for k, _ := range v2 {
		v1[k] = true
	}
	return v1
}

func SearchForUsagesInTc[T SearchTerm](
	dir, fileType string,
	searchPattern T,
	degreesOfSeparation int,
) TestCasesMap {
	testCases := TestCasesMap{}
	nonTcMatches := []FileResult{}

	/*
	   For recursive search make sure that only one method declaration is found for a
	   usedInMethod search. This way we are sure to only find TCs related to the correct method.
	   In some cases the usedInMethod will have a generic name like `connect` which might result
	   in a lot of result that are not relevant to our search.
	*/

	if degreesOfSeparation <= 0 {
		return TestCasesMap{}
	}

	methodDeclarationNum := 0

	results := SearchInRepo(dir, fileType, searchPattern)
	for _, result := range results {
		for idx, match := range result.matches {
			if !match.isMethodDecl {
				continue
			}

			methodDeclarationNum++
			// If we find search results that result in multiple method declarations
			// we can't reliably use the result from the search cause our search term
			// is not unique -> return no results
			if methodDeclarationNum > 1 {
				errorTxt := fmt.Sprint(
					"Found multiple method declaration for this search pattern. Discarding TC results: ",
					searchPattern,
				)
				log.Println(ErrorStyle.Render(errorTxt))
				return TestCasesMap{}
			}

			// Remove any declaration match from results so that we don't
			// consider it as a nonTcMatch
			result.RemoveMatch(idx)
		}

		if len(result.matches) == 0 {
			continue
		}

		fmt.Println(result)
		if result.tcId != "" {
			testCases[result.tcId] = true
		} else {
			nonTcMatches = append(nonTcMatches, result)
		}
	}

	searched := map[string]bool{}
	log.Printf(
		"%s\n%s\n%s\n%s",
		InfoStyle.Render("Non TC results"),
		InfoStyle.Render("----------------"),
		nonTcMatches,
		InfoStyle.Render("----------------"),
	)
	for _, fileResult := range nonTcMatches {
		for _, searchResult := range fileResult.matches {
			if searchResult.usedInMethod == "" {
				errorTxt := fmt.Sprintf(
					"No containing method found for match:\n%s\n%d: %s",
					searchResult.file,
					searchResult.line,
					searchResult.matchLineTxt,
				)
				log.Println(ErrorStyle.Render(errorTxt))
				continue
			}

			// Add word boudary to make sure we search for exact word matches
			newSearchTerm := fmt.Sprintf("\\b%s\\b", searchResult.usedInMethod)
			newSearchPattern, err := regexp.Compile(newSearchTerm)
			if err != nil {
				errorTxt := fmt.Sprint("Couldn't compile method pattern regexp for: ", newSearchTerm)
				log.Fatalln(ErrorStyle.Render(errorTxt))
			}

			if _, ok := searched[newSearchTerm]; ok {
				warningTxt := fmt.Sprint("Containing method already searched: ", searchResult.usedInMethod)
				log.Println(WarningStyle.Render(warningTxt))
				continue
			}

			infoTxt := fmt.Sprintf("Extending search for %v by %s", searchPattern, newSearchPattern)
			log.Printf(InfoStyle.Render(infoTxt))

			// We just track the search term and not the regexp pattern for simplicity
			searched[newSearchTerm] = true
			foundTcs := SearchForUsagesInTc(dir, fileType, newSearchPattern, degreesOfSeparation-1)
			testCases = UpdateMap(testCases, foundTcs)
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
	arg.MustParse(&args)

	setupLogger(args.LogFile)

	// pattern := `\.outputHeater\.set_disconnected`
	var (
		searchPatternRegex *regexp.Regexp
		err                error
	)
	if args.UseRegex {
		searchPatternRegex, err = regexp.Compile(args.Pattern)
		if err != nil {
			errorTxt := fmt.Sprintf("Couldn't compile search pattern %s: %v", args.Pattern, err)
			log.Fatalf(ErrorStyle.Render(errorTxt))
		}
	}

	start := time.Now()

	searchTxt := args.Pattern
	if args.UseRegex {
		searchTxt = searchPatternRegex.String()
	}

	log.Printf(ImportantStyle.Render(fmt.Sprintf(
		"Searching for: R(%v) |%s| (%s) %s D(%d)",
		args.UseRegex,
		searchTxt,
		args.FileType,
		args.Dir,
		args.Distance,
	)))

	var testCases TestCasesMap
	if args.UseRegex {
		testCases = SearchForUsagesInTc(args.Dir, args.FileType, searchPatternRegex, args.Distance)
	} else {
		testCases = SearchForUsagesInTc(args.Dir, args.FileType, args.Pattern, args.Distance)
	}

	log.Println()

	infoTxt := fmt.Sprintf("Used in test cases (%d):", len(testCases))
	log.Printf(InfoStyle.Render(infoTxt))
	log.Println(InfoStyle.Render(testCases.String()))

	log.Println("Elapsed time", time.Since(start).Seconds())
}
