package settings

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

const DefaultMovieRenamePattern = "${title} (${year})/${title} (${year})"
const DefaultTVRenamePattern = "${showTitle}/Season ${seasonNumber}/${showTitle} - S${seasonNumberPad}E${episodeNumberPad} - ${episodeTitle}"

var renameToken = regexp.MustCompile(`\$\{([A-Za-z][A-Za-z0-9]*)\}`)

func validateRenamePattern(pattern string, tv bool) error {
	if pattern == "" || len(pattern) > 1024 {
		return errors.New("rename pattern must contain 1–1024 bytes")
	}
	if strings.ContainsAny(pattern, `\:*?"<>|`) || strings.IndexFunc(pattern, unicode.IsControl) >= 0 || strings.HasPrefix(pattern, "/") {
		return errors.New("rename pattern must be a safe relative path")
	}
	for _, part := range strings.Split(pattern, "/") {
		if part == "" || part == "." || part == ".." {
			return errors.New("rename pattern contains an invalid path segment")
		}
	}
	allowed := " originalTitle year resolution videoCodec audioCodec "
	if tv {
		allowed += "showTitle showOriginalTitle seasonNumber seasonNumberPad episodeNumber episodeNumberPad episodeTitle "
	} else {
		allowed += "title edition imdbId "
	}
	for _, token := range renameToken.FindAllStringSubmatch(pattern, -1) {
		if !strings.Contains(allowed, " "+token[1]+" ") {
			return fmt.Errorf("unsupported naming token %q", token[1])
		}
	}
	if strings.ContainsAny(renameToken.ReplaceAllString(pattern, "value"), "${}") {
		return errors.New("malformed naming token")
	}
	return nil
}
