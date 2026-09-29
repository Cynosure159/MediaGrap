//go:build linux || darwin

package library

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func supplementalLocation(t *testing.T, s *Service, id int64) MediaLocation {
	t.Helper()
	location, err := s.LocateMedia(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	return location
}

func supplementalWrite(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("owned"), 0600); err != nil {
		t.Fatal(err)
	}
}

func supplementalSwap(t *testing.T, path, replacement, outside string) {
	t.Helper()
	if err := os.Rename(path, path+"-original"); err != nil {
		t.Fatal(err)
	}
	switch replacement {
	case "symlink":
		if err := os.Symlink(outside, path); err != nil {
			t.Fatal(err)
		}
	case "fifo":
		if err := unix.Mkfifo(path, 0600); err != nil {
			t.Fatal(err)
		}
	case "directory":
		if err := os.Mkdir(path, 0750); err != nil {
			t.Fatal(err)
		}
	}
}

// A timeout is a failure, not a successful cancellation of a blocked syscall.
func supplementalNoHang(t *testing.T, run func() supplementalAudit) supplementalAudit {
	t.Helper()
	done := make(chan supplementalAudit, 1)
	go func() { done <- run() }()
	select {
	case result := <-done:
		return result
	case <-time.After(3 * time.Second):
		t.Fatal("supplemental filesystem operation hung")
		return supplementalAudit{}
	}
}

func TestSupplementalAuditRejectsDirectorySwapsBeforeUse(t *testing.T) {
	for _, boundary := range []string{"source", "ancestor", "parent", "extras"} {
		for _, replacement := range []string{"symlink", "fifo", "directory"} {
			t.Run(boundary+"/"+replacement, func(t *testing.T) {
				s, id, main := supplementalFixture(t, "Collection/Movie/Movie.mkv")
				parent := filepath.Dir(main)
				extra := filepath.Join(parent, "Extras")
				supplementalWrite(t, filepath.Join(extra, "clip.mkv"))
				location := supplementalLocation(t, s, id)
				root := s.sourceRoot(t.Context(), location.Item.SourceID)
				outside := t.TempDir()
				supplementalWrite(t, filepath.Join(outside, "secret.mkv"))
				supplementalWrite(t, filepath.Join(outside, "Movie", "secret.mkv"))
				target, operation, relative := root, "source-open", root
				switch boundary {
				case "ancestor":
					target, operation, relative = filepath.Dir(parent), "directory-open", "Collection"
				case "parent":
					target, operation, relative = parent, "directory-open", "Collection/Movie"
				case "extras":
					target, operation, relative = extra, "directory-open", "Collection/Movie/Extras"
				}
				swapped := false
				ancillaryAfterSwap := 0
				ops := supplementalOperations{before: func(op, path string) error {
					if op == operation && path == relative && !swapped {
						supplementalSwap(t, target, replacement, outside)
						swapped = true
					} else if swapped && (op == "lstat" || op == "readdir" || op == "enumerate-open") {
						ancillaryAfterSwap++
					}
					return nil
				}}
				result := supplementalNoHang(t, func() supplementalAudit { return s.auditSupplementalFilesWithOperations(t.Context(), location, ops) })
				if !swapped || result.status != "unreadable" || len(result.files) != 0 || ancillaryAfterSwap != 0 {
					t.Fatalf("swap=%v subsequent ancillary ops=%d result=%+v", swapped, ancillaryAfterSwap, result)
				}
			})
		}
	}
}

