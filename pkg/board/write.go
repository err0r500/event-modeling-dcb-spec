package board

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// WriteBoardFiles writes the manifest and per-slice JSON files atomically.
// Stale .json files not in the current set are removed.
// If srcDir and images are provided, copies image files preserving relative paths.
func WriteBoardFiles(outdir string, manifest BoardManifest, slices map[string]map[string]any, srcDir string, images []string) error {
	if err := os.MkdirAll(outdir, 0o755); err != nil {
		return err
	}

	keep := map[string]bool{"board.json": true}

	// Write slice files
	for filename, data := range slices {
		keep[filename] = true
		b, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}
		if err := writeIfChanged(filepath.Join(outdir, filename), b); err != nil {
			return err
		}
	}

	// Write manifest
	b, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := writeIfChanged(filepath.Join(outdir, "board.json"), b); err != nil {
		return err
	}

	// Copy images
	for _, img := range images {
		srcPath := filepath.Join(srcDir, img)
		dstPath := filepath.Join(outdir, img)
		if err := copyFile(srcPath, dstPath); err != nil {
			// Log but don't fail - image might not exist yet
			continue
		}
		keep[img] = true
	}

	return cleanStale(outdir, keep)
}

// copyFile copies a file, creating parent directories as needed.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return writeIfChanged(dst, data)
}

// WriteBoardError writes a board.json with errors only, removing all slice files.
func WriteBoardError(outdir string, boardName string, errs []string) error {
	if err := os.MkdirAll(outdir, 0o755); err != nil {
		return err
	}

	manifest := BoardManifest{Name: boardName, Errors: errs}
	b, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := writeIfChanged(filepath.Join(outdir, "board.json"), b); err != nil {
		return err
	}
	return cleanStale(outdir, map[string]bool{"board.json": true})
}

// writeIfChanged writes data only if the file content differs. Uses atomic tmp+rename.
func writeIfChanged(path string, data []byte) error {
	existing, err := os.ReadFile(path)
	if err == nil && bytes.Equal(existing, data) {
		return nil
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// cleanStale removes .json files in outdir not in the keep set.
func cleanStale(outdir string, keep map[string]bool) error {
	entries, err := os.ReadDir(outdir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".json") && !keep[name] {
			os.Remove(filepath.Join(outdir, name))
		}
	}
	return nil
}
