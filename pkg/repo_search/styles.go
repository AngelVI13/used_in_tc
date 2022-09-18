package repo_search

import "github.com/charmbracelet/lipgloss"

var (
	// https://en.wikipedia.org/wiki/ANSI_escape_code
	// 8-bit color table shows where the numbers below come from
	ImportantStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("3")) // gold

	FilenameStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("6")) // green-blue

	MatchStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("5")) // magenta

	InfoStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")) // bright blue

	WarningStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("11")) // bright yellow

	ErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("9")) // bright red
)
