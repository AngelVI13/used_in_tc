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

func FindApprovedTcs(
	testCases TestCasesMap,
	workItems WorkItems,
) (approved []TestCase, notApproved []TestCase) {
	approved = []TestCase{}
	notApproved = []TestCase{}

	for id, tc := range testCases {
		// If workItems is not provided -> assume all are approved
		if workItems == nil {
			approved = append(approved, tc)
			continue
		}

		info, ok := workItems[id]
		if !ok {
			log.Println("Cant find TC ID in workItems: ", id)
			notApproved = append(notApproved, tc)
			continue
		}

		if info.Status == "Approved" {
			approved = append(approved, tc)
		} else {
			notApproved = append(notApproved, tc)
		}
	}
	return approved, notApproved
}

func CreateProtocolXml(testCases TestCasesMap, wiExportPath string) string {
	protocols := ""
	tcBySetup := GetTcBySetup(testCases)

	var workItems WorkItems
	if wiExportPath != "" {
		workItems = GetWorkItemsFromPolarionExport(wiExportPath)
	}

	// if we have workitems information, Include Approved/NotApproved info to protocols txt
	approvedTxt := ""
	if workItems != nil {
		approvedTxt = "Approved "
	}

	for setup, tests := range tcBySetup {
		protocols += fmt.Sprintf("\n<!-- %s %s-->\n", setup, approvedTxt)

		approved, notApproved := FindApprovedTcs(tests, workItems)

		// sort test cases for each setup by duration
		sort.Slice(approved, func(i, j int) bool {
			return approved[i].DurationSec() < approved[j].DurationSec()
		})

		// sort test cases for each setup by duration
		sort.Slice(notApproved, func(i, j int) bool {
			return notApproved[i].DurationSec() < notApproved[j].DurationSec()
		})

		for _, tc := range approved {
			protocols += tc.Protocol()
			protocols += "\n"
		}

		if len(notApproved) > 0 {
			protocols += fmt.Sprintf("\n<!-- %s - NOT APPROVED -->\n", setup)
			for _, tc := range notApproved {
				protocols += tc.Protocol()
				protocols += "\n"
			}
		}
	}
	return protocols
}

func CreateXml(template, outPath, searchPattern, protocols string) string {
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
