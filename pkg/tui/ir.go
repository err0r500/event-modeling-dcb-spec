package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"

	"github.com/err0r500/event-modeling-dcb-spec/pkg/board"
	"github.com/err0r500/event-modeling-dcb-spec/pkg/render"
)

type viewMode int

const (
	boardMode viewMode = iota
	searchMode
	detailMode
	errorMode
)

// irReloadedMsg is sent when the IR directory watcher detects a change.
type irReloadedMsg struct {
	manifest *board.BoardManifest
	slices   map[string]map[string]any
	err      error
}

// irWaitTickMsg is sent every 100ms while waiting for a file to appear.
type irWaitTickMsg struct{}

// IRModel is the TUI model for IR directory mode.
type IRModel struct {
	irDir    string
	manifest *board.BoardManifest
	slices   map[string]map[string]any

	mode           viewMode
	previousMode   viewMode
	previousFile   string // file we were viewing in detailMode (for error recovery)
	currentFile    string // file currently being viewed in detailMode
	waitingForFile string // file path we're waiting to appear (empty if not waiting)
	width          int
	height         int
	viewport       viewport.Model
	ready          bool
	tree           *TreeState
	reloadErr      string

	searchInput textinput.Model
}

// NewIRModel creates a TUI model from an IR directory.
func NewIRModel(dir string) (IRModel, error) {
	manifest, slices, err := loadIRDir(dir)
	if err != nil {
		return IRModel{}, err
	}

	tree := NewTreeState(manifest, slices)

	ti := textinput.New()
	ti.Prompt = "/ "
	ti.CharLimit = 64

	m := IRModel{
		irDir:       dir,
		manifest:    manifest,
		slices:      slices,
		mode:        boardMode,
		tree:        tree,
		searchInput: ti,
	}
	// Show manifest errors on initial load
	if len(manifest.Errors) > 0 {
		m.reloadErr = strings.Join(manifest.Errors, "\n")
	}
	return m, nil
}

func (m IRModel) Init() tea.Cmd {
	return m.watchIRDirCmd()
}

func waitTickCmd() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(100 * time.Millisecond)
		return irWaitTickMsg{}
	}
}

func (m IRModel) watchIRDirCmd() tea.Cmd {
	dir := m.irDir
	return func() tea.Msg {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return irReloadedMsg{err: err}
		}
		defer watcher.Close()

		if err := watcher.Add(dir); err != nil {
			return irReloadedMsg{err: err}
		}

		for {
			select {
			case ev, ok := <-watcher.Events:
				if !ok {
					return nil
				}
				if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) != 0 {
					time.Sleep(100 * time.Millisecond)
					for len(watcher.Events) > 0 {
						<-watcher.Events
					}
					manifest, slices, err := loadIRDir(dir)
					return irReloadedMsg{manifest: manifest, slices: slices, err: err}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return nil
				}
				return irReloadedMsg{err: err}
			}
		}
	}
}

