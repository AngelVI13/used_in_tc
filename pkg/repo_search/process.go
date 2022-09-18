package repo_search

import (
	"fmt"
	"log"
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

// NOTE: this is very project specific
func ProcessTc(text, path string) (isTc bool, tcId string) {
	tcPathPattern, err := regexp.Compile(`test_cases/.*?/test_.*?\.py`)
	if err != nil {
		errorTxt := fmt.Sprintf("Couldn't compile TC path pattern: %v", err)
		log.Fatal(ErrorStyle.Render(errorTxt))
	}
	isTc = tcPathPattern.FindString(path) != ""

	tcId = ""
	if isTc {
		tcIdPattern, err := regexp.Compile(`Polarion ID: (?P<id>[a-zA-Z0-9]+-\d+)`)
		if err != nil {
			errorTxt := fmt.Sprintf("Couldn't compile TC ID pattern: %v", err)
			log.Fatal(ErrorStyle.Render(errorTxt))
		}
		idIndex := tcIdPattern.SubexpIndex("id")
		match := tcIdPattern.FindStringSubmatch(text)
		if match != nil {
			tcId = match[idIndex]
		} else {
			errorTxt := fmt.Sprintf("ERROR: Couldn't find ID for TC %s", path)
			log.Print(ErrorStyle.Render(errorTxt))
		}
	}

	return isTc, tcId
}
