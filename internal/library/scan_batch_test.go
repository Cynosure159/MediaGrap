package library

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mediagrap/mediagrap/internal/metadata"
	"github.com/mediagrap/mediagrap/internal/platform/database"
	"modernc.org/sqlite"
)

// The driver observer counts actual submitted statements and transactions (not
// inferred per-item estimates). It is instance-scoped and used only by tests.
type scanSQLCounts struct {
	mu                       sync.Mutex
	statements, transactions int
	writes                   map[string]int
}

func (c *scanSQLCounts) statement(query string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.statements++
	for _, table := range []string{"media_items", "sidecar_assets", "tv_shows", "tv_seasons", "tv_episodes", "media_metadata"} {
		q := strings.ToLower(strings.TrimSpace(query))
		if strings.HasPrefix(q, "insert into "+table+"(") || strings.HasPrefix(q, "insert or ignore into "+table+"(") || strings.HasPrefix(q, "delete from "+table+" ") || strings.HasPrefix(q, "update "+table+" ") {
			c.writes[table]++
		}
	}
}
func (c *scanSQLCounts) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.statements = 0
	c.transactions = 0
	c.writes = map[string]int{}
}
func (c *scanSQLCounts) snapshot() (int, int, map[string]int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	w := map[string]int{}
	for k, v := range c.writes {
		w[k] = v
	}
	return c.statements, c.transactions, w
}

type scanCountingDriver struct{ counts *scanSQLCounts }
type scanCountingConn struct {
	driver.Conn
	counts *scanSQLCounts
}

func (d scanCountingDriver) Open(name string) (driver.Conn, error) {
	c, e := (&sqlite.Driver{}).Open(name)
	if e != nil {
		return nil, e
	}
	return &scanCountingConn{c, d.counts}, nil
}
func (c *scanCountingConn) ExecContext(ctx context.Context, q string, a []driver.NamedValue) (driver.Result, error) {
	c.counts.statement(q)
	return c.Conn.(driver.ExecerContext).ExecContext(ctx, q, a)
}
func (c *scanCountingConn) QueryContext(ctx context.Context, q string, a []driver.NamedValue) (driver.Rows, error) {
	c.counts.statement(q)
	return c.Conn.(driver.QueryerContext).QueryContext(ctx, q, a)
}
func (c *scanCountingConn) BeginTx(ctx context.Context, o driver.TxOptions) (driver.Tx, error) {
	c.counts.mu.Lock()
	c.counts.transactions++
	c.counts.mu.Unlock()
	return c.Conn.(driver.ConnBeginTx).BeginTx(ctx, o)
}

var scanTestDriverID atomic.Int64

