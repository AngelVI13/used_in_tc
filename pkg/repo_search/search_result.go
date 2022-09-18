package repo_search

import "fmt"

type SearchResult struct {
	file         string
	line         int
	col          int
	colEnd       int
	matchLineTxt string
	usedInMethod string
	isMethodDecl bool
}

func (r SearchResult) String() string {
	// Apply styling to line number and match
	out := MatchStyle.Render(fmt.Sprintf("%d: ", r.line))
	out += r.matchLineTxt[:r.col]
	out += MatchStyle.Render(r.matchLineTxt[r.col:r.colEnd])
	out += r.matchLineTxt[r.colEnd:]

	// return fmt.Sprintf("%d: %s", r.line, r.matchLineTxt)
	return out
}
