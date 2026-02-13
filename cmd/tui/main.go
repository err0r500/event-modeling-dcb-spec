package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/fairway/eventmodelingspec/pkg/tui"
)

func main() {
	dir := flag.String("dir", "", "IR directory to load")
	flag.Parse()

	if *dir == "" {
		fmt.Fprintln(os.Stderr, "error: -dir is required")
		flag.Usage()
		os.Exit(1)
	}

	m, err := tui.NewIRModel(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
