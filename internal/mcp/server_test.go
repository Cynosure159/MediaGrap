package mcp

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mediagrap/mediagrap/internal/library"
	"github.com/mediagrap/mediagrap/internal/platform/database"
	"github.com/mediagrap/mediagrap/internal/tokens"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func setup(t *testing.T) (*Handler, *tokens.Service, tokens.Created, *sql.DB) {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "mcp.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err = database.Migrate(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{`INSERT INTO users(id,username,password_hash) VALUES(1,'admin','unused')`, `INSERT INTO sources(id,name,root_path) VALUES(1,'visible','/private/one'),(2,'hidden','/private/two')`, `INSERT INTO media_items(id,source_id,relative_path,title_hint,file_size,modified_at) VALUES(1,1,'secret/file.mkv','Visible movie',1,'now'),(2,2,'hidden.mkv','Hidden movie',1,'now')`, `INSERT INTO jobs(id,source_id,kind,state,payload,error_message) VALUES(1,1,'scan','failed','secret payload','secret error'),(2,2,'scan','succeeded','',''),(3,NULL,'scan','succeeded','','')`} {
		if _, err = db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	ts := tokens.New(db)
	token, err := ts.Create(t.Context(), 1, tokens.Input{Name: "test", SourceIDs: []int64{1}, ExpiresAt: time.Now().Add(time.Hour).UnixMilli()})
	if err != nil {
		t.Fatal(err)
	}
	return New(ts, library.NewService(db, nil), []string{"https://trusted.example"}, nil), ts, token, db
}

type authTransport struct {
	base   http.RoundTripper
	secret string
}

func (a authTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+a.secret)
	return a.base.RoundTrip(r)
}
func TestOfficialClientReadOnlyAndRevocation(t *testing.T) {
	h, ts, token, _ := setup(t)
	server := httptest.NewServer(h)
	defer server.Close()
	client := sdk.NewClient(&sdk.Implementation{Name: "MediaGrap integration test", Version: "1.6.1"}, nil)
	session, err := client.Connect(t.Context(), &sdk.StreamableClientTransport{Endpoint: server.URL, HTTPClient: &http.Client{Transport: authTransport{http.DefaultTransport, token.Secret}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	tools, err := session.ListTools(t.Context(), nil)
	if err != nil || len(tools.Tools) != 4 {
		t.Fatalf("tools %+v %v", tools, err)
	}
	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: "list_media", Arguments: map[string]any{}})
	if err != nil || result.IsError {
		t.Fatalf("call %+v %v", result, err)
	}
	body, _ := json.Marshal(result)
	if !bytes.Contains(body, []byte("Visible movie")) || bytes.Contains(body, []byte("Hidden")) || bytes.Contains(body, []byte("secret")) || bytes.Contains(body, []byte("/private")) {
		t.Fatalf("unsafe result %s", body)
	}
	result, err = session.CallTool(t.Context(), &sdk.CallToolParams{Name: "get_media", Arguments: map[string]any{"id": 2}})
	if err != nil || !result.IsError {
		t.Fatal("cross-source read allowed")
	}
	result, err = session.CallTool(t.Context(), &sdk.CallToolParams{Name: "list_jobs", Arguments: map[string]any{}})
	if err != nil || result.IsError {
		t.Fatal(err)
	}
	body, _ = json.Marshal(result)
	if bytes.Contains(body, []byte("secret")) || bytes.Contains(body, []byte(`"id":2`)) || bytes.Contains(body, []byte(`"id":3`)) {
		t.Fatalf("unsafe jobs %s", body)
	}
	result, err = session.CallTool(t.Context(), &sdk.CallToolParams{Name: "list_media", Arguments: map[string]any{"limit": 101}})
	if err != nil || !result.IsError {
		t.Fatal("page limit bypass")
	}
	if err = ts.Revoke(t.Context(), 1, token.Token.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = session.ListTools(t.Context(), nil); err == nil {
		t.Fatal("revoked token accepted")
	}
}
func request(h *Handler, token, body, origin, version string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "/mcp", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json, text/event-stream")
	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set("Origin", origin)
	r.Header.Set("MCP-Protocol-Version", version)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestProtocolBoundsAndAudit(t *testing.T) {
	h, _, token, db := setup(t)
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	if w := request(h, token.Secret, body, "https://evil.example", ProtocolVersion); w.Code != 403 {
		t.Fatal(w.Code)
	}
	if w := request(h, "wrong", body, "", ProtocolVersion); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w := request(h, token.Secret, body, "", "1999-01-01"); w.Code != 400 {
		t.Fatal(w.Code)
	}
	if w := request(h, token.Secret, strings.Repeat("x", 65537), "", ProtocolVersion); w.Code != 413 {
		t.Fatal(w.Code)
	}
	for range 61 {
		request(h, token.Secret, body, "", ProtocolVersion)
	}
	if w := request(h, token.Secret, body, "", ProtocolVersion); w.Code != 429 {
		t.Fatal(w.Code)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM mcp_audit`).Scan(&n); err != nil || n == 0 {
		t.Fatalf("audit %d %v", n, err)
	}
}
func TestTokenStorageScopeAndExpiry(t *testing.T) {
	_, ts, token, db := setup(t)
	var stored []byte
	if err := db.QueryRow(`SELECT token_hash FROM api_tokens`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if len(stored) != 32 || bytes.Contains(stored, []byte(token.Secret)) {
		t.Fatal("plaintext token stored")
	}
	if _, err := ts.Create(t.Context(), 1, tokens.Input{Name: "bad", SourceIDs: []int64{1}, Scopes: []string{"admin"}, ExpiresAt: time.Now().Add(time.Hour).UnixMilli()}); err == nil {
		t.Fatal("unknown admin scope granted")
	}
	db.Exec(`UPDATE api_tokens SET expires_at=0`)
	if _, err := ts.Authenticate(t.Context(), token.Secret); err == nil {
		t.Fatal("expired token accepted")
	}
}

func TestDeletedSourceGrantDoesNotFollowReusedID(t *testing.T) {
	_, ts, token, db := setup(t)
	if _, err := db.Exec(`DELETE FROM sources WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO sources(id,name,root_path) VALUES(1,'new source','/different-root')`); err != nil {
		t.Fatal(err)
	}
	current, err := ts.Authenticate(t.Context(), token.Secret)
	if err != nil {
		t.Fatal(err)
	}
	if current.Allows(1) {
		t.Fatal("deleted source grant followed reused ID")
	}
}
