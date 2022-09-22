package repo_search

import (
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"strings"
)

type StatusProps struct {
	XMLName xml.Name `xml:"status"`
	Name    string   `xml:"name,attr"`
}

type FieldsProp struct {
	XMLName xml.Name     `xml:"fields"`
	Id      string       `xml:"id"`
	Status  *StatusProps `xml:"status"`
	Title   string       `xml:"title"`
}

type FieldOptions struct {
	XMLName xml.Name `xml:"option"`
	Name    string   `xml:"name,attr"`
}

type CustomProps struct {
	XMLName xml.Name        `xml:"field"`
	Id      string          `xml:"id,attr"`
	Options []*FieldOptions `xml:"multi-enum>option"`
}

type WorkItemXml struct {
	XMLName      xml.Name       `xml:"workItem"`
	Fields       *FieldsProp    `xml:"fields"`
	CustomFields []*CustomProps `xml:"customFields>field"`
}

type WorkItemsXml struct {
	XMLName xml.Name       `xml:"workItems"`
	Items   []*WorkItemXml `xml:"workItem"`
}

type WorkItem struct {
	Id                    string
	Title                 string
	Status                string
	RiskReductionMeasures []string
}

func (item *WorkItem) Valid() bool {
	return item.Id != "" && item.Title != "" && item.Status != ""
}

func (item WorkItem) String() string {
	return fmt.Sprintf(
		"%s,%s,%s,%s",
		item.Id,
		item.Title,
		item.Status,
		strings.Join(item.RiskReductionMeasures, ", "),
	)

}

type WorkItems map[string]*WorkItem

func (w WorkItems) String() string {
	out := ""
	count := 1
	for _, item := range w {
		//out += fmt.Sprintf("%d. %s\n", count, item)
		out += fmt.Sprintf("%s\n", item)
		count += 1
	}

	return out
}

func GetRiskReductionMeasures(item *WorkItemXml) []string {
	rrm := []string{}
	for _, prop := range item.CustomFields {
		if prop.Id != "riskreductionmeasure" {
			continue
		}

		for _, measure := range prop.Options {
			rrm = append(rrm, measure.Name)
		}
	}
	return rrm
}

func ParseWorkItems(workItems *WorkItemsXml) WorkItems {
	parsedItems := WorkItems{}
	for _, item := range workItems.Items {
		parsedItems[item.Fields.Id] = &WorkItem{
			Id:                    item.Fields.Id,
			Title:                 item.Fields.Title,
			Status:                item.Fields.Status.Name,
			RiskReductionMeasures: GetRiskReductionMeasures(item),
		}
	}

	return parsedItems
}

func GetWorkItemsFromPolarionExport(path string) WorkItems {
	masterData, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Failed to read polarion file `%s`: %+v", path, err)
	}

	var workItemsXml WorkItemsXml
	err = xml.Unmarshal(masterData, &workItemsXml)
	if err != nil {
		log.Fatalf("Failed to unmarshal polarion file `%s`: %+v", path, err)
	}

	// Convert from xml structure of work items to a more usable structure
	workItems := ParseWorkItems(&workItemsXml)

	return workItems
}
