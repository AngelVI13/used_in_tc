package repo_search

import (
	"fmt"
	"log"
	"os"
	"strings"
)

const (
	VlReplaceResults        = "<!-- REPLACE WITH RESULTS-->"
	VlReplaceSearchPattern  = "<!-- REPLACE WITH SEARCH PATTERN-->"
	ProtocolTemplate        = "<protocol project-id=\"4008APackage2\" id=\"%s\">\n\t%s\n</protocol>"
	ScriptReferenceTemplate = "<test-script-reference>%s</test-script-reference>"
	ScriptUrlTemplate       = "http://desw-svn1.schweinfurt.germany.fresenius.de/svn/4008A/apps/trunk/test_automation/%s"
	SearchPatternTemplate   = "<!-- SEARCH: %s -->"
)

func CreateXml(template, outPath, searchPattern string, testCases TestCasesMap) string {
	protocols := ""
	for tc, path := range testCases {
		// Format test case path to expected test script reference path url
		_, after, found := strings.Cut(path, "test_cases")
		if !found {
			errorTxt := fmt.Sprintf("Couldn't find '/test_cases/' in path: %s", path)
			log.Fatalf(ErrorStyle.Render(errorTxt))
		}
		tcPath := "test_cases" + after
		tcPath = strings.ReplaceAll(tcPath, "\\", "/")

		testScriptUrl := fmt.Sprintf(ScriptUrlTemplate, tcPath)
		testScriptReference := fmt.Sprintf(ScriptReferenceTemplate, testScriptUrl)
		protocols += fmt.Sprintf(ProtocolTemplate, tc, testScriptReference)
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
