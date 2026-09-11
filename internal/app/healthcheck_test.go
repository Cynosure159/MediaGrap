package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckReadinessUsesRunningServer(t *testing.T) {
	for _, code := range []int{200, 503} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/readyz" {
					t.Error(r.URL.Path)
				}
				w.WriteHeader(code)
			}))
			defer server.Close()
			err := CheckReadiness(t.Context(), strings.TrimPrefix(server.URL, "http://"))
			if (err == nil) != (code == 200) {
				t.Fatalf("status=%d err=%v", code, err)
			}
		})
	}
}
