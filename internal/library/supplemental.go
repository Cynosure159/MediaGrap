package library

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	supplementalDirectoryEntryBudget = 256
	supplementalMaxRows              = 64
	supplementalDirectoryBatch       = 32
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

// Only a regular non-sample parent video makes an Extra(s) child supplemental.
func entriesHaveMainVideo(entries []fs.DirEntry) bool {
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

type supplementalAudit struct {
	files   []FileAuditEntry
	status  string
	warning string
}

// supplementalOperations is request-local fault/operation instrumentation. It cannot
// replace filesystem results or bypass confinement and cancellation checks.
type supplementalOperations struct {
	before func(operation, path string) error
}

func (ops supplementalOperations) check(ctx context.Context, operation, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if ops.before != nil {
		if err := ops.before(operation, path); err != nil {
			return err
		}
	}
	return ctx.Err()
}

type supplementalCandidate struct {
	path string
	kind string
	info os.FileInfo
}

var errSupplementalLimit = errors.New("supplemental entry budget exhausted")
var errSupplementalRows = errors.New("supplemental row limit reached")
var errSupplementalUnsafe = errors.New("supplemental entry is unsafe or changed")

// auditSupplementalFiles is separate from Files, which supplies rename/NFO plans.
func (s *Service) auditSupplementalFiles(ctx context.Context, location MediaLocation) supplementalAudit {
	return s.auditSupplementalFilesWithOperations(ctx, location, supplementalOperations{})
}

func (s *Service) auditSupplementalFilesWithOperations(ctx context.Context, location MediaLocation, ops supplementalOperations) supplementalAudit {
	result := supplementalAudit{files: []FileAuditEntry{}, status: "not_associated"}
	fail := func(err error) supplementalAudit {
		result.files = nil
		result.status = "unreadable"
		result.warning = "supplemental scan found an unsafe, changed or unreadable entry"
		if ctx.Err() != nil {
			result.status = "incomplete"
			result.warning = "supplemental scan cancelled"
		}
		return result
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	if !supplementalRootSupported {
		result.status = "unsupported"
		result.warning = "supplemental filesystem confinement is unsupported on this platform"
		return result
	}
	// Use the same persisted TV exclusion as the movie catalog. A failed lookup
	// (including a missing source/item) must not authorize any ancillary I/O.
	var rootPath string
	var movie bool
	err := s.db.QueryRowContext(ctx, `SELECT src.root_path, m.missing=0 AND NOT EXISTS(SELECT 1 FROM tv_episodes e WHERE e.media_item_id=m.id) FROM media_items m JOIN sources src ON src.id=m.source_id WHERE m.id=? AND m.source_id=? AND m.relative_path=?`, location.Item.ID, location.Item.SourceID, location.Item.RelativePath).Scan(&rootPath, &movie)
	if err != nil {
		return fail(err)
	}
	relative := filepath.Clean(location.Item.RelativePath)
	parentPath := filepath.Dir(relative)
	if !movie || parentPath == "." || !filepath.IsLocal(relative) {
		result.warning = "supplemental files require a movie in its own directory, not a source root or TV episode"
		return result
	}
	root, err := openSupplementalSource(ctx, rootPath, ops)
	if err != nil {
		return fail(err)
	}
	defer root.Close()
	parent := root
	// Pin each component, rejecting symlinks and checking the opened identity.
	// All later enumeration and leaf Lstat use these handles, not absolute paths.
	prefix := ""
	for _, component := range strings.Split(parentPath, string(filepath.Separator)) {
		prefix = filepath.Join(prefix, component)
		child, openErr := openSupplementalChild(ctx, parent, component, prefix, nil, ops)
		if openErr != nil {
			return fail(openErr)
		}
		defer child.Close()
		parent = child
	}
	if err := ops.check(ctx, "lstat", relative); err != nil {
		return fail(err)
	}
	selected, err := parent.Lstat(filepath.Base(relative))
	if err != nil {
		result.status = "missing"
		result.warning = "selected movie is unavailable"
		return result
	}
	if !selected.Mode().IsRegular() {
		return fail(errSupplementalUnsafe)
	}
	budget := supplementalDirectoryEntryBudget
	mainCount := 0
	selectedMain := false
	var samples, extras []supplementalCandidate
	err = enumerateSupplementalDirectory(ctx, parent, parentPath, &budget, ops, func(name string, info os.FileInfo) error {
		path := filepath.Join(parentPath, name)
		if info.IsDir() {
			if strings.EqualFold(name, "extra") || strings.EqualFold(name, "extras") {
				extras = append(extras, supplementalCandidate{path: path, info: info})
			}
			return nil
		}
		if !videoExtensions[strings.ToLower(filepath.Ext(name))] {
			return nil
		}
		if !info.Mode().IsRegular() {
			return errSupplementalUnsafe
		}
		if isSampleVideo(name) {
			if len(samples) <= supplementalMaxRows {
				samples = append(samples, supplementalCandidate{path: path, kind: "sample", info: info})
			}
		} else {
			mainCount++
			selectedMain = name == filepath.Base(relative) && os.SameFile(selected, info)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, errSupplementalLimit) {
			result.status = "incomplete"
			result.warning = "entry budget exhausted before unique movie association could be proved"
			return result
		}
		return fail(err)
	}
	if mainCount != 1 || !selectedMain {
		result.warning = "supplemental files require exactly one direct non-sample main video"
		return result
	}
	candidates := samples
	limitErr := error(nil)
	if len(candidates) > supplementalMaxRows {
		candidates = candidates[:supplementalMaxRows]
		limitErr = errSupplementalRows
	}
	for _, extra := range extras {
		if limitErr != nil {
			break
		}
		directory, openErr := openSupplementalChild(ctx, parent, filepath.Base(extra.path), extra.path, extra.info, ops)
		if openErr != nil {
			return fail(openErr)
		}
		readErr := enumerateSupplementalDirectory(ctx, directory, extra.path, &budget, ops, func(name string, info os.FileInfo) error {
			if info.IsDir() {
				return nil
			} // Deliberately nonrecursive.
			if !videoExtensions[strings.ToLower(filepath.Ext(name))] {
				return nil
			}
			if !info.Mode().IsRegular() {
				return errSupplementalUnsafe
			}
			if len(candidates) >= supplementalMaxRows {
				return errSupplementalRows
			}
			candidates = append(candidates, supplementalCandidate{path: filepath.Join(extra.path, name), kind: "extra", info: info})
			return nil
		})
		directory.Close()
		if readErr != nil {
			if errors.Is(readErr, errSupplementalRows) || errors.Is(readErr, errSupplementalLimit) {
				limitErr = readErr
				break
			}
			return fail(readErr)
		}
	}
	// Metadata is a rooted Lstat snapshot; no candidate leaf is ever opened,
	// probed, or re-resolved after collection (including FIFO/symlink swaps).
	files, err := supplementalAuditCandidates(ctx, candidates, ops)
	if err != nil {
		return fail(err)
	}
	result.files = files
	result.status = "ready"
	if limitErr != nil {
		result.status = "truncated"
		result.warning = "supplemental display or entry limit reached; unvisited entries were not checked"
	}
	return result
}

// The configured source's ancestors are the existing administrator-controlled
// mount boundary. The literal / . suffix forces directory resolution on Unix,
// so a source-directory-to-FIFO swap cannot block OpenRoot before validation.
func openSupplementalSource(ctx context.Context, path string, ops supplementalOperations) (*os.Root, error) {
	if err := ops.check(ctx, "source-lstat", path); err != nil {
		return nil, err
	}
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !before.IsDir() || before.Mode()&os.ModeSymlink != 0 {
		return nil, errSupplementalUnsafe
	}
	if err := ops.check(ctx, "source-open", path); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(path + string(filepath.Separator) + ".")
	if err != nil {
		return nil, err
	}
	if err = ops.check(ctx, "source-stat", path); err == nil {
		var opened os.FileInfo
		opened, err = root.Stat(".")
		if err == nil && (!opened.IsDir() || !os.SameFile(before, opened)) {
			err = errSupplementalUnsafe
		}
	}
	if err == nil {
		err = ops.check(ctx, "source-recheck", path)
	}
	if err == nil {
		after, statErr := os.Lstat(path)
		err = statErr
		if err == nil && (!after.IsDir() || !os.SameFile(before, after)) {
			err = errSupplementalUnsafe
		}
	}
	if err != nil {
		root.Close()
		return nil, err
	}
	return root, nil
}

func openSupplementalChild(ctx context.Context, parent *os.Root, name, path string, expected os.FileInfo, ops supplementalOperations) (*os.Root, error) {
	if err := ops.check(ctx, "directory-lstat", path); err != nil {
		return nil, err
	}
	before, err := parent.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !before.IsDir() || (expected != nil && !os.SameFile(before, expected)) {
		return nil, errSupplementalUnsafe
	}
	if err := ops.check(ctx, "directory-open", path); err != nil {
		return nil, err
	}
	// Keep the suffix literal: filepath.Join would remove the directory-only
	// constraint. Root confines even a racing symlink; identity checks reject it.
	child, err := parent.OpenRoot(name + string(filepath.Separator) + ".")
	if err != nil {
		return nil, err
	}
	if err = ops.check(ctx, "directory-stat", path); err == nil {
		var opened os.FileInfo
		opened, err = child.Stat(".")
		if err == nil && (!opened.IsDir() || !os.SameFile(before, opened)) {
			err = errSupplementalUnsafe
		}
	}
	if err == nil {
		err = ops.check(ctx, "directory-recheck", path)
	}
	if err == nil {
		after, statErr := parent.Lstat(name)
		err = statErr
		if err == nil && (!after.IsDir() || !os.SameFile(before, after)) {
			err = errSupplementalUnsafe
		}
	}
	if err != nil {
		child.Close()
		return nil, err
	}
	return child, nil
}

func enumerateSupplementalDirectory(ctx context.Context, root *os.Root, path string, budget *int, ops supplementalOperations, visit func(string, os.FileInfo) error) error {
	if err := ops.check(ctx, "enumerate-open", path); err != nil {
		return err
	}
	directory, err := openSupplementalDirectory(root)
	if err != nil {
		return err
	}
	defer directory.Close()
	if err := ops.check(ctx, "enumerate-stat", path); err != nil {
		return err
	}
	info, err := directory.Stat()
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errSupplementalUnsafe
	}
	for {
		if err := ops.check(ctx, "readdir", path); err != nil {
			return err
		}
		if *budget <= 0 {
			return errSupplementalLimit
		}
		batchSize := min(supplementalDirectoryBatch, *budget)
		entries, readErr := directory.ReadDir(batchSize)
		// A real I/O error takes precedence over rows/entry limits, even when
		// ReadDir also returned entries. EOF alone is intentional completion.
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return readErr
		}
		*budget -= len(entries)
		for _, entry := range entries {
			entryPath := filepath.Join(path, entry.Name())
			if err := ops.check(ctx, "lstat", entryPath); err != nil {
				return err
			}
			info, err := root.Lstat(entry.Name())
			if err != nil {
				return err
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return errSupplementalUnsafe
			}
			if err := ops.check(ctx, "visit", entryPath); err != nil {
				return err
			}
			if err := visit(entry.Name(), info); err != nil {
				return err
			}
		}
		if errors.Is(readErr, io.EOF) || len(entries) < batchSize {
			return nil
		}
		if *budget <= 0 {
			if err := ops.check(ctx, "readdir-overflow", path); err != nil {
				return err
			}
			more, err := directory.ReadDir(1)
			if err != nil && !errors.Is(err, io.EOF) {
				return err
			}
			if len(more) > 0 {
				return errSupplementalLimit
			}
			if errors.Is(err, io.EOF) {
				return nil
			}
			return errSupplementalLimit
		}
	}
}

func supplementalAuditCandidates(ctx context.Context, candidates []supplementalCandidate, ops supplementalOperations) ([]FileAuditEntry, error) {
	files := make([]FileAuditEntry, 0, len(candidates))
	if err := ops.check(ctx, "collected", ""); err != nil {
		return nil, err
	}
	for _, candidate := range candidates {
		if err := ops.check(ctx, "candidate", candidate.path); err != nil {
			return nil, err
		}
		info := candidate.info
		files = append(files, FileAuditEntry{
			RelativePath: candidate.path, Kind: candidate.kind, Size: info.Size(),
			MIMEType:    mime.TypeByExtension(strings.ToLower(filepath.Ext(candidate.path))),
			ModifiedAt:  info.ModTime().UTC().Format(time.RFC3339),
			Permissions: info.Mode().Perm().String(), Writable: info.Mode().Perm()&0o222 != 0,
			Regular: true, Valid: true, Warnings: supplementalWarnings(info),
		})
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return files, nil
}

func supplementalWarnings(info os.FileInfo) []string {
	if info.Mode().Perm()&0o222 == 0 {
		return []string{"read_only"}
	}
	return []string{}
}
