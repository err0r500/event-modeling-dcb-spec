package tui

import "github.com/charmbracelet/lipgloss"

var (
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4444")).
			Bold(true)

	// Tree styles
	treeContextStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#7D56F4"))

	treeChapterStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00AAFF"))

	treeSliceStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AAAAAA"))

	treeCursorStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("57")).
			Foreground(lipgloss.Color("229"))

	treeExpandedIcon   = "▼ "
	treeCollapsedIcon  = "▶ "
	treeLeafIcon       = "  "
	treeIndent         = "  "
)
