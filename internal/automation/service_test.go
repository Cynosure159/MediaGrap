package automation

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/mediagrap/mediagrap/internal/library"
	"github.com/mediagrap/mediagrap/internal/metadata"
	"github.com/mediagrap/mediagrap/internal/platform/database"
	"github.com/mediagrap/mediagrap/internal/tokens"
)

func fixture(t *testing.T) (*Service, tokens.Token, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "movie.mkv"), []byte("valuable video"), 0600); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(filepath.Join(t.TempDir(), "automation.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err = database.Migrate(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{`INSERT INTO users(id,username,password_hash) VALUES(1,'admin','unused')`, `INSERT INTO sources(id,name,root_path) VALUES(1,'movies',?)`, `INSERT INTO media_items(id,source_id,relative_path,title_hint,file_size,modified_at) VALUES(1,1,'movie.mkv','Movie',14,'now')`} {
		var args []any
		if strings.Contains(q, "?") {
			args = []any{root}
		}
		if _, err = db.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	ts := tokens.New(db)
	created, err := ts.Create(t.Context(), 1, tokens.Input{Name: "automation", SourceIDs: []int64{1}, Scopes: []string{"media:read", "metadata:read", "jobs:read", "jobs:write", "metadata:write", "plans:preview", "plans:apply"}, ExpiresAt: time.Now().Add(time.Hour).UnixMilli()})
	if err != nil {
		t.Fatal(err)
	}
	lib := library.NewService(db, []string{root})
	lib.SetFFprobePath("")
	meta := metadata.NewService(db, nil)
	if _, err = meta.Save(t.Context(), metadata.Record{MediaItemID: 1, Title: "Approved title"}); err != nil {
		t.Fatal(err)
	}
	return New(db, ts, lib, meta), created.Token, root
}
func preview(t *testing.T, s *Service, p tokens.Token) Plan {
	t.Helper()
	plan, err := s.Preview(t.Context(), p, "write", PreviewInput{MediaID: 1, IdempotencyKey: "preview-1"})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}
func approveAndQueue(t *testing.T, s *Service, p tokens.Token, plan Plan) Queued {
	t.Helper()
	in := ApplyInput{PlanID: plan.ID, Version: 1, Digest: plan.Digest, IdempotencyKey: "apply-01"}
	if err := s.Approve(t.Context(), 1, in); err != nil {
		t.Fatal(err)
	}
	queued, err := s.Apply(t.Context(), p, plan.Kind, in)
	if err != nil {
		t.Fatal(err)
	}
	return queued
}
func execute(t *testing.T, s *Service, q Queued) error {
	t.Helper()
	job, err := s.library.Job(t.Context(), q.JobID)
	if err != nil {
		t.Fatal(err)
	}
	return s.runPlan(t.Context(), job, func(int, string) {})
}
func TestApprovalBoundToTokenDigestAndOneJob(t *testing.T) {
	s, p, root := fixture(t)
	plan := preview(t, s, p)
	if _, err := os.Stat(filepath.Join(root, "movie.nfo")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("preview wrote file")
	}
	in := ApplyInput{PlanID: plan.ID, Version: 1, Digest: plan.Digest, IdempotencyKey: "apply-01"}
	if _, err := s.Apply(t.Context(), p, "write", in); err == nil {
		t.Fatal("unapproved apply allowed")
	}
	forged := in
	forged.Digest = "wrong"
	if err := s.Approve(t.Context(), 1, forged); err == nil {
		t.Fatal("changed digest approved")
	}
	q := approveAndQueue(t, s, p, plan)
	again, err := s.Apply(t.Context(), p, "write", in)
	if err != nil || again.JobID != q.JobID {
		t.Fatalf("idempotency: %+v %v", again, err)
	}
	other := p
	other.ID = "different"
	if _, err = s.Apply(t.Context(), other, "write", in); err == nil {
		t.Fatal("cross-token approval reuse")
	}
	if err = execute(t, s, q); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(root, "movie.nfo"))
	if err != nil || !strings.Contains(string(content), "Approved title") {
		t.Fatalf("NFO %s %v", content, err)
	}
	if err = execute(t, s, q); err == nil {
		t.Fatal("re-executed applied plan")
	}
	current, err := s.Plan(t.Context(), p, plan.ID)
	if err != nil || current.State != "applied" || current.Items[0].State != "done" {
		t.Fatalf("plan %+v %v", current, err)
	}
	var count int
	s.db.QueryRow(`SELECT COUNT(*) FROM system_events WHERE event_type='write_plan.applied'`).Scan(&count)
	if count != 1 {
		t.Fatal("applied event missing or duplicated")
	}
}
func TestPlanRejectsChangedFileExpiryAndRevocation(t *testing.T) {
	for _, scenario := range []string{"changed", "metadata_changed", "expired", "revoked", "symlink"} {
		t.Run(scenario, func(t *testing.T) {
			s, p, root := fixture(t)
			plan := preview(t, s, p)
			if scenario == "expired" {
				if err := s.Approve(t.Context(), 1, ApplyInput{PlanID: plan.ID, Version: 1, Digest: plan.Digest}); err != nil {
					t.Fatal(err)
				}
				s.db.Exec(`UPDATE automation_plans SET approval_expires_at=0`)
				if _, err := s.Apply(t.Context(), p, "write", ApplyInput{PlanID: plan.ID, Version: 1, Digest: plan.Digest, IdempotencyKey: "apply-01"}); err == nil {
					t.Fatal("expired approval applied")
				}
				return
			}
			q := approveAndQueue(t, s, p, plan)
			switch scenario {
			case "metadata_changed":
				s.metadata.Save(t.Context(), metadata.Record{MediaItemID: 1, Title: "Changed after approval"})
			case "changed":
				os.WriteFile(filepath.Join(root, "movie.nfo"), []byte("external edit"), 0600)
			case "revoked":
				s.tokens.Revoke(t.Context(), 1, p.ID)
			case "symlink":
				outside := filepath.Join(t.TempDir(), "precious.nfo")
				os.WriteFile(outside, []byte("untouched"), 0600)
				os.Symlink(outside, filepath.Join(root, "movie.nfo"))
			}
			if err := execute(t, s, q); err == nil {
				t.Fatal("unsafe plan executed")
			}
			if scenario == "changed" {
				value, _ := os.ReadFile(filepath.Join(root, "movie.nfo"))
				if string(value) != "external edit" {
					t.Fatal("external content overwritten")
				}
			}
		})
	}
}
func TestRecoverPublishedWriteAfterDatabaseFailure(t *testing.T) {
	s, p, root := fixture(t)
	plan := preview(t, s, p)
	q := approveAndQueue(t, s, p, plan)
	if _, err := s.db.Exec(`CREATE TRIGGER fail_result BEFORE UPDATE ON automation_operations WHEN NEW.state='done' BEGIN SELECT RAISE(ABORT,'injected'); END`); err != nil {
		t.Fatal(err)
	}
	if err := execute(t, s, q); err == nil {
		t.Fatal("expected injected failure")
	}
	before, err := os.ReadFile(filepath.Join(root, "movie.nfo"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`DROP TRIGGER fail_result`); err != nil {
		t.Fatal(err)
	}
	if err = s.Recover(t.Context()); err != nil {
		t.Fatal(err)
	}
	plan, err = s.Plan(t.Context(), p, plan.ID)
	if err != nil || plan.State != "applied" {
		t.Fatalf("recovery %+v %v", plan, err)
	}
	after, _ := os.ReadFile(filepath.Join(root, "movie.nfo"))
	if string(before) != string(after) {
		t.Fatal("recovery rewrote file")
	}
	if err = s.Recover(t.Context()); err != nil {
		t.Fatal(err)
	}
	var count int
	s.db.QueryRow(`SELECT COUNT(*) FROM system_events`).Scan(&count)
	if count != 1 {
		t.Fatal("recovery duplicated event")
	}
}
func TestRecoveryNeverOverwritesAmbiguousTarget(t *testing.T) {
	s, p, root := fixture(t)
	plan := preview(t, s, p)
	q := approveAndQueue(t, s, p, plan)
	s.db.Exec(`UPDATE automation_plans SET state='running' WHERE id=?`, plan.ID)
	s.db.Exec(`UPDATE automation_operations SET state='staged',expected_hash='wrong' WHERE plan_id=?`, plan.ID)
	os.WriteFile(filepath.Join(root, "movie.nfo"), []byte("other writer"), 0600)
	if err := s.Recover(t.Context()); err != nil {
		t.Fatal(err)
	}
	current, _ := s.Plan(t.Context(), p, plan.ID)
	if current.State != "needs_review" {
		t.Fatal(current.State)
	}
	if err := execute(t, s, q); err == nil {
		t.Fatal("recovery silently resumed")
	}
	value, _ := os.ReadFile(filepath.Join(root, "movie.nfo"))
	if string(value) != "other writer" {
		t.Fatal("recovery modified file")
	}
}
func TestConcurrentApplyQueuesOnce(t *testing.T) {
	s, p, _ := fixture(t)
	plan := preview(t, s, p)
	in := ApplyInput{PlanID: plan.ID, Version: 1, Digest: plan.Digest, IdempotencyKey: "concurrent-apply"}
	if err := s.Approve(t.Context(), 1, in); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan Queued, 4)
	errs := make(chan error, 4)
	for range 4 {
		wg.Go(func() { q, err := s.Apply(t.Context(), p, "write", in); results <- q; errs <- err })
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var id int64
	for q := range results {
		if id != 0 && q.JobID != id {
			t.Fatal("multiple jobs")
		}
		id = q.JobID
	}
}
func TestRenamePlanMovesOnlyPreviewedFiles(t *testing.T) {
	s, p, root := fixture(t)
	os.WriteFile(filepath.Join(root, "movie.nfo"), []byte("sidecar"), 0600)
	plan, err := s.Preview(t.Context(), p, "rename", PreviewInput{MediaID: 1, Pattern: "Renamed", IdempotencyKey: "rename-1"})
	if err != nil {
		t.Fatal(err)
	}
	q := approveAndQueue(t, s, p, plan)
	if err = execute(t, s, q); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "Renamed.mkv"))
	if err != nil || string(data) != "valuable video" {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(root, "movie.mkv")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("source remained")
	}
	var relative string
	s.db.QueryRow(`SELECT relative_path FROM media_items WHERE id=1`).Scan(&relative)
	if relative != "Renamed.mkv" {
		t.Fatal(relative)
	}
}
func TestTaskIdempotencyAndReadOnlyGrant(t *testing.T) {
	s, p, root := fixture(t)
	readOnly := p
	readOnly.Scopes = []string{"media:read"}
	input := TaskInput{SourceID: 1, IdempotencyKey: "scan-request"}
	if _, err := s.QueueTask(t.Context(), readOnly, "scan_source", input); err == nil {
		t.Fatal("read-only token queued task")
	}
	q, err := s.QueueTask(t.Context(), p, "scan_source", input)
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.QueueTask(t.Context(), p, "scan_source", input)
	if err != nil || q.JobID != again.JobID {
		t.Fatal("duplicate task")
	}
	job, err := s.library.Job(t.Context(), q.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.runTask(t.Context(), job, func(int, string) {}); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(root, "movie.nfo")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("scan wrote NFO")
	}
}
func TestRootedPathValidation(t *testing.T) {
	rootPath := t.TempDir()
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	outside := t.TempDir()
	os.Symlink(outside, filepath.Join(rootPath, "escape"))
	for _, path := range []string{"../escape", "/absolute", "a/../b", "escape/file.nfo"} {
		if _, err := inspect(root, path, true); err == nil {
			t.Fatalf("accepted %s", path)
		}
	}
}

