package repo_search

import (
	"fmt"
	"log"
	"os"
	"sort"
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

		if out[setup] == nil {
			out[setup] = TestCasesMap{}
		}
		out[setup][id] = tc
	}

	return out
}

func CreateXml(template, outPath, searchPattern string, testCases TestCasesMap) string {
	protocols := ""
	tcBySetup := GetTcBySetup(testCases)

	for setup, tests := range tcBySetup {
		protocols += fmt.Sprintf("\n<!-- %s -->\n", setup)

		testCasesSlice := []TestCase{}
		for _, tc := range tests {
			testCasesSlice = append(testCasesSlice, tc)
		}

		// sort test cases for each setup by duration
		sort.Slice(testCasesSlice, func(i, j int) bool {
			return testCasesSlice[i].DurationSec() < testCasesSlice[j].DurationSec()
		})

		for _, tc := range testCasesSlice {
			protocols += tc.Protocol()
			protocols += "\n"
		}
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
