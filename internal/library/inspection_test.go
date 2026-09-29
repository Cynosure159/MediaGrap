package library

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mediagrap/mediagrap/internal/platform/database"
	renameSettings "github.com/mediagrap/mediagrap/internal/settings"
)

type fakeMediaProber struct {
	calls int
}

func (p *fakeMediaProber) Probe(context.Context, string) (MediaInspection, error) {
	p.calls++
	return MediaInspection{
		ProbeStatus: "ready",
		Format:      ProbeFormat{Name: "matroska", DurationSeconds: 120, BitRate: 1_000_000},
		Video:       []VideoStream{{Index: 0, Codec: "hevc", Width: 3840, Height: 2160, BitDepth: 10, HDR: "HDR10"}},
		Audio:       []AudioStream{{Index: 1, Codec: "eac3", Channels: 6, ChannelLayout: "5.1", Language: "eng"}},
		Subtitles:   []SubtitleStream{{Index: 2, Codec: "subrip", Language: "zho"}},
	}, nil
}

func inspectionFixture(t *testing.T) (*Service, int64, string) {
	t.Helper()
	root := t.TempDir()
	db, err := database.Open(filepath.Join(t.TempDir(), "inspection.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.Migrate(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	result, err := db.ExecContext(t.Context(), `INSERT INTO sources(name,root_path) VALUES(?,?)`, "Movies", root)
	if err != nil {
		t.Fatal(err)
	}
	sourceID, _ := result.LastInsertId()
	mediaPath := filepath.Join(root, "Old.Movie.2024.mkv")
	if err := os.WriteFile(mediaPath, []byte("video fixture"), 0o640); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(mediaPath)
	result, err = db.ExecContext(t.Context(), `INSERT INTO media_items(source_id,relative_path,title_hint,year_hint,file_size,modified_at) VALUES(?,?,?,?,?,?)`, sourceID, filepath.Base(mediaPath), "Old Movie", 2024, info.Size(), info.ModTime().UTC().Format("2006-01-02T15:04:05Z07:00"))
	if err != nil {
		t.Fatal(err)
	}
	itemID, _ := result.LastInsertId()
	return NewService(db, []string{root}), itemID, mediaPath
}

func TestParseFFprobeMapsVideoAudioSubtitleAndHDR(t *testing.T) {
	raw := []byte(`{"format":{"format_name":"matroska","duration":"123.5","bit_rate":"8000000"},"streams":[{"index":0,"codec_type":"video","codec_name":"hevc","profile":"Main 10","width":3840,"height":2160,"pix_fmt":"yuv420p10le","color_transfer":"smpte2084"},{"index":1,"codec_type":"audio","codec_name":"eac3","channels":6,"channel_layout":"5.1","tags":{"language":"eng","title":"Main"}},{"index":2,"codec_type":"subtitle","codec_name":"subrip","tags":{"LANGUAGE":"zho"}}]}`)
	inspection, err := parseFFprobe(raw)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Format.DurationSeconds != 123.5 || len(inspection.Video) != 1 || inspection.Video[0].HDR != "HDR10" || inspection.Video[0].BitDepth != 10 {
		t.Fatalf("unexpected video mapping: %#v", inspection)
	}
	if len(inspection.Audio) != 1 || inspection.Audio[0].Language != "eng" || len(inspection.Subtitles) != 1 || inspection.Subtitles[0].Language != "zho" {
		t.Fatalf("unexpected stream mapping: %#v", inspection)
	}
}

func TestInspectMediaCachesProbeByFileFingerprintAndAuditsSymlinks(t *testing.T) {
	service, itemID, mediaPath := inspectionFixture(t)
	prober := &fakeMediaProber{}
	service.prober = prober
	if err := os.WriteFile(filepath.Join(filepath.Dir(mediaPath), "Old.Movie.2024.nfo"), []byte("<movie><title>Old Movie</title></movie>"), 0o440); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("missing-poster.jpg", filepath.Join(filepath.Dir(mediaPath), "Old.Movie.2024.poster.jpg")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(mediaPath), "Old.Movie.2024.bad.jpg"), []byte("not an image"), 0o600); err != nil {
		t.Fatal(err)
	}

	first, err := service.InspectMedia(t.Context(), itemID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.InspectMedia(t.Context(), itemID)
	if err != nil {
		t.Fatal(err)
	}
	if prober.calls != 1 || first.Cached || !second.Cached {
		t.Fatalf("expected one probe and a cache hit, calls=%d first.cached=%t second.cached=%t", prober.calls, first.Cached, second.Cached)
	}
	var foundReadOnlyNFO, foundSymlink, foundInvalidMIME bool
	for _, file := range second.Files {
		if file.Kind == "nfo" && !file.Writable && file.Valid {
			foundReadOnlyNFO = true
		}
		if file.Symlink && !file.Valid {
			foundSymlink = true
		}
		if file.Kind == "image" && !file.Valid {
			for _, warning := range file.Warnings {
				foundInvalidMIME = foundInvalidMIME || warning == "invalid_mime"
			}
		}
	}
	if !foundReadOnlyNFO || !foundSymlink || !foundInvalidMIME {
		t.Fatalf("expected real permission and symlink warnings: %#v", second.Files)
	}
}

func TestPreviewNamingIsReadOnlyAndDetectsConflicts(t *testing.T) {
	service, itemID, mediaPath := inspectionFixture(t)
	service.prober = &fakeMediaProber{}
	sidecarPath := filepath.Join(filepath.Dir(mediaPath), "Old.Movie.2024.en.srt")
	if err := os.WriteFile(sidecarPath, []byte("subtitle"), 0o600); err != nil {
		t.Fatal(err)
	}
	conflictPath := filepath.Join(filepath.Dir(mediaPath), "New Movie (2024).mkv")
	if err := os.WriteFile(conflictPath, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	year := 2024
	preview, err := service.PreviewNaming(t.Context(), itemID, "${title} (${year})", NamingValues{Title: "New Movie", Year: &year})
	if err != nil {
		t.Fatal(err)
	}
	if !preview.ReadOnly || preview.Items[0].Operation != "conflict" || !preview.Items[0].Conflict {
		t.Fatalf("unexpected preview: %#v", preview)
	}
	if _, err := os.Stat(mediaPath); err != nil {
		t.Fatalf("dry-run changed the media file: %v", err)
	}
	if _, err := os.Stat(sidecarPath); err != nil {
		t.Fatalf("dry-run changed the sidecar: %v", err)
	}
	if _, err := service.PreviewNaming(t.Context(), itemID, "../${title}", NamingValues{Title: "Unsafe"}); err == nil {
		t.Fatal("expected path-producing pattern to be rejected")
	}
	optional, err := service.PreviewNaming(t.Context(), itemID, "${title}${ [,resolution,]}", NamingValues{Title: "New Movie", Year: &year})
	if err != nil || optional.Items[0].PlannedPath != "New Movie [2160p].mkv" {
		t.Fatalf("optional legacy preview=%+v err=%v", optional, err)
	}
	if _, err := service.PreviewNaming(t.Context(), itemID, "${,edition,}", NamingValues{Title: "New Movie"}); err == nil {
		t.Fatal("legacy preview accepted unsupported optional token")
	}
}

func TestPreviewRenamePlanOptionalPreservesExtensionAndDetectsCollision(t *testing.T) {
	service, itemID, mediaPath := inspectionFixture(t)
	service.prober = &fakeMediaProber{}
	sidecarPath := filepath.Join(filepath.Dir(mediaPath), "Old.Movie.2024.nfo")
	if err := os.WriteFile(sidecarPath, []byte("<movie/>"), 0o600); err != nil {
		t.Fatal(err)
	}
	pattern := "${title}${ [,resolution,]}"
	plan, err := service.PreviewRenamePlan(t.Context(), itemID, pattern)
	if err != nil || plan.HasConflicts {
		t.Fatalf("optional movie plan=%+v err=%v", plan, err)
	}
	var video, nfo RenamePlanItem
	for _, item := range plan.Items {
		if item.Kind == "video" {
			video = item
		}
		if item.Kind == "nfo" {
			nfo = item
		}
	}
	if video.PlannedPath != "Old Movie [2160p].mkv" || nfo.PlannedPath != "Old Movie [2160p].nfo" {
		t.Fatalf("optional movie targets lost extension/sidecar suffix: video=%+v nfo=%+v", video, nfo)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(mediaPath), video.PlannedPath), []byte("collision"), 0o600); err != nil {
		t.Fatal(err)
	}
	collision, err := service.PreviewRenamePlan(t.Context(), itemID, pattern)
	if err != nil || !collision.HasConflicts {
		t.Fatalf("expected optional collision, plan=%+v err=%v", collision, err)
	}
	settingsService := renameSettings.NewService(service.db, renameSettings.Defaults{})
	newPattern := "${title} (${year})"
	if _, err := settingsService.Update(t.Context(), renameSettings.Update{MovieRenamePattern: &newPattern}); err != nil {
		t.Fatal(err)
	}
	frozen, err := service.GetRenamePlan(t.Context(), plan.ID)
	if err != nil || frozen.Pattern != pattern || frozen.HasConflicts {
		t.Fatalf("first plan was changed by settings save or later preview: %+v err=%v", frozen, err)
	}
}

func TestResolvedParentWithinRootRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "escaped")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if resolvedParentWithinRoot(root, filepath.Join(link, "Movie.mkv")) {
		t.Fatal("expected a parent symlink escaping the source root to be rejected")
	}
}
