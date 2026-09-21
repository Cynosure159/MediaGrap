package library

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// directorySnapshot belongs to one directory visit, never to the service or
// library. Sorted sidecar names allow literal prefix lookup in flat libraries.
// File types are revalidated with Lstat when an asset is attributed.
type directorySnapshot struct {
	entries      []fs.DirEntry
	sidecarNames []string
	videos       int
}

func newDirectorySnapshot(entries []fs.DirEntry) directorySnapshot {
	snapshot := directorySnapshot{entries: entries}
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&fs.ModeSymlink != 0 {
			continue
		}
		extension := strings.ToLower(filepath.Ext(entry.Name()))
		if videoExtensions[extension] && !isSampleVideo(entry.Name()) {
			snapshot.videos++
		}
		if _, ok := sidecarExtensions[extension]; ok {
			snapshot.sidecarNames = append(snapshot.sidecarNames, entry.Name())
		}
	}
	sort.Strings(snapshot.sidecarNames)
	return snapshot
}

func (snapshot directorySnapshot) sidecars(root, relative string) []Sidecar {
	assets := []Sidecar{}
	if root == "" {
		return assets
	}
	directory := filepath.Dir(filepath.Join(root, relative))
	base := strings.TrimSuffix(filepath.Base(relative), filepath.Ext(relative))
	names := snapshot.sidecarNames
	_, _, _, isEpisode := parseEpisodeHint(filepath.Base(relative))
	if snapshot.videos != 1 || isEpisode {
		prefix := base + "."
		start := sort.SearchStrings(names, prefix)
		end := start
		for end < len(names) && strings.HasPrefix(names[end], prefix) {
			end++
		}
		names = names[start:end]
	}
	for _, name := range names {
		path := filepath.Join(directory, name)
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		relative, err := filepath.Rel(root, path)
		if err == nil {
			assets = append(assets, Sidecar{RelativePath: relative, Kind: sidecarExtensions[strings.ToLower(filepath.Ext(name))]})
		}
	}
	return assets
}

// walkScanDirectories indexes direct files before descending. The helper's
// snapshot is released before recursion; only pending child paths and the
// parent's regular-main-video flag survive. No whole-library cache is retained.
func walkScanDirectories(ctx context.Context, root string, readDir func(string) ([]fs.DirEntry, error), skip func(string, bool) (bool, error), visit func(string, directorySnapshot) error) error {
	var walk func(string, bool) error
	walk = func(directory string, parentHasMain bool) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		info, err := os.Lstat(directory)
		if err != nil {
			return err
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return nil
		}
		if !info.IsDir() {
			return &fs.PathError{Op: "readdir", Path: directory, Err: fs.ErrInvalid}
		}
		excluded, err := skip(directory, parentHasMain)
		if err != nil || excluded {
			return err
		}
		children, hasMain, err := scanDirectory(ctx, directory, readDir, visit)
		if err != nil {
			return err
		}
		for _, child := range children {
			if err := walk(child, hasMain); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(root, false)
}

func scanDirectory(ctx context.Context, directory string, readDir func(string) ([]fs.DirEntry, error), visit func(string, directorySnapshot) error) ([]string, bool, error) {
	entries, err := readDir(directory)
	if err != nil {
		return nil, false, err
	}
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	snapshot := newDirectorySnapshot(entries)
	children := []string{}
	needsMain := false
	for _, entry := range entries {
		if entry.IsDir() && entry.Type()&fs.ModeSymlink == 0 {
			children = append(children, filepath.Join(directory, entry.Name()))
			name := strings.ToLower(entry.Name())
			needsMain = needsMain || name == "extra" || name == "extras"
		}
	}
	hasMain := needsMain && entriesHaveMainVideo(entries)
	return children, hasMain, visit(directory, snapshot)
}
