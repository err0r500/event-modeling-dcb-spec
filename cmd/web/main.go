package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/fairway/eventmodelingspec/pkg/board"
	"github.com/fairway/eventmodelingspec/pkg/web"
)

var (
	filePath  string
	boardName string
	port      int

	mu           sync.RWMutex
	cachedBoard  *board.Board
	cachedErrors []string
	sliceCache   map[string][]byte
)

func main() {
	flag.StringVar(&filePath, "file", "", "CUE file to load (required)")
	flag.StringVar(&boardName, "board", "", "Board name (default: first found)")
	flag.IntVar(&port, "port", 3000, "HTTP port")
	flag.Parse()

	if filePath == "" {
		fmt.Fprintln(os.Stderr, "error: -file is required")
		flag.Usage()
		os.Exit(1)
	}

	if err := reloadBoard(); err != nil {
		fmt.Fprintf(os.Stderr, "error loading board: %v\n", err)
		os.Exit(1)
	}

	distFS, err := fs.Sub(web.Assets, "dist")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/.board/", handleBoard)
	mux.HandleFunc("/.reload", handleReload)
	mux.Handle("/", http.FileServer(http.FS(distFS)))

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Serving at http://localhost%s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func reloadBoard() error {
	b, warnings, err := board.LoadBoardPermissive(filePath, boardName)
	if err != nil {
		return err
	}

	mu.Lock()
	cachedBoard = b
	cachedErrors = warnings
	sliceCache = make(map[string][]byte)
	mu.Unlock()

	return nil
}

func handleReload(w http.ResponseWriter, r *http.Request) {
	if err := reloadBoard(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("reloaded"))
}

func handleBoard(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/.board/")

	mu.RLock()
	b := cachedBoard
	errs := cachedErrors
	cache := sliceCache
	mu.RUnlock()

	if b == nil {
		http.Error(w, "board not loaded", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if path == "board.json" {
		manifest, slices, _ := board.ReifyBoardFiles(b, errs)

		// Pre-cache slices
		mu.Lock()
		for filename, data := range slices {
			if _, ok := sliceCache[filename]; !ok {
				j, _ := json.Marshal(data)
				sliceCache[filename] = j
			}
		}
		mu.Unlock()

		json.NewEncoder(w).Encode(manifest)
		return
	}

	// Slice file request
	if data, ok := cache[path]; ok {
		w.Write(data)
		return
	}

	// Generate on-the-fly if not cached
	_, slices, _ := board.ReifyBoardFiles(b, errs)
	if data, ok := slices[path]; ok {
		j, _ := json.Marshal(data)

		mu.Lock()
		sliceCache[path] = j
		mu.Unlock()

		w.Write(j)
		return
	}

	http.NotFound(w, r)
}
