package settings

import (
	"github.com/mediagrap/mediagrap/internal/renamepattern"
)

const DefaultMovieRenamePattern = "${title} (${year})/${title} (${year})"
const DefaultTVRenamePattern = "${showTitle}/Season ${seasonNumber}/${showTitle} - S${seasonNumberPad}E${episodeNumberPad} - ${episodeTitle}"

func validateRenamePattern(pattern string, tv bool) error {
	allowed := renamepattern.Allowed("originalTitle", "year", "resolution", "videoCodec", "audioCodec")
	if tv {
		for _, token := range []string{"showTitle", "showOriginalTitle", "seasonNumber", "seasonNumberPad", "episodeNumber", "episodeNumberPad", "episodeTitle"} {
			allowed[token] = struct{}{}
		}
	} else {
		allowed["title"] = struct{}{}
		allowed["edition"] = struct{}{}
		allowed["imdbId"] = struct{}{}
	}
	return renamepattern.Validate(pattern, allowed)
}