func (m IRModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case irReloadedMsg:
		if msg.err != nil {
			m.reloadErr = msg.err.Error()
			if m.mode != errorMode {
				m.previousMode = m.mode
				if m.mode == detailMode && m.currentFile != "" {
					m.previousFile = m.currentFile
				}
			}
			m.mode = errorMode
			wrapped := lipgloss.NewStyle().Width(m.width - 2).Render(m.reloadErr)
			m.viewport.SetContent(wrapped)
			m.viewport.GotoTop()
			return m, m.watchIRDirCmd()
		}
		m.manifest = msg.manifest
		m.slices = msg.slices
		m.tree = NewTreeState(m.manifest, m.slices)
		// Show manifest-level errors
		if len(m.manifest.Errors) > 0 {
			m.reloadErr = strings.Join(m.manifest.Errors, "\n")
			if m.mode != errorMode {
				m.previousMode = m.mode
				if m.mode == detailMode && m.currentFile != "" {
					m.previousFile = m.currentFile
				}
			}
			m.mode = errorMode
			wrapped := lipgloss.NewStyle().Width(m.width - 2).Render(m.reloadErr)
			m.viewport.SetContent(wrapped)
			m.viewport.GotoTop()
		} else {
			// No errors - clear error state and restore previous view
			m.reloadErr = ""
			if m.mode == errorMode {
				if m.previousMode == detailMode && m.previousFile != "" {
					// Try to restore to detail view
					if data, ok := m.slices[m.previousFile]; ok {
						m.mode = detailMode
						m.currentFile = m.previousFile
						output, _ := render.RenderSliceIR(data, m.width)
						m.viewport.SetContent(output)
					} else {
						// File not ready yet, wait for it
						m.waitingForFile = m.previousFile
						m.mode = boardMode
					}
				} else {
					m.mode = m.previousMode
				}
				m.previousFile = ""
			}
		}
		// Check if we're waiting for a file to appear
		if m.waitingForFile != "" {
			if data, ok := m.slices[m.waitingForFile]; ok {
				// File appeared, restore to detailMode
				m.mode = detailMode
				m.currentFile = m.waitingForFile
				output, _ := render.RenderSliceIR(data, m.width)
				m.viewport.SetContent(output)
				m.waitingForFile = ""
			} else {
				// Keep waiting
				return m, tea.Batch(m.watchIRDirCmd(), waitTickCmd())
			}
		} else if m.mode == detailMode && m.currentFile != "" {
			if data, ok := m.slices[m.currentFile]; ok {
				output, _ := render.RenderSliceIR(data, m.width)
				m.viewport.SetContent(output)
			}
		}
		return m, m.watchIRDirCmd()

	case irWaitTickMsg:
		if m.waitingForFile == "" {
			return m, nil
		}
		// Check if file exists now
		filePath := filepath.Join(m.irDir, m.waitingForFile)
		if _, err := os.Stat(filePath); err == nil {
			// File exists, reload to get the data
			manifest, slices, err := loadIRDir(m.irDir)
			if err == nil && slices[m.waitingForFile] != nil {
				m.manifest = manifest
				m.slices = slices
				m.tree = NewTreeState(m.manifest, m.slices)
				m.mode = detailMode
				m.currentFile = m.waitingForFile
				output, _ := render.RenderSliceIR(slices[m.waitingForFile], m.width)
				m.viewport.SetContent(output)
				m.waitingForFile = ""
				return m, m.watchIRDirCmd()
			}
		}
		// Keep waiting
		return m, waitTickCmd()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-2)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 2
		}
		if m.mode == detailMode && m.currentFile != "" {
			if data, ok := m.slices[m.currentFile]; ok {
				output, _ := render.RenderSliceIR(data, m.width)
				m.viewport.SetContent(output)
			}
		}
		return m, nil

	case tea.KeyMsg:
		if m.mode == searchMode {
			switch msg.String() {
			case "esc":
				m.mode = boardMode
				m.searchInput.SetValue("")
				return m, nil
			case "enter":
				m.mode = boardMode
				return m, nil
			default:
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				return m, cmd
			}
		}

		switch msg.String() {
		case "q", "ctrl+c":
			if m.mode == detailMode {
				m.mode = boardMode
				m.currentFile = ""
				return m, nil
			}
			return m, tea.Quit
		case "esc":
			// Cancel waiting for file
			if m.waitingForFile != "" {
				m.waitingForFile = ""
				return m, nil
			}
			if m.mode == detailMode {
				m.mode = boardMode
				m.currentFile = ""
				return m, nil
			}
			if m.mode == errorMode {
				m.mode = boardMode
				return m, nil
			}
		case "/":
			if m.mode == boardMode {
				m.mode = searchMode
				m.searchInput.Focus()
				return m, textinput.Blink
			}
		case "e":
			if (m.mode == boardMode || m.mode == detailMode) && m.reloadErr != "" {
				m.mode = errorMode
				wrapped := lipgloss.NewStyle().Width(m.width - 2).Render(m.reloadErr)
				m.viewport.SetContent(wrapped)
				m.viewport.GotoTop()
				return m, nil
			}

		// Tree navigation
		case "j", "down":
			if m.mode == boardMode {
				m.tree.MoveDown()
				return m, nil
			}
		case "k", "up":
			if m.mode == boardMode {
				m.tree.MoveUp()
				return m, nil
			}
		case "enter", "l":
			if m.mode == boardMode {
				if m.tree.Expand() {
					// It was a slice - open detail view
					file := m.selectedSliceFile()
					if data := m.slices[file]; data != nil {
						m.mode = detailMode
						m.currentFile = file
						output, err := render.RenderSliceIR(data, m.width)
						if err != nil {
							m.viewport.SetContent(fmt.Sprintf("Error rendering: %v", err))
						} else {
							m.viewport.SetContent(output)
						}
						m.viewport.GotoTop()
					}
				}
				return m, nil
			}
		case "h":
			if m.mode == boardMode {
				m.tree.Collapse()
				return m, nil
			}
		case " ":
			if m.mode == boardMode {
				m.tree.Toggle()
				return m, nil
			}
		}

		if m.mode == detailMode || m.mode == errorMode {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// selectedSliceFile returns the file path for the currently selected row.
func (m IRModel) selectedSliceFile() string {
	idx := m.tree.CurrentFlowIndex()
	if idx < 0 || idx >= len(m.manifest.Flow) {
		return ""
	}
	entry := m.manifest.Flow[idx]

	// For stories, follow sliceRef
	if entry.Kind == "story" && entry.SliceRef != "" {
		for _, e := range m.manifest.Flow {
			if e.Kind == "slice" && e.Name == entry.SliceRef && e.File != "" {
				return e.File
			}
		}
		return ""
	}
	return entry.File
}

func (m IRModel) View() string {
	if !m.ready {
		return "Loading..."
	}
	switch m.mode {
	case detailMode:
		return m.renderDetailView()
	case errorMode:
		return m.renderErrorView()
	default:
		return m.renderBoardView()
	}
}

func (m IRModel) renderDetailView() string {
	name := ""
	if node := m.tree.Current(); node != nil {
		name = node.Name
	}

	header := titleStyle.
		Width(m.width).
		Render(fmt.Sprintf(" %s > %s ", m.manifest.Name, name))

	footer := lipgloss.NewStyle().
		Width(m.width).
		Foreground(lipgloss.Color("#626262")).
		Render(fmt.Sprintf(" %d%%  |  j/k: scroll  esc: back  q: quit",
			int(m.viewport.ScrollPercent()*100)))

	if m.reloadErr != "" {
		errMsg := m.reloadErr
		if len(errMsg) > m.width-20 {
			errMsg = errMsg[:m.width-20] + "..."
		}
		return header + "\n" + m.viewport.View() + "\n" +
			errorStyle.Render("error: "+errMsg+" [e: details]") + "\n" + footer
	}

	return header + "\n" + m.viewport.View() + "\n" + footer
}

func (m IRModel) renderErrorView() string {
	header := errorStyle.Width(m.width).Render(" Error ")
	footer := footerStyle.Width(m.width).Render(" j/k: scroll  esc: back ")
	return header + "\n" + m.viewport.View() + "\n" + footer
}

func (m IRModel) renderBoardView() string {
	var s strings.Builder

	// Header
	header := titleStyle.Width(m.width).Render(fmt.Sprintf(" %s ", m.manifest.Name))
	s.WriteString(header + "\n\n")

	// Tree view
	s.WriteString(m.renderTree())

	// Footer with keybindings
	s.WriteString("\n")
	if m.waitingForFile != "" {
		s.WriteString(footerStyle.Render("waiting for "+m.waitingForFile+"... [esc: cancel]") + "\n")
	} else if m.reloadErr != "" {
		errMsg := m.reloadErr
		if len(errMsg) > m.width-20 {
			errMsg = errMsg[:m.width-20] + "..."
		}
		s.WriteString(errorStyle.Render("error: "+errMsg+" [e: details]") + "\n")
	}
	s.WriteString(footerStyle.Render(" j/k: nav  enter/l: expand/open  h: collapse  space: toggle  q: quit"))

	return s.String()
}

// renderTree renders the tree view.
func (m IRModel) renderTree() string {
	var lines []string
	visibleHeight := m.height - 6 // account for header + footer
	if visibleHeight < 5 {
		visibleHeight = 5
	}

	// Calculate visible window
	start := 0
	if m.tree.Cursor >= visibleHeight {
		start = m.tree.Cursor - visibleHeight + 1
	}
	end := start + visibleHeight
	if end > len(m.tree.FlatView) {
		end = len(m.tree.FlatView)
	}

	for i := start; i < end; i++ {
		node := m.tree.FlatView[i]
		isCursor := i == m.tree.Cursor

		// Build line: indent + icon + name + extras
		var line strings.Builder

		// Indent
		for j := 0; j < node.Depth; j++ {
			line.WriteString(treeIndent)
		}

		// Expand icon
		if len(node.Children) > 0 {
			if m.tree.Expanded[node] {
				line.WriteString(treeExpandedIcon)
			} else {
				line.WriteString(treeCollapsedIcon)
			}
		} else {
			line.WriteString(treeLeafIcon)
		}

		// Name with type prefix for slices
		name := node.Name
		switch node.Kind {
		case NodeSlice:
			prefix := ""
			if node.SliceType == "change" {
				prefix = "[CMD] "
			} else if node.SliceType == "view" {
				prefix = "[VIEW] "
			}
			name = prefix + name
			if node.DevStatus != "" {
				name += " (" + node.DevStatus + ")"
			}
		}
		line.WriteString(name)

		// Apply styling
		lineStr := line.String()
		var styled string
		if isCursor {
			styled = treeCursorStyle.Width(m.width).Render(lineStr)
		} else {
			switch node.Kind {
			case NodeContext:
				styled = treeContextStyle.Render(lineStr)
			case NodeChapter:
				styled = treeChapterStyle.Render(lineStr)
			default:
				styled = treeSliceStyle.Render(lineStr)
			}
		}

		lines = append(lines, styled)
	}

	return strings.Join(lines, "\n")
}

// --- IR data helpers ---

func loadIRDir(dir string) (*board.BoardManifest, map[string]map[string]any, error) {
	manifestPath := filepath.Join(dir, "board.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read board.json: %w", err)
	}

	var manifest board.BoardManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, nil, fmt.Errorf("parse board.json: %w", err)
	}

	slices := make(map[string]map[string]any)
	for _, entry := range manifest.Flow {
		if entry.File == "" {
			continue
		}
		sliceData, err := os.ReadFile(filepath.Join(dir, entry.File))
		if err != nil {
			continue // slice file may not exist yet
		}
		var m map[string]any
		if err := json.Unmarshal(sliceData, &m); err != nil {
			continue
		}
		slices[entry.File] = m
	}

	return &manifest, slices, nil
}

