package artwork

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const MaxArtworkBytes int64 = 25 << 20 // 25 MiB safety limit

// ValidateTMDbImageURL verifies that the URL strictly targets https://image.tmdb.org/t/p/... without queries or userinfo.
func ValidateTMDbImageURL(value string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "https" || parsed.Host != "image.tmdb.org" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || !strings.HasPrefix(parsed.EscapedPath(), "/t/p/") {
		return errors.New("must be an HTTPS image.tmdb.org URL")
	}
	return nil
}

// DownloadJPEG fetches an image from sourceURL, validates JPEG mime and magic bytes, and writes to a temporary file in target directory.
func DownloadJPEG(ctx context.Context, client *http.Client, sourceURL string, targetDirectory string) (string, error) {
	if client == nil {
		return "", errors.New("artwork client is unavailable")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return "", errors.New("create artwork request")
	}
	response, err := client.Do(request)
	if err != nil {
		return "", errors.New("download artwork request failed")
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("artwork provider returned HTTP %d", response.StatusCode)
	}

	contentType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || contentType != "image/jpeg" {
		return "", errors.New("artwork provider did not return a JPEG image")
	}

	reader := bufio.NewReader(response.Body)
	header, err := reader.Peek(3)
	if err != nil || len(header) != 3 || header[0] != 0xff || header[1] != 0xd8 || header[2] != 0xff {
		return "", errors.New("artwork provider returned an invalid JPEG image")
	}

	temp, err := os.CreateTemp(targetDirectory, ".mediagrap-*.jpg")
	if err != nil {
		return "", err
	}
	tempName := temp.Name()

	written, copyErr := io.Copy(temp, io.LimitReader(reader, MaxArtworkBytes+1))
	if copyErr == nil && written > MaxArtworkBytes {
		copyErr = errors.New("artwork exceeds the 25 MiB safety limit")
	}
	if copyErr == nil {
		copyErr = temp.Sync()
	}
	if closeErr := temp.Close(); copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		_ = os.Remove(tempName)
		return "", fmt.Errorf("write artwork: %w", copyErr)
	}
	return tempName, nil
}
