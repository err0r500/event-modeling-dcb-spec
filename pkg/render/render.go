package render

import (
	"strings"

	"cuelang.org/go/cue"
)

// renderImageASCII is disabled for TUI (images only in web version)
func renderImageASCII(imgPath string, width int) string {
	return ""
}

// normalizeIndent finds the minimum leading space across all lines and trims that from each
func normalizeIndent(ascii string) []string {
	lines := strings.Split(ascii, "\n")
	if len(lines) == 0 {
		return nil
	}

	// Find minimum indent (ignoring empty lines)
	minIndent := -1
	for _, line := range lines {
		if line == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if minIndent < 0 || indent < minIndent {
			minIndent = indent
		}
	}

	if minIndent <= 0 {
		return lines
	}

	// Trim minIndent from each line
	result := make([]string, len(lines))
	for i, line := range lines {
		if len(line) >= minIndent {
			result[i] = line[minIndent:]
		} else {
			result[i] = line
		}
	}
	return result
}

// getString extracts a string from a CUE value at the given path
func getString(v cue.Value, path string) string {
	val := v.LookupPath(cue.ParsePath(path))
	if val.Err() != nil {
		return ""
	}
	s, err := val.String()
	if err != nil {
		return ""
	}
	return s
}
