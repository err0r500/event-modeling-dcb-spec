package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"

	"github.com/err0r500/event-modeling-dcb-spec/pkg/board"
	"github.com/err0r500/event-modeling-dcb-spec/pkg/tui"
	"github.com/err0r500/event-modeling-dcb-spec/pkg/web"
)

func main() {
	var (
		file      = flag.String("file", "", "CUE file to load (required)")
		boardName = flag.String("board", "", "Board name (default: first found)")
		outdir    = flag.String("outdir", "", "IR output directory (required)")
		watch     = flag.Bool("watch", true, "Watch CUE files and regenerate IR")
		webFlag   = flag.Bool("web", false, "Also run web server")
		port      = flag.Int("port", 3000, "Web server port")
		noTui     = flag.Bool("no-tui", false, "Disable TUI")
	)
	flag.Parse()

	if *file == "" {
		fmt.Fprintln(os.Stderr, "error: -file is required")
		flag.Usage()
		os.Exit(1)
	}
	if *outdir == "" {
		fmt.Fprintln(os.Stderr, "error: -outdir is required")
		flag.Usage()
		os.Exit(1)
	}

	// Initial render
	if err := writeIR(*file, *boardName, *outdir); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Start web server in background
	if *webFlag {
		fmt.Printf("starting the webserver on http://localhost:%d", port)
		go runWebServer(*outdir, *port)
	}

	// Start file watcher in background
	if *watch {
		go watchAndWrite(*file, *boardName, *outdir)
	}


	// Run TUI (blocking) or just wait
	if !*noTui {
		runTUI(*outdir)
	} else if *watch || *webFlag {
		// Keep running without TUI
		select {}
	}
}

func writeIR(filePath, boardName, outdir string) error {
	b, warnings, err := board.LoadBoardPermissive(filePath, boardName)
	if err != nil {
		board.WriteBoardError(outdir, boardName, []string{err.Error()})
		return err
	}

	srcDir := filepath.Dir(filePath)
	manifest, slices, images := board.ReifyBoardFiles(b, warnings)
	return board.WriteBoardFiles(outdir, manifest, slices, srcDir, images)
}

func watchAndWrite(filePath, boardName, outdir string) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		log.Fatalf("abs path: %v", err)
	}
	dir := filepath.Dir(absPath)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("watcher: %v", err)
	}
	defer watcher.Close()

	if err := watcher.Add(dir); err != nil {
		log.Fatalf("watch dir: %v", err)
	}

	log.Printf("watching %s → %s", dir, outdir)

	for {
		select {
		case ev, ok := <-watcher.Events:
			if !ok {
				return
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}
			time.Sleep(100 * time.Millisecond)
			for len(watcher.Events) > 0 {
				<-watcher.Events
			}
			if err := writeIR(filePath, boardName, outdir); err != nil {
				log.Printf("error: %v", err)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("watcher error: %v", err)
		}
	}
}

func runWebServer(outdir string, port int) {
	distFS, err := fs.Sub(web.Assets, "dist")
	if err != nil {
		log.Fatalf("web assets: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/.board/", http.StripPrefix("/.board/", http.FileServer(http.Dir(outdir))))
	mux.Handle("/", http.FileServer(http.FS(distFS)))

	addr := fmt.Sprintf(":%d", port)
	log.Printf("web server at http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("web server: %v", err)
	}
}

func runTUI(outdir string) {
	m, err := tui.NewIRModel(outdir)
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
