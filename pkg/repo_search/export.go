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
	SearchPatternTemplate  = "<!-- SEARCH: %s -->"
)

func GetTcBySetup(testCases TestCasesMap) map[string]TestCasesMap {
	out := map[string]TestCasesMap{}

	for id, tc := range testCases {
		setup := tc.info.setup
		setup = strings.TrimSpace(setup)
		setup = strings.Split(setup, " ")[0]
		setup = strings.ToLower(setup)

		out[setup][id] = tc
	}

	return out
}

func CreateXml(template, outPath, searchPattern string, testCases TestCasesMap) string {
	protocols := ""
	// TODO: sort TCs by setup and then by duration for each setup
	for _, tc := range testCases {
		protocols += tc.Protocol()
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