func TestSupplementalAuditPinsOpenedDirectoriesAcrossRename(t *testing.T) {
	for _, boundary := range []string{"ancestor", "parent", "extras"} {
		t.Run(boundary, func(t *testing.T) {
			s, id, main := supplementalFixture(t, "Collection/Movie/Movie.mkv")
			parent := filepath.Dir(main)
			extra := filepath.Join(parent, "Extras")
			supplementalWrite(t, filepath.Join(extra, "clip.mkv"))
			location := supplementalLocation(t, s, id)
			outside := t.TempDir()
			supplementalWrite(t, filepath.Join(outside, "secret.mkv"))
			// The matching basename has a conspicuous size; neither it nor the
			// outside-only name may appear after swapping an already pinned path.
			if err := os.WriteFile(filepath.Join(outside, "clip.mkv"), make([]byte, 9999), 0600); err != nil {
				t.Fatal(err)
			}
			target := extra
			if boundary == "parent" {
				target = parent
			}
			if boundary == "ancestor" {
				target = filepath.Dir(parent)
			}
			swapped := false
			ops := supplementalOperations{before: func(op, path string) error {
				if op == "readdir" && path == "Collection/Movie/Extras" && !swapped {
					supplementalSwap(t, target, "symlink", outside)
					swapped = true
				}
				return nil
			}}
			result := s.auditSupplementalFilesWithOperations(t.Context(), location, ops)
			if !swapped || result.status != "ready" || len(result.files) != 1 || result.files[0].Size != 5 || !strings.HasSuffix(result.files[0].RelativePath, "clip.mkv") {
				t.Fatalf("pinned directory was redirected: %+v", result)
			}
		})
	}
}

func TestSupplementalAuditNeverOpensCandidateLeaves(t *testing.T) {
	for _, timing := range []string{"lstat", "visit", "collected"} {
		for _, replacement := range []string{"symlink", "fifo"} {
			t.Run(timing+"/"+replacement, func(t *testing.T) {
				s, id, main := supplementalFixture(t, "Movie/Movie.mkv")
				leaf := filepath.Join(filepath.Dir(main), "Extras", "clip.mkv")
				supplementalWrite(t, leaf)
				outside := filepath.Join(t.TempDir(), "secret.mkv")
				if err := os.WriteFile(outside, make([]byte, 9999), 0600); err != nil {
					t.Fatal(err)
				}
				location := supplementalLocation(t, s, id)
				swapped := false
				ops := supplementalOperations{before: func(op, path string) error {
					if op == timing && (path == "Movie/Extras/clip.mkv" || timing == "collected") && !swapped {
						supplementalSwap(t, leaf, replacement, outside)
						swapped = true
					}
					return nil
				}}
				result := supplementalNoHang(t, func() supplementalAudit { return s.auditSupplementalFilesWithOperations(t.Context(), location, ops) })
				if !swapped {
					t.Fatal("swap hook not reached")
				}
				if timing == "lstat" {
					if result.status != "unreadable" || len(result.files) != 0 {
						t.Fatalf("unsafe leaf accepted: %+v", result)
					}
				} else if result.status != "ready" || len(result.files) != 1 || result.files[0].Size != 5 {
					t.Fatalf("snapshot leaf was reopened: %+v", result)
				}
			})
		}
	}
}

func TestSupplementalAuditClassificationGuardsBeforeFilesystemWork(t *testing.T) {
	for _, kind := range []string{"source-root", "episode", "query-failure"} {
		t.Run(kind, func(t *testing.T) {
			relative := "Movie/Movie.mkv"
			if kind == "source-root" {
				relative = "Movie.mkv"
			}
			if kind == "episode" {
				relative = "Show/Season 01/Show.S01E01.mkv"
			}
			s, id, main := supplementalFixture(t, relative)
			supplementalWrite(t, filepath.Join(filepath.Dir(main), "sample.mp4"))
			supplementalWrite(t, filepath.Join(filepath.Dir(main), "Extras", "clip.mkv"))
			if kind == "source-root" {
				supplementalWrite(t, filepath.Join(filepath.Dir(main), "Nested", "Other.mkv"))
			}
			if kind == "episode" {
				if err := s.scan(t.Context(), 0, 1, "full", nil); err != nil {
					t.Fatal(err)
				}
				var count int
				if err := s.db.QueryRow(`SELECT count(*) FROM tv_episodes WHERE media_item_id=?`, id).Scan(&count); err != nil || count != 1 {
					t.Fatalf("not an indexed episode: %d %v", count, err)
				}
			}
			location := supplementalLocation(t, s, id)
			if kind == "query-failure" {
				if err := s.db.Close(); err != nil {
					t.Fatal(err)
				}
			}
			operations := 0
			result := s.auditSupplementalFilesWithOperations(t.Context(), location, supplementalOperations{before: func(_, _ string) error { operations++; return nil }})
			want := "not_associated"
			if kind == "query-failure" {
				want = "unreadable"
			}
			if result.status != want || len(result.files) != 0 || operations != 0 {
				t.Fatalf("classification authorized I/O: ops=%d result=%+v", operations, result)
			}
		})
	}
}

