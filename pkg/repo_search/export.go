package repo_search

import (
	"fmt"
	"log"
	"os"
	"strings"
)

const (
	VlReplaceTxt     = "<!-- REPLACE -->"
	ProtocolTemplate = "<protocol project-id=\"4008APackage2\" id=\"%s\"></protocol>"
)

func CreateXml(templatePath, outPath string, testCases TestCasesMap) {
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

	text = strings.Replace(text, VlReplaceTxt, protocols, 1)

	err = os.WriteFile(outPath, []byte(text), 0666)
	if err != nil {
		errorTxt := fmt.Sprintf("ERROR: Couldn't write to file %s: %v", outPath, err)
		log.Fatal(ErrorStyle.Render(errorTxt))
	}
}
