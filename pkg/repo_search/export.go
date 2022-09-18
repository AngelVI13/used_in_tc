package repo_search

import (
	"fmt"
	"log"
	"os"
	"strings"
)

const (
	VlReplaceResults       = "<!-- REPLACE WITH RESULTS-->"
	VlReplaceSearchPattern = "<!-- REPLACE WITH SEARCH PATTERN-->"
	ProtocolTemplate       = "<protocol project-id=\"4008APackage2\" id=\"%s\"></protocol>"
	SearchPatternTemplate  = "<!-- SEARCH: %s -->"
)

func CreateXml(templatePath, outPath, searchPattern string, testCases TestCasesMap) string {
	data, err := os.ReadFile(templatePath)
	if err != nil {
		errorTxt := fmt.Sprintf("ERROR: Couldn't read file %s: %v", templatePath, err)
		log.Fatal(ErrorStyle.Render(errorTxt))
	}
	text := string(data)

	protocols := ""
	for tc := range testCases {
		protocols += fmt.Sprintf(ProtocolTemplate, tc)
		protocols += "\n"
	}

	text = strings.Replace(text, VlReplaceResults, protocols, 1)
	text = strings.Replace(
		text,
		VlReplaceSearchPattern,
		fmt.Sprintf(SearchPatternTemplate, searchPattern),
		1,
	)

	outFilename := AddTimestampToFilename(outPath, ".xml")
	err = os.WriteFile(outFilename, []byte(text), 0666)
	if err != nil {
		errorTxt := fmt.Sprintf("ERROR: Couldn't write to file %s: %v", outFilename, err)
		log.Fatal(ErrorStyle.Render(errorTxt))
	}

	return outFilename
}
