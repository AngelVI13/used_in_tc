package repo_search

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const MethodPatternStr = `def\s*(?P<name>.*?)\(`

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
		log.Fatal(ErrorStyle.Render(errorTxt))
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
		log.Fatal(ErrorStyle.Render(errorTxt))
	}
	nameIdx := methodPattern.SubexpIndex("name")

	match := methodPattern.FindStringSubmatch(s)
	if match == nil {
		return ""
	}
	return match[nameIdx]

}

func GetContainingMethod(pretext string) string {
	// Pre compile some method testing patters
	testMethodPattern, err := regexp.Compile(`^test_(\d+)_`)
	if err != nil {
		errorTxt := fmt.Sprintf("Couldn't compile test method declaration regex: %v", err)
		log.Fatal(ErrorStyle.Render(errorTxt))
	}

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

		// Do not use any test case official method as a containing method
		if testMethodPattern.MatchString(methodName) {
			break
		}

		// Do not use __init__ as containg method cause we can't search for it
		if methodName == "__init__" {
			break
		}

		usedInMethod = methodName
		break
	}

	return usedInMethod
}

func GetFilesFromDir(root string, fileType string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		// TODO: add blacklist to script args
		if !info.IsDir() && strings.HasSuffix(path, fileType) && !strings.HasSuffix(path, "_pb2"+fileType) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func AddTimestampToFilename(filename, ext string) string {
	currentTime := time.Now()
	timeStamp := currentTime.Format("2006_01_02__15_04_05")

	name := strings.TrimSuffix(filename, ext)
	newFilename := fmt.Sprintf("%s_%s%s", name, timeStamp, ext)

	return newFilename
}
