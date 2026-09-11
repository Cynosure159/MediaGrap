package library

import (
	"os"
	"path/filepath"
	"strings"
)

// isSampleVideo recognizes only explicit final sample tokens, never arbitrary
// substrings in a movie title (for example "The Sample (2024)").
func isSampleVideo(name string) bool {
	stem := strings.ToLower(strings.TrimSuffix(filepath.Base(name), filepath.Ext(name)))
	return stem == "sample" || strings.HasSuffix(stem, "-sample") || strings.HasSuffix(stem, ".sample") || strings.HasSuffix(stem, "_sample")
}

// An Extra(s) folder is supplemental only when its parent contains a regular
// main video. A source/movie simply named Extras is not excluded by its name.
func isSupplementalDirectory(root, path string) bool {
	if path == root {
		return false
	}
	name := strings.ToLower(filepath.Base(path))
	if name != "extra" && name != "extras" {
		return false
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !videoExtensions[strings.ToLower(filepath.Ext(entry.Name()))] || isSampleVideo(entry.Name()) {
			continue
		}
		if info, err := entry.Info(); err == nil && info.Mode().IsRegular() {
			return true
		}
	}
	return false
}
