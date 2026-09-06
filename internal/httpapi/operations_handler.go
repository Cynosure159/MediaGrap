package httpapi

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

type auditEntryView struct {
	ID             int64  `json:"id"`
	Action         string `json:"action"`
	MediaItemID    *int64 `json:"mediaItemId,omitempty"`
	ShowID         *int64 `json:"showId,omitempty"`
	Target         string `json:"target"`
	Detail         string `json:"detail"`
	Outcome        string `json:"outcome"`
	Backup         string `json:"backup"`
	Recoverability string `json:"recoverability"`
	CreatedAt      string `json:"createdAt"`
}

func (s *server) cancelJob(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, true); !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_job", "Invalid job id")
		return
	}
	job, err := s.library.CancelJob(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusConflict, "job_not_cancellable", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *server) retryJob(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, true); !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_job", "Invalid job id")
		return
	}
	job, err := s.library.RetryJob(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusConflict, "job_not_retryable", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}

func (s *server) streamJobEvents(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, false); !ok {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "stream_unavailable", "Event streaming is unavailable")
		return
	}
	cursor, _ := strconv.ParseInt(r.Header.Get("Last-Event-ID"), 10, 64)
	if queryCursor, err := strconv.ParseInt(r.URL.Query().Get("cursor"), 10, 64); err == nil && queryCursor > cursor {
		cursor = queryCursor
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	ticker := time.NewTicker(time.Second)
	keepAlive := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	defer keepAlive.Stop()
	for {
		events, err := s.library.JobEventsAfter(r.Context(), cursor, 100)
		if err != nil {
			return
		}
		for _, event := range events {
			payload, _ := json.Marshal(event)
			_, _ = fmt.Fprintf(w, "id: %d\nevent: job\ndata: %s\n\n", event.ID, payload)
			cursor = event.ID
		}
		if len(events) > 0 {
			flusher.Flush()
		}
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		case <-keepAlive.C:
			_, _ = fmt.Fprint(w, ": keep-alive\n\n")
			flusher.Flush()
		}
	}
}

func (s *server) listAuditEntries(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r, false); !ok {
		return
	}
	rows, err := s.db.QueryContext(r.Context(), `SELECT id,action,media_item_id,show_id,target_path,detail,outcome,backup_path,recoverability,created_at FROM audit_entries ORDER BY id DESC LIMIT 200`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Unable to list audit entries")
		return
	}
	defer rows.Close()
	items := make([]auditEntryView, 0)
	for rows.Next() {
		var item auditEntryView
		var mediaID, showID sql.NullInt64
		var target, backup string
		if err := rows.Scan(&item.ID, &item.Action, &mediaID, &showID, &target, &item.Detail, &item.Outcome, &backup, &item.Recoverability, &item.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "Unable to read audit entries")
			return
		}
		if mediaID.Valid {
			item.MediaItemID = &mediaID.Int64
		}
		if showID.Valid {
			item.ShowID = &showID.Int64
		}
		item.Target = safeAuditPath(target)
		item.Backup = safeAuditPath(backup)
		if strings.HasSuffix(item.Action, ".apply") {
			item.Outcome = "applied"
			item.Recoverability = "atomic_replace_only"
		} else if strings.HasSuffix(item.Action, ".preview") {
			item.Outcome = "previewed"
		}
		items = append(items, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func safeAuditPath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	parent := filepath.Base(filepath.Dir(filepath.Clean(path)))
	base := filepath.Base(filepath.Clean(path))
	if parent == "." || parent == string(filepath.Separator) {
		return base
	}
	return filepath.Join(parent, base)
}

func directoryUsage(path string) map[string]any {
	var total int64
	_ = filepath.WalkDir(path, func(_ string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		if info, infoErr := entry.Info(); infoErr == nil {
			total += info.Size()
		}
		return nil
	})
	info, err := os.Stat(path)
	available := err == nil && info.IsDir()
	writable := available && unix.Access(path, unix.W_OK) == nil
	return map[string]any{
		"path":      filepath.Base(filepath.Clean(path)),
		"available": available,
		"writable":  writable,
		"usedBytes": total,
	}
}

func (s *server) operationsStatus(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r, false)
	if !ok {
		return
	}
	ctx := r.Context()
	settingsView, err := s.settingsService.View(ctx, session.User.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Unable to load system status")
		return
	}
	sources, err := s.library.ListSources(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Unable to inspect media mounts")
		return
	}
	latestMigration := ""
	_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version),'') FROM schema_migrations`).Scan(&latestMigration)
	databaseBytes := int64(0)
	journalMode := "unknown"
	_ = s.db.QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&journalMode)
	if s.runtime.ConfigDir != "" {
		if info, statErr := os.Stat(filepath.Join(s.runtime.ConfigDir, "mediagrap.db")); statErr == nil {
			databaseBytes = info.Size()
		}
	}
	mounts := make([]map[string]any, 0, len(sources))
	for _, source := range sources {
		info, statErr := os.Stat(source.RootPath)
		available := statErr == nil && info.IsDir()
		mounts = append(mounts, map[string]any{
			"id": source.ID, "name": source.Name, "available": available,
			"writable": available && source.Writable, "itemCount": source.ItemCount, "scanMode": source.ScanMode,
			"scheduleEnabled": source.ScheduleEnabled, "nextScanAt": source.NextScanAt, "lastScanAt": source.LastScanAt,
		})
	}
	cache := map[string]any{"path": "cache", "available": false, "writable": false, "usedBytes": int64(0)}
	if s.runtime.CacheDir != "" {
		cache = directoryUsage(s.runtime.CacheDir)
	}
	tests := map[string]any{}
	rows, testErr := s.db.QueryContext(ctx, `SELECT target,status,COALESCE(http_status,0),duration_ms,message,tested_at FROM connection_test_results WHERE id IN (SELECT MAX(id) FROM connection_test_results GROUP BY target)`)
	if testErr == nil {
		defer rows.Close()
		for rows.Next() {
			var target, status, message, testedAt string
			var httpStatus int
			var duration int64
			if rows.Scan(&target, &status, &httpStatus, &duration, &message, &testedAt) == nil {
				tests[target] = map[string]any{"status": status, "httpStatus": httpStatus, "durationMs": duration, "message": message, "testedAt": testedAt}
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"application": map[string]string{"name": "MediaGrap", "version": s.build.Version, "commit": s.build.Commit, "builtAt": s.build.BuiltAt},
		"database":    map[string]any{"ready": s.db.PingContext(ctx) == nil, "latestMigration": latestMigration, "sizeBytes": databaseBytes, "walMode": strings.EqualFold(journalMode, "wal"), "journalMode": journalMode},
		"cache":       cache,
		"mounts":      mounts,
		"providers": []map[string]any{
			{"id": "tmdb", "configured": settingsView.TMDbAPIKeyConfigured, "status": configuredStatus(settingsView.TMDbAPIKeyConfigured)},
			{"id": "fanart_tv", "configured": settingsView.FanartTVAPIKeyConfigured, "status": configuredStatus(settingsView.FanartTVAPIKeyConfigured)},
		},
		"network":         map[string]any{"proxyConfigured": settingsView.OutboundProxyConfigured, "noProxyConfigured": settingsView.NoProxyConfigured},
		"connectionTests": tests,
	})
}

func configuredStatus(configured bool) string {
	if configured {
		return "configured"
	}
	return "not_configured"
}
