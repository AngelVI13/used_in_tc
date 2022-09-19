package main

import (
	_ "embed"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/AngelVI13/used_in_tc/pkg/repo_search"
	"github.com/alexflint/go-arg"
)

//go:embed vl_template.xml
var templateXml string

var args struct {
	UseRegex bool   `arg:"-r,--regex" default:"false" help:"Flag that enables regex search"`
	FileType string `arg:"-t,--type" default:".py" help:"Filetypes to search (i.e. '.py')"`

	// If match is not inside a testcase -> search for usage of containing method.
	// How many levels of search to perform (trying to find a TC usage) before giving up
	Distance int `arg:"-d,--dist" default:"6" help:"Levels of recursive search"`

	LogFile string `arg:"-l,--log" default:"search.log" help:"Log filename"`
	OutFile string `arg:"-o,--out" default:"search_tc.xml" help:"Output xml filename"`

	Pattern string `arg:"positional,required" help:"Pattern to search for"`
	Dir     string `arg:"positional,required" help:"Directory to search in"`
}

func setupLogger(filename string) {
	// Delete old log file
	os.Remove(filename)
	// Set up logging to stdout and file
	logFile, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
}

func main() {
	arg.MustParse(&args)

	setupLogger(args.LogFile)

	// pattern := `\.outputHeater\.set_disconnected`
	var (
		searchPatternRegex *regexp.Regexp
		err                error
	)
	if args.UseRegex {
		searchPatternRegex, err = regexp.Compile(args.Pattern)
		if err != nil {
			errorTxt := fmt.Sprintf("Couldn't compile search pattern %s: %v", args.Pattern, err)
			log.Fatalf(repo_search.ErrorStyle.Render(errorTxt))
		}
	}

	start := time.Now()

	searchTxt := args.Pattern
	if args.UseRegex {
		searchTxt = searchPatternRegex.String()
	}

	log.Printf(repo_search.ImportantStyle.Render(fmt.Sprintf(
		"Searching for: R(%v) |%s| (%s) %s D(%d)",
		args.UseRegex,
		searchTxt,
		args.FileType,
		args.Dir,
		args.Distance,
	)))

	var testCases repo_search.TestCasesMap
	// Initialize global var holding already searched data
	repo_search.AlreadySearched = map[string]bool{}

	if args.UseRegex {
		testCases = repo_search.SearchForUsagesInTc(args.Dir, args.FileType, searchPatternRegex, args.Distance)
	} else {
		testCases = repo_search.SearchForUsagesInTc(args.Dir, args.FileType, args.Pattern, args.Distance)
	}

	outFilename := repo_search.CreateXml(templateXml, args.OutFile, searchTxt, testCases)

	log.Println()

	searchInfoTxt := fmt.Sprintf("Search results for: %s", searchTxt)
	log.Println(repo_search.ImportantStyle.Render(searchInfoTxt))

	infoTxt := fmt.Sprintf("Used in test cases (%d):", len(testCases))
	log.Println(repo_search.InfoStyle.Render(infoTxt))
	log.Println(repo_search.InfoStyle.Render(testCases.String()))

	infoTxt = fmt.Sprintf("TC Xml created successfully: %s", outFilename)
	log.Println(repo_search.ImportantStyle.Render(infoTxt))

	log.Println("Elapsed time", time.Since(start).Seconds())
}
