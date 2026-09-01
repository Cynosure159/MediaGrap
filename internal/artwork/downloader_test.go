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

func TestFanartPreviewURLReplacesAssetPathPrefix(t *testing.T) {
	got, err := FanartPreviewURL("https://assets.fanart.tv/fanart/movies/550/poster/example.png")
	if err != nil {
		t.Fatal(err)
	}
	want := "https://assets.fanart.tv/preview/movies/550/poster/example.png"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	if _, err := FanartPreviewURL("https://assets.fanart.tv/poster/example.png"); err == nil {
		t.Fatal("expected non-fanart asset path to be rejected")
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

func TestDownloadPNG(t *testing.T) {
	dir := t.TempDir()
	validPNG := []byte("\x89PNG\r\n\x1a\n")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(validPNG)
	}))
	defer server.Close()
	tempPath, err := DownloadImage(t.Context(), server.Client(), server.URL+"/logo.png", dir, "image/png")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempPath)
	data, err := os.ReadFile(tempPath)
	if err != nil || string(data) != string(validPNG) {
		t.Fatalf("unexpected PNG content: %v", err)
	}
}
