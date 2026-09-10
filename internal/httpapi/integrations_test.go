package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/mediagrap/mediagrap/internal/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mediagrap/mediagrap/internal/auth"
	"github.com/mediagrap/mediagrap/internal/automation"
	"github.com/mediagrap/mediagrap/internal/httpapi"
	"github.com/mediagrap/mediagrap/internal/library"
	"github.com/mediagrap/mediagrap/internal/metadata"
	"github.com/mediagrap/mediagrap/internal/settings"
	"github.com/mediagrap/mediagrap/internal/tokens"
)

func TestApprovalRequiresAdminSessionAndCSRF(t *testing.T) {
	db := testDatabase(t)
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "movie.mkv"), []byte("media"), 0600)
	if _, err := db.Exec(`INSERT INTO sources(id,name,root_path) VALUES(1,'movies',?)`, root); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO media_items(id,source_id,relative_path,title_hint,file_size,modified_at) VALUES(1,1,'movie.mkv','Movie',5,'now')`); err != nil {
		t.Fatal(err)
	}
	authService := auth.NewService(db)
	session, err := authService.Setup(t.Context(), "admin", "strong-test-password")
	if err != nil {
		t.Fatal(err)
	}
	ts := tokens.New(db)
	created, err := ts.Create(t.Context(), session.User.ID, tokens.Input{Name: "writer", SourceIDs: []int64{1}, Scopes: []string{"media:read", "jobs:read", "plans:preview", "plans:apply"}, ExpiresAt: time.Now().Add(time.Hour).UnixMilli()})
	if err != nil {
		t.Fatal(err)
	}
	lib := library.NewService(db, []string{root})
	meta := metadata.NewService(db, nil)
	meta.Save(t.Context(), metadata.Record{MediaItemID: 1, Title: "Movie"})
	service := automation.New(db, ts, lib, meta)
	plan, err := service.Preview(t.Context(), created.Token, "write", automation.PreviewInput{MediaID: 1, IdempotencyKey: "preview-http"})
	if err != nil {
		t.Fatal(err)
	}
	mcpHandler := mcp.New(ts, lib, nil, slog.Default())
	mcpHandler.SetAutomation(service)
	server := httpapi.NewServer(slog.Default(), db, httpapi.BuildInfo{}, authService, lib, meta, settings.NewService(db, settings.Defaults{}), httpapi.RuntimePaths{Automation: service, MCP: mcpHandler})

	httpServer := httptest.NewServer(server)
	defer httpServer.Close()
	client := sdk.NewClient(&sdk.Implementation{Name: "integration", Version: "1.6.1"}, nil)
	clientSession, err := client.Connect(t.Context(), &sdk.StreamableClientTransport{Endpoint: httpServer.URL + "/mcp", HTTPClient: &http.Client{Transport: bearerTransport{created.Secret}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()
	discovered, err := clientSession.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range discovered.Tools {
		if strings.Contains(tool.Name, "approve") {
			t.Fatal("MCP self-approval exposed")
		}
	}
	params := &sdk.CallToolParams{Name: "apply_write_plan", Arguments: map[string]any{"planId": plan.ID, "version": 1, "digest": plan.Digest, "idempotencyKey": "http-apply-once"}}
	result, err := clientSession.CallTool(t.Context(), params)
	if err != nil || !result.IsError {
		t.Fatalf("unapproved call %+v %v", result, err)
	}
	body, _ := json.Marshal(automation.ApplyInput{Version: 1, Digest: plan.Digest})
	for _, scenario := range []struct {
		name         string
		cookie, csrf bool
		want         int
	}{{"bearer cannot approve", false, false, 401}, {"session needs CSRF", true, false, 403}, {"review approved", true, true, 204}} {
		t.Run(scenario.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/api/v1/automation/plans/"+plan.ID+"/approve", bytes.NewReader(body))
			r.Header.Set("Authorization", "Bearer "+created.Secret)
			if scenario.cookie {
				r.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: session.Token})
			}
			if scenario.csrf {
				r.Header.Set("X-CSRF-Token", session.CSRFToken)
			}
			w := httptest.NewRecorder()
			server.ServeHTTP(w, r)
			if w.Code != scenario.want {
				t.Fatalf("%d %s", w.Code, w.Body.String())
			}
		})
	}
	result, err = clientSession.CallTool(t.Context(), params)
	if err != nil || result.IsError {
		t.Fatalf("approved call %+v %v", result, err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { lib.RunWorker(ctx); close(done) }()
	defer func() { cancel(); <-done }()
	deadline := time.Now().Add(3 * time.Second)
	for {
		current, err := service.Plan(t.Context(), created.Token, plan.ID)
		if err != nil {
			t.Fatal(err)
		}
		if current.State == "applied" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("plan did not complete: %s", current.State)
		}
		time.Sleep(20 * time.Millisecond)
	}
	content, err := os.ReadFile(filepath.Join(root, "movie.nfo"))
	if err != nil || !strings.Contains(string(content), "<title>Movie</title>") {
		t.Fatalf("HTTP approved NFO %s %v", content, err)
	}

}
func TestIntegrationManagementIsNotBearerAuthenticated(t *testing.T) {
	server := testServer(testDatabase(t))
	for _, path := range []string{"/api/v1/webhooks", "/api/v1/api-tokens", "/api/v1/integrations/status"} {
		r := httptest.NewRequest("GET", path, nil)
		r.Header.Set("Authorization", "Bearer mgp_fake")
		w := httptest.NewRecorder()
		server.ServeHTTP(w, r)
		if w.Code != 401 {
			t.Fatalf("%s: %d", path, w.Code)
		}
	}
}

type bearerTransport struct{ secret string }

func (b bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.secret)
	return http.DefaultTransport.RoundTrip(r)
}
