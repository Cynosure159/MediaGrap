package httpapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mediagrap/mediagrap/internal/library"
	"github.com/mediagrap/mediagrap/internal/metadata"
)

func TestMovieDetailFallsBackWithoutChangingBadNFO(t *testing.T) {
	for _, test := range []struct {
		name, content string
		bad           bool
	}{
		{"empty", "", true}, {"whitespace", " \n\t", true},
		{"truncated", "<movie><title>Partial", true}, {"no-title", "<movie></movie>", true},
		{"oversize", strings.Repeat("x", (1<<20)+1), true},
		{"valid", "<movie><title>Blonde</title><year>2022</year></movie>", false},
		{"missing", "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			tc := setupTestContext(t)
			tc.createAdminSession(t)
			source, err := library.NewService(tc.db, []string{tc.mediaRoot}).CreateSource(t.Context(), "Movies", tc.mediaRoot)
			if err != nil {
				t.Fatal(err)
			}
			sourceID := source.ID
			item, err := tc.db.Exec(`INSERT INTO media_items(source_id,relative_path,title_hint,year_hint,file_size,modified_at) VALUES(?,'Blonde.2022.mkv','Blonde',2022,1,'now')`, sourceID)
			if err != nil {
				t.Fatal(err)
			}
			id, err := item.LastInsertId()
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(filepath.Join(tc.mediaRoot, "Blonde.2022.mkv"), []byte("fixture"), 0600); err != nil {
				t.Fatal(err)
			}
			nfoPath := filepath.Join(tc.mediaRoot, "Blonde.2022.nfo")
			if test.name != "missing" {
				if err = os.WriteFile(nfoPath, []byte(test.content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			response := tc.doRequest(httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/media/%d", id), nil), true, false)
			if response.Code != 200 {
				t.Fatalf("%d %s", response.Code, response.Body.String())
			}
			var detail struct {
				Item     library.MediaItem `json:"item"`
				Metadata metadata.Record   `json:"metadata"`
				Origin   string            `json:"metadataOrigin"`
				Warning  string            `json:"metadataWarning"`
			}
			if err = json.Unmarshal(response.Body.Bytes(), &detail); err != nil {
				t.Fatal(err)
			}
			if detail.Item.ID != id || detail.Item.TitleHint != "Blonde" || detail.Item.YearHint == nil || *detail.Item.YearHint != 2022 {
				t.Fatalf("wrong identity: %+v", detail.Item)
			}
			if test.bad && (detail.Warning != "invalid_nfo" || detail.Origin != "empty" || detail.Metadata.Title != "") {
				t.Fatalf("bad fallback: %+v", detail)
			}
			if !test.bad && detail.Warning != "" {
				t.Fatal(detail.Warning)
			}
			if test.name == "valid" && (detail.Metadata.Title != "Blonde" || detail.Origin != "nfo") {
				t.Fatal("valid NFO not hydrated")
			}
			if test.name != "missing" {
				after, err := os.ReadFile(nfoPath)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(after, []byte(test.content)) {
					t.Fatal("NFO changed")
				}
			}
			if test.bad {
				var count int
				if err = tc.db.QueryRow(`SELECT count(*) FROM media_metadata WHERE media_item_id=?`, id).Scan(&count); err != nil {
					t.Fatal(err)
				}
				if count != 0 {
					t.Fatal("fallback persisted invalid metadata")
				}
			}
		})
	}
}
