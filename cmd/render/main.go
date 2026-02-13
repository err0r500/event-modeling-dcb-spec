package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/err0r500/event-modeling-dcb-spec/pkg/board"
)

func main() {
	var (
		file      = flag.String("file", "", "CUE file to load")
		boardName = flag.String("board", "", "Name of board in file (default: first found)")
		outdir    = flag.String("outdir", "", "Write multi-file IR to directory")
		watch     = flag.Bool("watch", false, "Watch CUE files and rewrite IR on change (requires -outdir)")
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

	if *watch && *outdir == "" {
		fmt.Fprintln(os.Stderr, "error: -watch requires -outdir")
		os.Exit(1)
	}

	if err := writeIR(*file, *boardName, *outdir); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if !*watch {
		return
	}
	watchAndWrite(*file, *boardName, *outdir)
}

func writeIR(filePath, boardName, outdir string) error {
	b, warnings, err := board.LoadBoardPermissive(filePath, boardName)
	if err != nil {
		// Hard error: write error-only manifest
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
			// Debounce
			time.Sleep(100 * time.Millisecond)
			for len(watcher.Events) > 0 {
				<-watcher.Events
			}
			if err := writeIR(filePath, boardName, outdir); err != nil {
				log.Printf("error: %v", err)
			} else {
				log.Printf("updated %s", outdir)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("watcher error: %v", err)
		}
	}
}
