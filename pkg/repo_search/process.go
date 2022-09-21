package repo_search

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
)

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
		log.Fatal(ErrorStyle.Render(errorTxt))
	}

	// +1 is needed to ignore the preceding newline
	matchTxt := (pretext[leftNewLineIdx+1:] +
		text[start:end] +
		posttext[:rightNewLineIdx])
	col := start - leftNewLineIdx - 1
	colEnd := end - leftNewLineIdx - 1

	usedInMethod := ""
	isMethodDecl := MatchContainerName(MethodContainer, matchTxt) != ""
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

func IsPathTc(path string) bool {
	sep := string(os.PathSeparator)
	if sep != "/" {
		sep = `\\` // on WIN make sure to escape the backslash
	}
	testCasePathPattern := fmt.Sprintf(`test_cases%s.*?%stest_.*?\.py`, sep, sep)

	tcPathPattern, err := regexp.Compile(testCasePathPattern)
	if err != nil {
		errorTxt := fmt.Sprintf("Couldn't compile TC path pattern: %v", err)
		log.Fatal(ErrorStyle.Render(errorTxt))
	}
	isTc := tcPathPattern.FindString(path) != ""
	return isTc
}

type TestCaseInfo struct {
	estimate string
	setup    string
	id       string
}

func ExtractTcElement(text, pattern, resultId string) string {
	element := ""
	elementPattern, err := regexp.Compile(pattern)
	if err != nil {
		errorTxt := fmt.Sprintf("Couldn't compile TC %s pattern: %v", resultId, err)
		log.Fatal(ErrorStyle.Render(errorTxt))
	}
	resultIdIndex := elementPattern.SubexpIndex(resultId)
	match := elementPattern.FindStringSubmatch(text)
	if match != nil {
		element = match[resultIdIndex]
	}
	return element
}

// NOTE: this is very project specific
func ProcessTc(text, path string) TestCaseInfo {
	tcIdPattern := `Polarion ID: (?P<id>[a-zA-Z0-9]+-\d+)`
	tcId := ExtractTcElement(text, tcIdPattern, "id")
	if tcId == "" {
		// TODO: Should any of these errors return an empty object?
		errorTxt := fmt.Sprintf("ERROR: Couldn't find %s for TC %s", tcId, path)
		log.Print(ErrorStyle.Render(errorTxt))
	}

	setupPattern := `Setup: (?P<setup>.*?)\n`
	setup := ExtractTcElement(text, setupPattern, "setup")
	if setup == "" {
		errorTxt := fmt.Sprintf("ERROR: Couldn't find %s for TC %s", setup, path)
		log.Print(ErrorStyle.Render(errorTxt))
	}

	estimatePattern := `Initial estimate: \b(?P<estimate>[0-9:]+)\b`
	estimate := ExtractTcElement(text, estimatePattern, "estimate")
	if estimate == "" {
		errorTxt := fmt.Sprintf("ERROR: Couldn't find %s for TC %s", estimate, path)
		log.Print(ErrorStyle.Render(errorTxt))
	}

	return TestCaseInfo{
		estimate: estimate,
		setup:    setup,
		id:       tcId,
	}
}
