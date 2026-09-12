package httpapi_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mediagrap/mediagrap/internal/auth"
	"github.com/mediagrap/mediagrap/internal/httpapi"
	"github.com/mediagrap/mediagrap/internal/library"
	"github.com/mediagrap/mediagrap/internal/metadata"
	"github.com/mediagrap/mediagrap/internal/platform/database"
	"github.com/mediagrap/mediagrap/internal/settings"
)

type testContext struct {
	db          *sql.DB
	server      http.Handler
	authService *auth.Service
	mediaRoot   string
	session     *auth.Session
}

func setupTestContext(t *testing.T, runtime ...httpapi.RuntimePaths) *testContext {
	t.Helper()
	root := t.TempDir()
	db, err := database.Open(filepath.Join(t.TempDir(), "api_test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.Migrate(t.Context(), db); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	authSvc := auth.NewService(db)
	libSvc := library.NewService(db, []string{root})
	metaSvc := metadata.NewService(db, metadata.NewTMDb(slog.Default(), http.DefaultClient, ""))
	setSvc := settings.NewService(db, settings.Defaults{MediaRoots: []string{root}})

	srv := httpapi.NewServer(
		slog.Default(),
		db,
		httpapi.BuildInfo{Version: "test-v1", Commit: "abcdef1", BuiltAt: "2026-09-01"},
		authSvc,
		libSvc,
		metaSvc,
		setSvc,
		runtime...,
	)

	return &testContext{
		db:          db,
		server:      srv,
		authService: authSvc,
		mediaRoot:   root,
	}
}

func (tc *testContext) createAdminSession(t *testing.T) {
	t.Helper()
	session, err := tc.authService.Setup(context.Background(), "admin", "strongadminpassword123")
	if err != nil {
		t.Fatalf("setup admin session: %v", err)
	}
	tc.session = session
}

func (tc *testContext) doRequest(req *http.Request, useAuth, useCSRF bool) *httptest.ResponseRecorder {
	if useAuth && tc.session != nil {
		req.AddCookie(&http.Cookie{
			Name:  auth.SessionCookieName,
			Value: tc.session.Token,
		})
	}
	if useCSRF && tc.session != nil {
		req.Header.Set("X-CSRF-Token", tc.session.CSRFToken)
	}
	rec := httptest.NewRecorder()
	tc.server.ServeHTTP(rec, req)
	return rec
}

func TestReadyEndpoint(t *testing.T) {
	tc := setupTestContext(t)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := tc.doRequest(req, false, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSystemSummaryEndpoint(t *testing.T) {
	tc := setupTestContext(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/summary", nil)
	rec := tc.doRequest(req, false, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestOperationsStatusRequiresAuthentication(t *testing.T) {
	tc := setupTestContext(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/operations/status", nil)
	rec := tc.doRequest(req, false, false)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
	tc.createAdminSession(t)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/operations/status", nil)
	rec = tc.doRequest(req, true, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSettingsPreferencesSourcePolicyAndRedactedConnectionTest(t *testing.T) {
	tc := setupTestContext(t)
	tc.createAdminSession(t)
	body := bytes.NewBufferString(`{"fallbackLanguage":"zh-CN","theme":"light","locale":"zh-CN","noProxy":"localhost,.lan"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", body)
	req.Header.Set("Content-Type", "application/json")
	rec := tc.doRequest(req, true, true)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"theme":"light"`) || !strings.Contains(rec.Body.String(), `"noProxyConfigured":true`) {
		t.Fatalf("unexpected settings response %d: %s", rec.Code, rec.Body.String())
	}

	source, err := tc.db.ExecContext(t.Context(), `INSERT INTO sources(name,root_path) VALUES(?,?)`, "Media", tc.mediaRoot)
	if err != nil {
		t.Fatal(err)
	}
	sourceID, _ := source.LastInsertId()
	body = bytes.NewBufferString(`{"scanMode":"full","scheduleEnabled":true,"scheduleIntervalMinutes":60}`)
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/sources/%d/policy", sourceID), body)
	req.Header.Set("Content-Type", "application/json")
	rec = tc.doRequest(req, true, true)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"scanMode":"full"`) || !strings.Contains(rec.Body.String(), `"scheduleEnabled":true`) {
		t.Fatalf("unexpected source policy response %d: %s", rec.Code, rec.Body.String())
	}

	body = bytes.NewBufferString(`{"target":"tmdb"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/settings/connection-tests", body)
	req.Header.Set("Content-Type", "application/json")
	rec = tc.doRequest(req, true, true)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"not_configured"`) {
		t.Fatalf("unexpected connection test response %d: %s", rec.Code, rec.Body.String())
	}
}

func TestJobCancelRetryAndAuditEndpoints(t *testing.T) {
	tc := setupTestContext(t)
	tc.createAdminSession(t)
	jobResult, err := tc.db.ExecContext(t.Context(), `INSERT INTO jobs(kind,state,payload) VALUES('scan','queued','{}')`)
	if err != nil {
		t.Fatal(err)
	}
	jobID, _ := jobResult.LastInsertId()
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/jobs/%d/cancel", jobID), nil)
	rec := tc.doRequest(req, true, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("cancel status %d: %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/jobs/%d/retry", jobID), nil)
	rec = tc.doRequest(req, true, true)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("retry status %d: %s", rec.Code, rec.Body.String())
	}
	_, err = tc.db.ExecContext(t.Context(), `INSERT INTO audit_entries(action,target_path,detail) VALUES('nfo.apply','/media/Movie/Movie.nfo','Applied successfully')`)
	if err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/audit-entries", nil)
	rec = tc.doRequest(req, true, false)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"target":"Movie/Movie.nfo"`) {
		t.Fatalf("unexpected audit response %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMovieInspectionAndNamingPreviewEndpointsAreReadOnly(t *testing.T) {
	tc := setupTestContext(t)
	tc.createAdminSession(t)
	mediaPath := filepath.Join(tc.mediaRoot, "Example.2024.mkv")
	if err := os.WriteFile(mediaPath, []byte("not a real video"), 0o600); err != nil {
		t.Fatal(err)
	}
	source, err := tc.db.ExecContext(t.Context(), `INSERT INTO sources(name,root_path) VALUES(?,?)`, "Movies", tc.mediaRoot)
	if err != nil {
		t.Fatal(err)
	}
	sourceID, _ := source.LastInsertId()
	item, err := tc.db.ExecContext(t.Context(), `INSERT INTO media_items(source_id,relative_path,title_hint,year_hint,file_size,modified_at) VALUES(?,?,?,?,?,?)`, sourceID, filepath.Base(mediaPath), "Example", 2024, 16, "2024-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	itemID, _ := item.LastInsertId()

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media/%d/inspection", itemID), nil)
	rec := tc.doRequest(req, true, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected inspection status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var inspection library.MediaInspection
	if err := json.Unmarshal(rec.Body.Bytes(), &inspection); err != nil || len(inspection.Files) != 1 {
		t.Fatalf("unexpected inspection response: %#v err=%v", inspection, err)
	}

	body := bytes.NewBufferString(`{"pattern":"${title} (${year})"}`)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/media/%d/naming-preview", itemID), body)
	req.Header.Set("Content-Type", "application/json")
	rec = tc.doRequest(req, true, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected naming preview status 200 without CSRF, got %d: %s", rec.Code, rec.Body.String())
	}
	var preview library.NamingPreview
	if err := json.Unmarshal(rec.Body.Bytes(), &preview); err != nil || !preview.ReadOnly || len(preview.Items) != 1 {
		t.Fatalf("unexpected naming preview: %#v err=%v", preview, err)
	}
	if _, err := os.Stat(mediaPath); err != nil {
		t.Fatalf("read-only endpoints changed the media file: %v", err)
	}
}

func TestSetupAndSessionFlow(t *testing.T) {
	tc := setupTestContext(t)

	// 1. Initial setup status: needsSetup = true
	req := httptest.NewRequest(http.MethodGet, "/api/v1/setup/status", nil)
	rec := tc.doRequest(req, false, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var statusRes map[string]bool
	_ = json.Unmarshal(rec.Body.Bytes(), &statusRes)
	if !statusRes["needsSetup"] {
		t.Fatal("expected needsSetup=true initially")
	}

	// 2. Perform setup
	setupPayload, _ := json.Marshal(map[string]string{
		"username": "superadmin",
		"password": "supersecurepassword123",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/setup", bytes.NewReader(setupPayload))
	req.Header.Set("Content-Type", "application/json")
	rec = tc.doRequest(req, false, false)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 on setup, got %d: %s", rec.Code, rec.Body.String())
	}

	// Extract session cookie from setup response
	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.SessionCookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected session cookie in setup response")
	}

	// 3. Status after setup: needsSetup = false
	req = httptest.NewRequest(http.MethodGet, "/api/v1/setup/status", nil)
	rec = tc.doRequest(req, false, false)
	_ = json.Unmarshal(rec.Body.Bytes(), &statusRes)
	if statusRes["needsSetup"] {
		t.Fatal("expected needsSetup=false after setup")
	}

	// 4. Duplicate setup returns 409
	req = httptest.NewRequest(http.MethodPost, "/api/v1/setup", bytes.NewReader(setupPayload))
	req.Header.Set("Content-Type", "application/json")
	rec = tc.doRequest(req, false, false)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409 on duplicate setup, got %d", rec.Code)
	}

	// 5. Test session with cookie
	req = httptest.NewRequest(http.MethodGet, "/api/v1/session", nil)
	req.AddCookie(sessionCookie)
	rec = tc.doRequest(req, false, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on /session with cookie, got %d", rec.Code)
	}
	var sessionRes struct {
		User      map[string]any `json:"user"`
		CSRFToken string         `json:"csrfToken"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &sessionRes)
	if sessionRes.CSRFToken == "" {
		t.Fatal("expected CSRF token in session response")
	}

	// 6. Login with correct credentials
	loginPayload, _ := json.Marshal(map[string]string{
		"username": "superadmin",
		"password": "supersecurepassword123",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/session", bytes.NewReader(loginPayload))
	req.Header.Set("Content-Type", "application/json")
	rec = tc.doRequest(req, false, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on login, got %d: %s", rec.Code, rec.Body.String())
	}

	// 7. Login with invalid password returns 401
	badLoginPayload, _ := json.Marshal(map[string]string{
		"username": "superadmin",
		"password": "wrongpassword12345",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/session", bytes.NewReader(badLoginPayload))
	req.Header.Set("Content-Type", "application/json")
	rec = tc.doRequest(req, false, false)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 on bad login, got %d", rec.Code)
	}

	// 8. Logout without CSRF returns 403
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/session", nil)
	req.AddCookie(sessionCookie)
	rec = tc.doRequest(req, false, false)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 on logout without CSRF, got %d", rec.Code)
	}

	// 9. Logout with CSRF returns 204
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/session", nil)
	req.AddCookie(sessionCookie)
	req.Header.Set("X-CSRF-Token", sessionRes.CSRFToken)
	rec = tc.doRequest(req, false, false)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204 on valid logout, got %d", rec.Code)
	}
}

func TestSourcesHandlers(t *testing.T) {
	tc := setupTestContext(t)
	tc.createAdminSession(t)

	// 1. List sources (initially empty)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sources", nil)
	rec := tc.doRequest(req, true, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Create source without CSRF -> 403
	body, _ := json.Marshal(map[string]string{"name": "TestMovies", "rootPath": tc.mediaRoot})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/sources", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = tc.doRequest(req, true, false)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 without CSRF, got %d", rec.Code)
	}

	// 3. Create source with CSRF -> 201
	req = httptest.NewRequest(http.MethodPost, "/api/v1/sources", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = tc.doRequest(req, true, true)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var createdSource struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &createdSource)
	if createdSource.ID == 0 {
		t.Fatal("expected non-zero source ID")
	}

	// 4. Trigger scan on source
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/sources/%d/scans", createdSource.ID), nil)
	rec = tc.doRequest(req, true, true)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status 202 on scan trigger, got %d: %s", rec.Code, rec.Body.String())
	}

	// 5. Delete source with CSRF -> 204
	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/sources/%d", createdSource.ID), nil)
	rec = tc.doRequest(req, true, true)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204 on delete, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMediaAndPlansHandlers(t *testing.T) {
	tc := setupTestContext(t)
	tc.createAdminSession(t)

	// Create movie file and seed DB
	movieFile := filepath.Join(tc.mediaRoot, "Inception.2010.mkv")
	_ = os.WriteFile(movieFile, []byte("dummy video"), 0o600)

	srcRes, _ := tc.db.Exec(`INSERT INTO sources(name, root_path) VALUES('Movies', ?)`, tc.mediaRoot)
	srcID, _ := srcRes.LastInsertId()
	itemRes, _ := tc.db.Exec(`INSERT INTO media_items(source_id, relative_path, title_hint, year_hint, file_size, modified_at) VALUES(?, 'Inception.2010.mkv', 'Inception', 2010, 1024, '2026-01-01T00:00:00Z')`, srcID)
	itemID, _ := itemRes.LastInsertId()

	// 1. List media
	req := httptest.NewRequest(http.MethodGet, "/api/v1/media", nil)
	rec := tc.doRequest(req, true, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on list media, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Get media detail
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media/%d", itemID), nil)
	rec = tc.doRequest(req, true, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on get media, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. Preview write plan
	planReqBody, _ := json.Marshal(map[string]any{
		"title":  "Inception",
		"genres": []string{"Action", "Sci-Fi"},
	})
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/media/%d/write-plans", itemID), bytes.NewReader(planReqBody))
	req.Header.Set("Content-Type", "application/json")
	rec = tc.doRequest(req, true, true)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 on preview write-plan, got %d: %s", rec.Code, rec.Body.String())
	}
	var planRes struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &planRes)
	if planRes.ID == "" {
		t.Fatal("expected write plan ID")
	}

	// 4. Get write plan
	req = httptest.NewRequest(http.MethodGet, "/api/v1/write-plans/"+planRes.ID, nil)
	rec = tc.doRequest(req, true, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on get write-plan, got %d", rec.Code)
	}

	// 5. Apply write plan
	req = httptest.NewRequest(http.MethodPost, "/api/v1/write-plans/"+planRes.ID+"/apply", nil)
	rec = tc.doRequest(req, true, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on apply write-plan, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestTVShowsHandlers(t *testing.T) {
	tc := setupTestContext(t)
	tc.createAdminSession(t)

	// Create TV show fixture in DB
	showDir := filepath.Join(tc.mediaRoot, "Breaking Bad (2008)", "Season 01")
	_ = os.MkdirAll(showDir, 0o750)
	_ = os.WriteFile(filepath.Join(showDir, "Breaking.Bad.S01E01.mkv"), []byte("dummy ep"), 0o600)

	srcRes, err := tc.db.Exec(`INSERT INTO sources(name, root_path) VALUES('TV', ?)`, tc.mediaRoot)
	if err != nil {
		t.Fatalf("insert source: %v", err)
	}
	srcID, _ := srcRes.LastInsertId()
	showRes, err := tc.db.Exec(`INSERT INTO tv_shows(source_id, relative_path, title_hint, year_hint) VALUES(?, 'Breaking Bad (2008)', 'Breaking Bad', 2008)`, srcID)
	if err != nil {
		t.Fatalf("insert tv_show: %v", err)
	}
	showID, _ := showRes.LastInsertId()

	seasonRes, err := tc.db.Exec(`INSERT INTO tv_seasons(show_id, season_number) VALUES(?, 1)`, showID)
	if err != nil {
		t.Fatalf("insert tv_season: %v", err)
	}
	seasonID, _ := seasonRes.LastInsertId()

	itemRes, err := tc.db.Exec(`INSERT INTO media_items(source_id, relative_path, title_hint, file_size, modified_at) VALUES(?, 'Breaking Bad (2008)/Season 01/Breaking.Bad.S01E01.mkv', 'Pilot', 1024, '2026-01-01T00:00:00Z')`, srcID)
	if err != nil {
		t.Fatalf("insert episode media_item: %v", err)
	}
	epItemID, _ := itemRes.LastInsertId()

	_, err = tc.db.Exec(`INSERT INTO tv_episodes(media_item_id, show_id, season_id, season_number, episode_start, episode_end, title_hint) VALUES(?, ?, ?, 1, 1, 1, 'Pilot')`, epItemID, showID, seasonID)
	if err != nil {
		t.Fatalf("insert tv_episode: %v", err)
	}

	// 1. List TV shows
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tv/shows", nil)
	rec := tc.doRequest(req, true, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on list tv shows, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Get TV show detail
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/tv/shows/%d", showID), nil)
	rec = tc.doRequest(req, true, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on get tv show, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. Get nonexistent TV show -> 404
	req = httptest.NewRequest(http.MethodGet, "/api/v1/tv/shows/99999", nil)
	rec = tc.doRequest(req, true, false)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for missing show, got %d", rec.Code)
	}
}

func TestSettingsAndJobsHandlers(t *testing.T) {
	tc := setupTestContext(t)
	tc.createAdminSession(t)

	// 1. Get Settings
	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	rec := tc.doRequest(req, true, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on get settings, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Update Settings with CSRF
	updateBody, _ := json.Marshal(map[string]any{
		"tmdbLanguage": "zh-CN",
	})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	rec = tc.doRequest(req, true, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on update settings, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. List Jobs
	req = httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
	rec = tc.doRequest(req, true, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on list jobs, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUnauthenticatedEndpointsReturn401(t *testing.T) {
	tc := setupTestContext(t)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/session"},
		{http.MethodGet, "/api/v1/sources"},
		{http.MethodGet, "/api/v1/media"},
		{http.MethodGet, "/api/v1/tv/shows"},
		{http.MethodGet, "/api/v1/settings"},
		{http.MethodGet, "/api/v1/jobs"},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			req := httptest.NewRequest(ep.method, ep.path, nil)
			rec := tc.doRequest(req, false, false)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected status 401, got %d for %s", rec.Code, ep.path)
			}
		})
	}
}