func scanBatchFixture(t testing.TB, count int) (*Service, Source, string, *scanSQLCounts) {
	t.Helper()
	root := t.TempDir()
	counts := &scanSQLCounts{}
	counts.reset()
	name := fmt.Sprintf("scan-count-%d", scanTestDriverID.Add(1))
	sql.Register(name, scanCountingDriver{counts})
	db, err := sql.Open(name, "file:"+filepath.ToSlash(filepath.Join(t.TempDir(), "scan.db"))+"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		t.Fatal(err)
	}
	// Same pool and durability settings as database.Open.
	db.SetMaxOpenConns(database.MaxOpenConns)
	db.SetMaxIdleConns(database.MaxOpenConns)
	t.Cleanup(func() { db.Close() })
	if err := database.Migrate(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	s := NewService(db, []string{root})
	s.SetLogger(slog.New(slog.NewTextHandler(io.Discard, nil)))
	source, err := s.CreateSource(t.Context(), "Fixture", root)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < count; i++ {
		snapshotFixture(t, root, fmt.Sprintf("Movie%05d.2024.mkv", i))
	}
	counts.reset()
	return s, source, root, counts
}

func TestScanBatchColdWarmAndOnePercentChange(t *testing.T) {
	s, source, root, counts := scanBatchFixture(t, 1000)
	run := func(label string, changed int) {
		t.Helper()
		counts.reset()
		if err := s.scan(t.Context(), 0, source.ID, "incremental", nil); err != nil {
			t.Fatal(err)
		}
		statements, transactions, writes := counts.snapshot()
		if transactions != 6 {
			t.Fatalf("%s: batch transactions=%d want 6", label, transactions)
		}
		if writes["sidecar_assets"] != changed {
			t.Fatalf("%s: sidecar rewrites=%d want %d", label, writes["sidecar_assets"], changed)
		}
		if label != "cold" && (writes["tv_shows"] != 0 || writes["tv_seasons"] != 0 || writes["tv_episodes"] != 0) {
			t.Fatalf("unchanged TV rewritten: %v", writes)
		}
		var active, seen int
		if err := s.db.QueryRow(`SELECT count(*),count(DISTINCT last_seen_at) FROM media_items WHERE missing=0`).Scan(&active, &seen); err != nil || active != 1000 || seen != 1 {
			t.Fatalf("seen tracking: %d %d %v", active, seen, err)
		}
		t.Logf("%s statements=%d transactions=%d writes=%v", label, statements, transactions, writes)
	}
	run("cold", 1000)
	run("warm", 0)
	for i := 0; i < 10; i++ {
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("Movie%05d.2024.mkv", i)), []byte("changed fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	run("one-percent-change", 10)
}

func TestScanBatchAvoidsRepeatedTVUpserts(t *testing.T) {
	s, source, root, counts := scanBatchFixture(t, 0)
	for i := 0; i < 450; i++ {
		snapshotFixture(t, root, fmt.Sprintf("Show (2024)/Season 01/Show.S01E%03d.mkv", i+1))
	}
	for _, mode := range []string{"incremental", "incremental", "full"} {
		counts.reset()
		if err := s.scan(t.Context(), 0, source.ID, mode, nil); err != nil {
			t.Fatal(err)
		}
		_, tx, w := counts.snapshot()
		if w["tv_shows"] > 1 || w["tv_seasons"] > 1 || tx != 4 {
			t.Fatalf("repeated TV upserts: %v tx=%d", w, tx)
		}
		shows, err := s.ListTVShows(t.Context(), "")
		if err != nil || len(shows) != 1 || shows[0].TitleHint != "Show" || shows[0].YearHint == nil || *shows[0].YearHint != 2024 || shows[0].EpisodeCount != 450 {
			t.Fatalf("shows=%+v err=%v", shows, err)
		}
	}
}

type scanHydratorSpy struct {
	service *metadata.Service
	calls   int
}

func (h *scanHydratorSpy) PrepareExistingNFO(ctx context.Context, id int64, path string) (metadata.Record, bool, error) {
	// This fixture contains one movie, so only one reader calls the spy.
	h.calls++
	return h.service.PrepareExistingNFO(ctx, id, path)
}
func (h *scanHydratorSpy) HydratePreparedNFO(ctx context.Context, record metadata.Record) error {
	return h.service.HydratePreparedNFO(ctx, record)
}
func TestScanBatchSidecarChangesAndMetadataPriority(t *testing.T) {
	s, source, root, counts := scanBatchFixture(t, 1)
	// A one-connection pool detects any hydration/FS callback inside a write tx.
	s.db.SetMaxOpenConns(1)
	h := &scanHydratorSpy{service: metadata.NewService(s.db, nil)}
	s.SetMetadataHydrator(h)
	scan := func() {
		t.Helper()
		if err := s.scan(t.Context(), 0, source.ID, "incremental", nil); err != nil {
			t.Fatal(err)
		}
	}
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	scan()
	if h.calls != 0 {
		t.Fatal("hydrated absent NFO")
	}
	write("movie.nfo", "<movie><title>Local title</title><year>2021</year></movie>")
	scan()
	var id int64
	if err := s.db.QueryRow(`SELECT id FROM media_items`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	record, err := h.service.Record(t.Context(), id)
	if err != nil || record.Title != "Local title" || record.Year == nil || *record.Year != 2021 {
		t.Fatalf("new NFO not hydrated: %+v %v", record, err)
	}
	before := h.calls
	counts.reset()
	scan()
	_, _, writes := counts.snapshot()
	if h.calls != before || writes["sidecar_assets"] != 0 || writes["media_metadata"] != 0 {
		t.Fatalf("warm rewrite/hydration: %v calls=%d", writes, h.calls)
	}
	// Metadata deletion must not leave an unchanged NFO permanently unhydrated.
	if _, err := s.db.Exec(`DELETE FROM media_metadata`); err != nil {
		t.Fatal(err)
	}
	scan()
	record, err = h.service.Record(t.Context(), id)
	if err != nil || record.Title != "Local title" {
		t.Fatal("unchanged NFO restoration failed", err)
	}
	for _, provider := range []string{"nfo", "tmdb"} {
		record.Provider = provider
		record.Title = "Manual or selected title"
		record.LockedFields = []string{"title"}
		if _, err := h.service.Save(t.Context(), record); err != nil {
			t.Fatal(err)
		}
		write("movie.nfo", "<movie><title>Changed external "+provider+"</title></movie>")
		scan()
		got, err := h.service.Record(t.Context(), id)
		if err != nil || got.Title != record.Title || got.Provider != provider || len(got.LockedFields) != 1 {
			t.Fatalf("saved priority lost: %+v %v", got, err)
		}
	}
	var previous string
	for _, action := range []string{"add", "modify", "delete"} {
		if err := s.db.QueryRow(`SELECT scan_fingerprint FROM media_items`).Scan(&previous); err != nil {
			t.Fatal(err)
		}
		switch action {
		case "add":
			write("poster.jpg", "image")
		case "modify":
			write("poster.jpg", "changed image")
		case "delete":
			if err := os.Remove(filepath.Join(root, "poster.jpg")); err != nil {
				t.Fatal(err)
			}
		}
		before = h.calls
		scan()
		var current string
		if err := s.db.QueryRow(`SELECT scan_fingerprint FROM media_items`).Scan(&current); err != nil || current == previous || h.calls != before {
			t.Fatalf("artwork %s fingerprint/hydration: %v calls=%d", action, err, h.calls)
		}
	}
	if err := os.Remove(filepath.Join(root, "movie.nfo")); err != nil {
		t.Fatal(err)
	}
	scan()
	assets, err := s.sidecars(t.Context(), id)
	if err != nil || len(assets) != 0 {
		t.Fatalf("deleted sidecars retained: %v %v", assets, err)
	}
	record, err = h.service.Record(t.Context(), id)
	if err != nil || record.Title != "Manual or selected title" {
		t.Fatal("NFO deletion erased saved metadata", err)
	}
	if _, err := s.db.Exec(`DELETE FROM media_metadata`); err != nil {
		t.Fatal(err)
	}
	write("movie.nfo", "invalid")
	scan()
	before = h.calls
	scan()
	if h.calls != before+1 {
		t.Fatal("failed hydration not retried")
	}
	write("movie.nfo", "<movie><title>Repaired</title></movie>")
	scan()
	record, err = h.service.Record(t.Context(), id)
	if err != nil || record.Title != "Repaired" {
		t.Fatalf("repair not hydrated: %+v %v", record, err)
	}
}

func TestScanBatchRollbackAndRecovery(t *testing.T) {
	s, source, root, _ := scanBatchFixture(t, 401)
	if _, err := s.db.Exec(`CREATE TRIGGER reject_scan BEFORE INSERT ON media_items WHEN NEW.relative_path='Movie00205.2024.mkv' BEGIN SELECT RAISE(ABORT,'injected batch failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.scan(t.Context(), 0, source.ID, "incremental", nil); err == nil {
		t.Fatal("injected transaction succeeded")
	}
	var count int
	if err := s.db.QueryRow(`SELECT count(*) FROM media_items`).Scan(&count); err != nil || count != 200 {
		t.Fatalf("batch failed to roll back: %d %v", count, err)
	}
	var completed sql.NullString
	if err := s.db.QueryRow(`SELECT last_scan_at FROM sources`).Scan(&completed); err != nil || completed.Valid {
		t.Fatal("failed scan completed", err)
	}
	if _, err := s.db.Exec(`DROP TRIGGER reject_scan`); err != nil {
		t.Fatal(err)
	}
	// A fresh service represents restart/retry: committed batches remain usable,
	// uncommitted work is replayed, and all unchanged rows receive the new epoch.
	restarted := NewService(s.db, []string{root})
	if err := restarted.scan(t.Context(), 0, source.ID, "incremental", nil); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`SELECT count(*) FROM media_items WHERE missing=0`).Scan(&count); err != nil || count != 401 {
		t.Fatalf("recovery lost rows: %d %v", count, err)
	}
	if err := s.db.QueryRow(`SELECT count(DISTINCT last_seen_at) FROM media_items`).Scan(&count); err != nil || count != 1 {
		t.Fatal("recovery skipped seen tracking", err)
	}
}

func TestScanBatchPartialTraversalNeverReconciles(t *testing.T) {
	for _, failure := range []string{"cancel", "read-error"} {
		t.Run(failure, func(t *testing.T) {
			s, source, root, _ := scanBatchFixture(t, 225)
			snapshotFixture(t, root, "ZFailure/Other.mkv")
			if err := s.scan(t.Context(), 0, source.ID, "incremental", nil); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(filepath.Join(root, "Movie00000.2024.mkv")); err != nil {
				t.Fatal(err)
			}
			if _, err := s.db.Exec(`UPDATE sources SET last_scan_at='old'`); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			injected := errors.New("injected traversal failure")
			s.readDirectory = func(path string) ([]fs.DirEntry, error) {
				if filepath.Base(path) == "ZFailure" {
					if failure == "cancel" {
						cancel()
					} else {
						return nil, injected
					}
				}
				return os.ReadDir(path)
			}
			if err := s.scan(ctx, 0, source.ID, "incremental", nil); err == nil {
				t.Fatal("failed traversal succeeded")
			}
			var hidden int
			var completed string
			if err := s.db.QueryRow(`SELECT count(*) FROM media_items WHERE missing=1`).Scan(&hidden); err != nil || hidden != 0 {
				t.Fatalf("partial scan hid rows: %d %v", hidden, err)
			}
			if err := s.db.QueryRow(`SELECT last_scan_at FROM sources`).Scan(&completed); err != nil || completed != "old" {
				t.Fatal("partial scan completed", err)
			}
			s.readDirectory = os.ReadDir
			if err := s.scan(t.Context(), 0, source.ID, "incremental", nil); err != nil {
				t.Fatal(err)
			}
			if err := s.db.QueryRow(`SELECT count(*) FROM media_items WHERE missing=1`).Scan(&hidden); err != nil || hidden != 1 {
				t.Fatal("retry did not reconcile", err)
			}
		})
	}
}

func TestScanBatchRejectsReplacedVideoAndOverLimit(t *testing.T) {
	s, source, root, _ := scanBatchFixture(t, 1)
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, entries[0].Name())
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "missing"), path); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareScanCandidate(root, path, entries[0], newDirectorySnapshot(entries)); err == nil {
		t.Fatal("replaced symlink indexed")
	}
	if err := (scanRepository{s.db}).applyBatch(t.Context(), source.ID, "seen", "incremental", make([]scanCandidate, scanBatchSize+1)); err == nil {
		t.Fatal("unbounded batch accepted")
	}
}

func TestScanBatchSidecarAndTVFailureRollBackTogether(t *testing.T) {
	for _, table := range []string{"sidecar_assets", "tv_episodes"} {
		t.Run(table, func(t *testing.T) {
			s, source, root, _ := scanBatchFixture(t, 1)
			snapshotFixture(t, root, "Show/Season 01/Show.S01E01.mkv")
			snapshotFixture(t, root, "Show/Season 01/Show.S01E01.nfo")
			if _, err := s.db.Exec(`CREATE TRIGGER reject_index BEFORE INSERT ON ` + table + ` BEGIN SELECT RAISE(ABORT,'injected index failure'); END`); err != nil {
				t.Fatal(err)
			}
			if err := s.scan(t.Context(), 0, source.ID, "incremental", nil); err == nil {
				t.Fatal("index error swallowed")
			}
			for _, name := range []string{"media_items", "sidecar_assets", "tv_shows", "tv_seasons", "tv_episodes"} {
				var count int
				if err := s.db.QueryRow(`SELECT count(*) FROM ` + name).Scan(&count); err != nil || count != 0 {
					t.Fatalf("%s not rolled back: %d %v", name, count, err)
				}
			}
			if _, err := s.db.Exec(`DROP TRIGGER reject_index`); err != nil {
				t.Fatal(err)
			}
			if err := s.scan(t.Context(), 0, source.ID, "incremental", nil); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestScanBatchFingerprintLimitAndFullRevalidation(t *testing.T) {
	s, source, root, counts := scanBatchFixture(t, 1)
	snapshotFixture(t, root, "movie.nfo")
	scan := func(mode string) {
		t.Helper()
		if err := s.scan(t.Context(), 0, source.ID, mode, nil); err != nil {
			t.Fatal(err)
		}
	}
	scan("incremental")
	path := filepath.Join(root, "movie.nfo")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// Explicitly demonstrate the documented stat-fingerprint blind spot.
	if err := os.WriteFile(path, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	counts.reset()
	scan("incremental")
	_, _, w := counts.snapshot()
	if w["sidecar_assets"] != 0 {
		t.Fatal("fixture did not preserve fingerprint")
	}
	counts.reset()
	scan("full")
	_, _, w = counts.snapshot()
	if w["sidecar_assets"] != 2 {
		t.Fatal("full scan failed to bypass fingerprint", w)
	}
}

func TestScanBatchInterruptedPersistedJobRecovery(t *testing.T) {
	s, source, root, _ := scanBatchFixture(t, 201)
	job, err := s.QueueScan(t.Context(), source.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Model a process lost after the first batch committed, before final success.
	if _, err := s.db.Exec(`CREATE TRIGGER stop_scan BEFORE INSERT ON media_items WHEN NEW.relative_path='Movie00200.2024.mkv' BEGIN SELECT RAISE(ABORT,'process interrupted'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.scan(t.Context(), job.ID, source.ID, "incremental", nil); err == nil {
		t.Fatal("interrupted fixture succeeded")
	}
	if _, err := s.db.Exec(`DROP TRIGGER stop_scan; UPDATE jobs SET state='running' WHERE id=?`, job.ID); err != nil {
		t.Fatal(err)
	}
	restarted := NewService(s.db, []string{root})
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { defer close(done); restarted.jobs.RunWorker(ctx) }()
	defer func() { cancel(); <-done }()
	waitState := func(want string) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			current, err := restarted.jobs.Get(t.Context(), job.ID)
			if err == nil && current.State == want {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("job did not reach %s", want)
	}
	waitState("interrupted")
	if _, err := restarted.RetryJob(t.Context(), job.ID); err != nil {
		t.Fatal(err)
	}
	waitState("succeeded")
	var count, epochs int
	if err := s.db.QueryRow(`SELECT count(*),count(DISTINCT last_seen_at) FROM media_items WHERE missing=0`).Scan(&count, &epochs); err != nil || count != 201 || epochs != 1 {
		t.Fatalf("persisted retry: count=%d epochs=%d err=%v", count, epochs, err)
	}
}

func TestScanBatchEpisodeSidecarOnlyChanges(t *testing.T) {
	s, source, root, counts := scanBatchFixture(t, 0)
	name := "Show/Season 01/Show.S01E01"
	snapshotFixture(t, root, name+".mkv")
	scan := func() {
		t.Helper()
		if err := s.scan(t.Context(), 0, source.ID, "incremental", nil); err != nil {
			t.Fatal(err)
		}
	}
	scan()
	for _, extension := range []string{".nfo", ".jpg"} {
		for _, operation := range []string{"add", "modify", "delete"} {
			path := filepath.Join(root, name+extension)
			if operation == "delete" {
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.WriteFile(path, []byte(operation), 0600); err != nil {
					t.Fatal(err)
				}
			}
			counts.reset()
			scan()
			_, _, w := counts.snapshot()
			if w["sidecar_assets"] == 0 || w["tv_episodes"] != 0 || w["tv_shows"] != 0 || w["tv_seasons"] != 0 {
				t.Fatalf("%s %s: inventory not refreshed or TV rewritten: %v", extension, operation, w)
			}
		}
	}
}

func TestScanBatchRootTVLastHintAcrossBatches(t *testing.T) {
	s, source, root, counts := scanBatchFixture(t, 0)
	for i := 1; i <= 225; i++ {
		title := "Alpha (2020)"
		if i > 200 {
			title = "Zulu (2024)"
		}
		snapshotFixture(t, root, fmt.Sprintf("%s S01E%03d.mkv", title, i))
	}
	scan := func(mode, title string, year int) {
		t.Helper()
		if err := s.scan(t.Context(), 0, source.ID, mode, nil); err != nil {
			t.Fatal(err)
		}
		shows, err := s.ListTVShows(t.Context(), "")
		if err != nil || len(shows) != 1 || shows[0].TitleHint != title || shows[0].YearHint == nil || *shows[0].YearHint != year {
			t.Fatalf("last root hint: %+v %v", shows, err)
		}
	}
	scan("incremental", "Zulu", 2024)
	for _, mode := range []string{"incremental", "full", "full"} {
		counts.reset()
		scan(mode, "Zulu", 2024)
		_, _, w := counts.snapshot()
		if w["tv_shows"] != 0 {
			t.Fatalf("stable %s oscillated root hints: %v", mode, w)
		}
	}
	for i := 201; i <= 225; i++ {
		if err := os.Remove(filepath.Join(root, fmt.Sprintf("Zulu (2024) S01E%03d.mkv", i))); err != nil {
			t.Fatal(err)
		}
	}
	scan("incremental", "Alpha", 2020)
	snapshotFixture(t, root, "ZZTop (2025) S01E226.mkv")
	scan("incremental", "ZZTop", 2025)
}

func TestScanBatchFullRepairsLegacyTVHintsWithoutChangingSavedTitle(t *testing.T) {
	s, source, root, counts := scanBatchFixture(t, 0)
	snapshotFixture(t, root, "Show (2024)/Season 01/Show.S01E01.mkv")
	if err := s.scan(t.Context(), 0, source.ID, "incremental", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`UPDATE tv_shows SET title_hint='stale',year_hint=1999; INSERT INTO tv_metadata(show_id,provider,title) SELECT id,'tmdb','Selected TV title' FROM tv_shows`); err != nil {
		t.Fatal(err)
	}
	counts.reset()
	if err := s.scan(t.Context(), 0, source.ID, "full", nil); err != nil {
		t.Fatal(err)
	}
	_, _, w := counts.snapshot()
	if w["tv_shows"] != 1 {
		t.Fatalf("hint repair writes=%v", w)
	}
	shows, err := s.ListTVShows(t.Context(), "")
	if err != nil || len(shows) != 1 || shows[0].TitleHint != "Show" || shows[0].YearHint == nil || *shows[0].YearHint != 2024 || shows[0].Title != "Selected TV title" {
		t.Fatalf("hint repair changed saved metadata: %+v %v", shows, err)
	}
}
