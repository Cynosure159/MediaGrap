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

func ValidateFanartImageURL(value string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "https" || parsed.Host != "assets.fanart.tv" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path == "" {
		return errors.New("must be an HTTPS assets.fanart.tv URL")
	}
	return nil
}

// FanartPreviewURL converts a Fanart.tv asset URL to the provider's preview
// endpoint. Fanart.tv expects /preview/ to replace /fanart/, not /preview to
// be appended to the original asset path.
func FanartPreviewURL(value string) (string, error) {
	if err := ValidateFanartImageURL(value); err != nil {
		return "", err
	}
	parsed, _ := url.Parse(strings.TrimSpace(value))
	const fanartPrefix = "/fanart/"
	if !strings.HasPrefix(parsed.Path, fanartPrefix) {
		return "", errors.New("must be a Fanart.tv asset URL")
	}
	parsed.Path = "/preview/" + strings.TrimPrefix(parsed.Path, fanartPrefix)
	parsed.RawPath = ""
	return parsed.String(), nil
}

// DownloadJPEG fetches an image from sourceURL, validates JPEG mime and magic bytes, and writes to a temporary file in target directory.
func DownloadJPEG(ctx context.Context, client *http.Client, sourceURL string, targetDirectory string) (string, error) {
	temp, err := DownloadImage(ctx, client, sourceURL, targetDirectory, "image/jpeg")
	if err != nil {
		message := err.Error()
		message = strings.Replace(message, "did not return a JPEG or PNG image", "did not return a JPEG image", 1)
		message = strings.Replace(message, "returned an invalid image", "returned an invalid JPEG image", 1)
		return "", errors.New(message)
	}
	return temp, nil
}

// DownloadImage accepts only JPEG or PNG responses, validates both the
// provider-declared MIME and file signature, and stages the result beside the
// target so the caller can atomically rename it.
func DownloadImage(ctx context.Context, client *http.Client, sourceURL string, targetDirectory string, expectedMIME string) (string, error) {
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
	if response.Request != nil && response.Request.URL.String() != request.URL.String() {
		return "", errors.New("artwork provider redirect is not allowed")
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("artwork provider returned HTTP %d", response.StatusCode)
	}

	contentType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || (contentType != "image/jpeg" && contentType != "image/png") {
		return "", errors.New("artwork provider did not return a JPEG or PNG image")
	}
	if expectedMIME != "" && expectedMIME != contentType {
		return "", fmt.Errorf("artwork provider returned %s, expected %s", contentType, expectedMIME)
	}

	reader := bufio.NewReader(response.Body)
	header, err := reader.Peek(3)
	validJPEG := len(header) >= 3 && header[0] == 0xff && header[1] == 0xd8 && header[2] == 0xff
	validPNG := false
	if contentType == "image/png" {
		pngHeader, pngErr := reader.Peek(8)
		validPNG = pngErr == nil && string(pngHeader) == "\x89PNG\r\n\x1a\n"
	}
	if (err != nil && contentType == "image/jpeg") || (contentType == "image/jpeg" && !validJPEG) || (contentType == "image/png" && !validPNG) {
		return "", errors.New("artwork provider returned an invalid image")
	}

	extension := ".jpg"
	if contentType == "image/png" {
		extension = ".png"
	}
	temp, err := os.CreateTemp(targetDirectory, ".mediagrap-*"+extension)
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