func TestSupplementalAuditUncertaintyOverridesLimits(t *testing.T) {
	for _, fault := range []string{"symlink", "stat", "read", "open"} {
		for _, budget := range []string{"rows", "entries"} {
			t.Run(fault+"/"+budget, func(t *testing.T) {
				s, id, main := supplementalFixture(t, "Movie/Movie.mkv")
				parent := filepath.Dir(main)
				supplementalWrite(t, filepath.Join(parent, "sample.mp4"))
				for i := 0; i < supplementalDirectoryEntryBudget+1; i++ {
					ext := ".mkv"
					if budget == "entries" {
						ext = ".txt"
					}
					supplementalWrite(t, filepath.Join(parent, "Extras", fmt.Sprintf("%03d%s", i, ext)))
				}
				location := supplementalLocation(t, s, id)
				faulted := false
				ops := supplementalOperations{before: func(op, path string) error {
					if faulted || !strings.HasPrefix(path, "Movie/Extras") {
						return nil
					}
					if (fault == "open" && op == "directory-open") || (fault == "read" && op == "readdir") || (fault == "stat" && op == "lstat") {
						faulted = true
						return os.ErrPermission
					}
					if fault == "symlink" && op == "lstat" {
						faulted = true
						supplementalSwap(t, filepath.Join(filepath.Dir(parent), path), "symlink", t.TempDir())
					}
					return nil
				}}
				result := s.auditSupplementalFilesWithOperations(t.Context(), location, ops)
				if !faulted || result.status != "unreadable" || len(result.files) != 0 {
					t.Fatalf("unsafe partial result: %+v faulted=%v", result, faulted)
				}
			})
		}
	}
}

func TestSupplementalAuditCancellationStopsEveryLevel(t *testing.T) {
	for _, point := range []string{"source-open", "directory-open", "readdir", "lstat", "visit", "collected", "candidate"} {
		t.Run(point, func(t *testing.T) {
			s, id, main := supplementalFixture(t, "Movie/Movie.mkv")
			for i := 0; i < 40; i++ {
				supplementalWrite(t, filepath.Join(filepath.Dir(main), "Extras", fmt.Sprintf("%02d.mkv", i)))
			}
			location := supplementalLocation(t, s, id)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			calls, afterCancellation := 0, 0
			ops := supplementalOperations{before: func(op, path string) error {
				if ctx.Err() != nil {
					afterCancellation++
				}
				if op == point {
					calls++
					// Cancel inside a batch or between candidates, not just on entry.
					nth := 1
					if point == "candidate" || point == "visit" || point == "lstat" {
						nth = 3
					}
					if calls == nth {
						cancel()
					}
				}
				return nil
			}}
			result := s.auditSupplementalFilesWithOperations(ctx, location, ops)
			if !errors.Is(ctx.Err(), context.Canceled) || afterCancellation != 0 || len(result.files) != 0 || result.status != "incomplete" {
				t.Fatalf("cancelled work continued: calls=%d after=%d result=%+v", calls, afterCancellation, result)
			}
		})
	}
}

func TestSupplementalAuditEntryBudgetRequiresCompleteMainProof(t *testing.T) {
	for _, where := range []string{"parent", "extras"} {
		t.Run(where, func(t *testing.T) {
			s, id, main := supplementalFixture(t, "Movie/Movie.mkv")
			parent := filepath.Dir(main)
			supplementalWrite(t, filepath.Join(parent, "sample.mp4"))
			directory := parent
			if where == "extras" {
				directory = filepath.Join(parent, "Extras")
			}
			for i := 0; i < supplementalDirectoryEntryBudget+1; i++ {
				supplementalWrite(t, filepath.Join(directory, fmt.Sprintf("%03d.txt", i)))
			}
			result := s.auditSupplementalFiles(t.Context(), supplementalLocation(t, s, id))
			wantStatus, wantRows := "incomplete", 0
			if where == "extras" {
				wantStatus, wantRows = "truncated", 1
			}
			if result.status != wantStatus || len(result.files) != wantRows {
				t.Fatalf("limit proof: %+v", result)
			}
		})
	}
}
