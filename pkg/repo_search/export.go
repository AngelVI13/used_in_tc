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

func CreateXml(template, outPath, searchPattern string, testCases TestCasesMap) string {
	protocols := ""
	for tc := range testCases {
		protocols += fmt.Sprintf(ProtocolTemplate, tc)
		protocols += "\n"
	}

	template = strings.Replace(template, VlReplaceResults, protocols, 1)
	template = strings.Replace(
		template,
		VlReplaceSearchPattern,
		fmt.Sprintf(SearchPatternTemplate, searchPattern),
		1,
	)

	outFilename := AddTimestampToFilename(outPath, ".xml")
	err := os.WriteFile(outFilename, []byte(template), 0666)
	if err != nil {
		errorTxt := fmt.Sprintf("ERROR: Couldn't write to file %s: %v", outFilename, err)
		log.Fatal(ErrorStyle.Render(errorTxt))
	}

	return outFilename
}
