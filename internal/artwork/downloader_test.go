package artwork

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestValidateTMDbImageURL(t *testing.T) {
	tests := []struct {
		url     string
		wantErr bool
	}{
		{"https://image.tmdb.org/t/p/original/abc12345.jpg", false},
		{"https://image.tmdb.org/t/p/w500/test.jpg", false},
		{"http://image.tmdb.org/t/p/original/test.jpg", true},
		{"https://evil.com/t/p/test.jpg", true},
		{"https://image.tmdb.org/t/p/test.jpg?query=bad", true},
		{"https://user:pass@image.tmdb.org/t/p/test.jpg", true},
		{"https://image.tmdb.org/other/path.jpg", true},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			err := ValidateTMDbImageURL(tc.url)
			if (err != nil) != tc.wantErr {
				t.Fatalf("url=%q: got err=%v, wantErr=%v", tc.url, err, tc.wantErr)
			}
		})
	}
}

func TestDownloadJPEG(t *testing.T) {
	dir := t.TempDir()

	// 1. Success: valid JPEG
	validJPEG := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46, 0x49, 0x46}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(validJPEG)
	}))
	defer ts.Close()

	tempPath, err := DownloadJPEG(t.Context(), ts.Client(), ts.URL+"/image.jpg", dir)
	if err != nil {
		t.Fatalf("DownloadJPEG failed: %v", err)
	}
	defer os.Remove(tempPath)

	data, err := os.ReadFile(tempPath)
	if err != nil || len(data) != len(validJPEG) {
		t.Fatalf("unexpected content downloaded: %v (len %d)", err, len(data))
	}

	// 2. Failure: invalid Content-Type
	tsText := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("not an image"))
	}))
	defer tsText.Close()

	_, err = DownloadJPEG(t.Context(), tsText.Client(), tsText.URL+"/image.jpg", dir)
	if err == nil || !strings.Contains(err.Error(), "did not return a JPEG image") {
		t.Fatalf("expected JPEG error, got %v", err)
	}

	// 3. Failure: invalid magic bytes
	tsFake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("PNG fake image bytes"))
	}))
	defer tsFake.Close()

	_, err = DownloadJPEG(t.Context(), tsFake.Client(), tsFake.URL+"/image.jpg", dir)
	if err == nil || !strings.Contains(err.Error(), "invalid JPEG image") {
		t.Fatalf("expected invalid JPEG image error, got %v", err)
	}
}