func TestVerifiedCrossDeviceMove(t *testing.T) {
	s, p, root := fixture(t)
	s.linkSource = func(*os.Root, string, string) error { return syscall.EXDEV }
	plan, err := s.Preview(t.Context(), p, "rename", PreviewInput{MediaID: 1, Pattern: "New Directory/Renamed", IdempotencyKey: "cross-device"})
	if err != nil {
		t.Fatal(err)
	}
	q := approveAndQueue(t, s, p, plan)
	if err = execute(t, s, q); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "New Directory", "Renamed.mkv"))
	if err != nil || string(data) != "valuable video" {
		t.Fatalf("copy %s %v", data, err)
	}
	if _, err = os.Stat(filepath.Join(root, "movie.mkv")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("source removed incorrectly")
	}
}
func TestPreviewIsIdempotentAndChangedArgumentsConflict(t *testing.T) {
	s, p, _ := fixture(t)
	first := preview(t, s, p)
	second := preview(t, s, p)
	if first.ID != second.ID || first.Digest != second.Digest {
		t.Fatal("preview changed")
	}
	if _, err := s.Preview(t.Context(), p, "write", PreviewInput{MediaID: 1, Pattern: "changed", IdempotencyKey: "preview-1"}); err == nil {
		t.Fatal("changed arguments accepted")
	}
}

