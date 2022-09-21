package repo_search

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
)

var AlreadySearched map[string]bool

type SearchTerm interface {
	string | *regexp.Regexp
}

const (
	ProtocolTemplate        = "<protocol project-id=\"4008APackage2\" id=\"%s\"> <!-- %s -->\n\t%s\n</protocol>"
	ScriptReferenceTemplate = "<test-script-reference>%s</test-script-reference>"
	ScriptUrlTemplate       = "http://desw-svn1.schweinfurt.germany.fresenius.de/svn/4008A/apps/trunk/test_automation/%s"
)

type TestCase struct {
	path string
	info TestCaseInfo
}

func (t *TestCase) Protocol() string {
	// Format test case path to expected test script reference path url
	_, after, found := strings.Cut(t.path, "test_cases")
	if !found {
		errorTxt := fmt.Sprintf("Couldn't find '/test_cases/' in path: %s", t.path)
		log.Fatalf(ErrorStyle.Render(errorTxt))
	}
	tcPath := "test_cases" + after
	tcPath = strings.ReplaceAll(tcPath, "\\", "/")

	testScriptUrl := fmt.Sprintf(ScriptUrlTemplate, tcPath)
	testScriptReference := fmt.Sprintf(ScriptReferenceTemplate, testScriptUrl)

	tcInfo := fmt.Sprintf("Duration: %s; Setup: %s", t.info.estimate, t.info.setup)
	out := fmt.Sprintf(ProtocolTemplate, t.info.id, tcInfo, testScriptReference)

	return out
}

// TestCasesMap Key is TC ID
type TestCasesMap map[string]TestCase

func (m TestCasesMap) String() string {
	out := "["
	for k, _ := range m {
		out += fmt.Sprintf("%s, ", k)
	}
	out += "]"
	return out
}

func UpdateMap(v1, v2 TestCasesMap) TestCasesMap {
	for k, v := range v2 {
		v1[k] = v
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
				log.Println(WarningStyle.Render(errorTxt))
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
		if result.tcInfo.id != "" {
			testCases[result.tcInfo.id] = TestCase{
				path: result.file,
				info: result.tcInfo,
			}
		} else {
			nonTcMatches = append(nonTcMatches, result)
		}
	}

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
				log.Println(WarningStyle.Render(errorTxt))
				continue
			}

			// Add word boudary to make sure we search for exact word matches
			newSearchTerm := fmt.Sprintf("\\b%s\\b", searchResult.usedInMethod)
			newSearchPattern, err := regexp.Compile(newSearchTerm)
			if err != nil {
				errorTxt := fmt.Sprint("Couldn't compile method pattern regexp for: ", newSearchTerm)
				log.Fatalln(ErrorStyle.Render(errorTxt))
			}

			if _, ok := AlreadySearched[newSearchTerm]; ok {
				/* NOTE: this spams too much
				warningTxt := fmt.Sprint("Containing method already searched: ", searchResult.usedInMethod)
				log.Println(WarningStyle.Render(warningTxt))
				*/
				continue
			}

			infoTxt := fmt.Sprintf("Extending search for %v by %s", searchPattern, newSearchPattern)
			log.Print(InfoStyle.Render(infoTxt))

			// We just track the search term and not the regexp pattern for simplicity
			AlreadySearched[newSearchTerm] = true
			foundTcs := SearchForUsagesInTc(dir, fileType, newSearchPattern, degreesOfSeparation-1)
			testCases = UpdateMap(testCases, foundTcs)
		}
	}

	return testCases
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

func SearchFile[T SearchTerm](path string, pattern T) *FileResult {
	results := []SearchResult{}

	data, err := os.ReadFile(path)
	if err != nil {
		errorTxt := fmt.Sprintf("ERROR: Couldn't read file %s: %v", path, err)
		log.Print(ErrorStyle.Render(errorTxt))
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

	isTc := IsPathTc(path)
	var tcInfo TestCaseInfo
	if isTc {
		tcInfo = ProcessTc(text, path)
	}
	return &FileResult{
		file:    path,
		matches: results,
		isTc:    isTc,
		tcInfo:  tcInfo,
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
