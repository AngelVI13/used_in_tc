package repo_search

import "fmt"

type FileResult struct {
	file    string
	matches []SearchResult
	isTc    bool
	tcId    string
}

func (r FileResult) String() string {
	out := FilenameStyle.Render(fmt.Sprintf("%s", r.file))
	out += "\n"

	for _, match := range r.matches {
		out += fmt.Sprintf("\t%s\n\n", match)
	}

	return out
}

func (r *FileResult) RemoveMatch(idx int) {
	r.matches[idx] = r.matches[len(r.matches)-1]
	r.matches = r.matches[:len(r.matches)-1]
}
