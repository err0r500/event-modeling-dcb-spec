package render

import (
	"strings"
)

// Box drawing characters
const (
	TopLeft     = "┌"
	TopRight    = "┐"
	BottomLeft  = "└"
	BottomRight = "┘"
	Horizontal  = "─"
	Vertical    = "│"
	LeftT       = "├"
	RightT      = "┤"
	TopT        = "┬"
	BottomT     = "┴"
	Cross       = "┼"
)

// Box represents an ASCII box with content
type Box struct {
	Width int
	Lines []string
}

// NewBox creates a new box with specified width
func NewBox(width int) *Box {
	return &Box{Width: width}
}

// AddLine adds a line of content to the box
func (b *Box) AddLine(content string) {
	b.Lines = append(b.Lines, content)
}

// AddSection adds a section divider
func (b *Box) AddSection() {
	b.Lines = append(b.Lines, "---SECTION---")
}

// Render outputs the box as a string
func (b *Box) Render() string {
	var sb strings.Builder

	innerWidth := b.Width - 2 // Account for borders

	// Top border
	sb.WriteString(TopLeft)
	sb.WriteString(strings.Repeat(Horizontal, innerWidth))
	sb.WriteString(TopRight)
	sb.WriteString("\n")

	// Content lines
	for _, line := range b.Lines {
		if line == "---SECTION---" {
			// Section divider
			sb.WriteString(LeftT)
			sb.WriteString(strings.Repeat(Horizontal, innerWidth))
			sb.WriteString(RightT)
			sb.WriteString("\n")
		} else {
			// Content line
			sb.WriteString(Vertical)
			sb.WriteString(padRight(line, innerWidth))
			sb.WriteString(Vertical)
			sb.WriteString("\n")
		}
	}

	// Bottom border
	sb.WriteString(BottomLeft)
	sb.WriteString(strings.Repeat(Horizontal, innerWidth))
	sb.WriteString(BottomRight)
	sb.WriteString("\n")

	return sb.String()
}

// padRight pads a string to the specified width
func padRight(s string, width int) string {
	runeCount := len([]rune(s))
	if runeCount >= width {
		return string([]rune(s)[:width])
	}
	return s + strings.Repeat(" ", width-runeCount)
}

// FormatKeyValue formats a key-value pair with proper spacing
func FormatKeyValue(key, value string, keyWidth int) string {
	return "  " + padRight(key+":", keyWidth) + " " + value
}

// FormatList formats a list of items
func FormatList(items []string) string {
	if len(items) == 0 {
		return "(none)"
	}
	return strings.Join(items, ", ")
}
