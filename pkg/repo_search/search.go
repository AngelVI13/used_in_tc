package repo_search

import (
	"fmt"
	"log"
	"os"
	"regexp"
)

type SearchTerm interface {
	string | *regexp.Regexp
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
			log.Print(InfoStyle.Render(infoTxt))

			// We just track the search term and not the regexp pattern for simplicity
			searched[newSearchTerm] = true
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