type artworkTransport func(*http.Request) (*http.Response, error)

func (f artworkTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestArtworkPlanPublishesValidatedCandidate(t *testing.T) {
	s, p, root := fixture(t)
	image := []byte{0xff, 0xd8, 0xff, 0x00, 0x01}
	client := &http.Client{Transport: artworkTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "image.tmdb.org" {
			t.Fatal("unexpected provider")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"image/jpeg"}}, Body: io.NopCloser(bytes.NewReader(image)), Request: r}, nil
	})}
	s.metadata = metadata.NewService(s.db, metadata.NewTMDb(slog.Default(), client, "test"))
	candidate := metadata.ArtworkCandidate{ID: "candidate-1", MediaItemID: 1, Provider: "tmdb", ProviderAssetID: "a", Kind: "poster", SourceURL: "https://image.tmdb.org/t/p/original/a.jpg", MimeType: "image/jpeg"}
	if err := metadata.NewRepository(s.db).ReplaceArtworkCandidates(t.Context(), []metadata.ArtworkCandidate{candidate}); err != nil {
		t.Fatal(err)
	}
	plan, err := s.Preview(t.Context(), p, "artwork", PreviewInput{MediaID: 1, IdempotencyKey: "artwork-preview", Selections: []metadata.ArtworkSelection{{Kind: "poster", CandidateID: candidate.ID}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(root, "poster.jpg")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("preview wrote image")
	}
	q := approveAndQueue(t, s, p, plan)
	if err = execute(t, s, q); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "poster.jpg"))
	if err != nil || !bytes.Equal(data, image) {
		t.Fatalf("image %v %v", data, err)
	}
}
