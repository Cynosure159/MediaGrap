package library

import (
	"context"
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

// indexedSupplementalDirectory preserves classification after the main file
// was deleted. Only direct parent videos count, never videos inside Extras.
func (s *Service) indexedSupplementalDirectory(ctx context.Context, sourceID int64, root, path string) (bool, error) {
	name := strings.ToLower(filepath.Base(path))
	if path == root || (name != "extra" && name != "extras") {
		return false, nil
	}
	parent, err := filepath.Rel(root, filepath.Dir(path))
	if err != nil {
		return false, err
	}
	prefix := ""
	if parent != "." {
		prefix = parent + string(filepath.Separator)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT relative_path FROM media_items WHERE source_id=? AND substr(relative_path,1,length(?))=? AND instr(substr(relative_path,length(?)+1),?)=0`, sourceID, prefix, prefix, prefix, string(filepath.Separator))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var relative string
		if err := rows.Scan(&relative); err != nil {
			return false, err
		}
		if !isSampleVideo(relative) {
			return true, nil
		}
	}
	return false, rows.Err()
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
