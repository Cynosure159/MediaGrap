package library

import (
	"strings"
	"testing"
)

func TestRename_SanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal", "My Movie", "My Movie"},
		{"unsafe chars", "My: Movie <1999>", "My Movie 1999"},
		{"trailing dots and spaces", " Movie. ", "Movie"},
		{"multiple spaces", "My   Movie", "My Movie"},
		{"slashes", "My/Movie\\", "My Movie"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestRename_ResolutionLabel(t *testing.T) {
	tests := []struct {
		width, height int
		expected      string
	}{
		{3840, 2160, "2160p"},
		{1920, 1080, "1080p"},
		{1280, 720, "720p"},
		{720, 480, "480p"},
		{0, 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := resolutionLabel(tt.width, tt.height)
			if got != tt.expected {
				t.Errorf("resolutionLabel(%d, %d) = %q; want %q", tt.width, tt.height, got, tt.expected)
			}
		})
	}
}

func TestRename_NamingTokenPattern(t *testing.T) {
	tokens := map[string]string{
		"title":         "The Matrix",
		"year":          "1999",
		"resolution":    "1080p",
		"videoCodec":    "H264",
		"originalTitle": "",
	}

	tests := []struct {
		name           string
		pattern        string
		expectedStem   string
		expectedUnknown string
		expectedUnavail string
	}{
		{
			name:         "all available",
			pattern:      "${title} (${year}) - ${resolution}",
			expectedStem: "The Matrix (1999) - 1080p",
		},
		{
			name:            "unknown token",
			pattern:         "${title} - ${fake}",
			expectedUnknown: "fake",
		},
		{
			name:            "unavailable token",
			pattern:         "${title} (${originalTitle})",
			expectedUnavail: "originalTitle",
		},
		{
			name:         "directory pattern",
			pattern:      "${title} (${year})/${title} - ${resolution}",
			expectedStem: "The Matrix (1999)/The Matrix - 1080p",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unknown := ""
			unavailable := ""
			stem := namingTokenPattern.ReplaceAllStringFunc(tt.pattern, func(token string) string {
				name := namingTokenPattern.FindStringSubmatch(token)[1]
				value, ok := tokens[name]
				if !ok {
					unknown = name
					return token
				}
				if value == "" {
					unavailable = name
				}
				return value
			})

			if unknown != tt.expectedUnknown {
				t.Errorf("expected unknown token %q, got %q", tt.expectedUnknown, unknown)
			}
			if unavailable != tt.expectedUnavail {
				t.Errorf("expected unavailable token %q, got %q", tt.expectedUnavail, unavailable)
			}
			if unknown == "" && unavailable == "" && stem != tt.expectedStem {
				t.Errorf("expected stem %q, got %q", tt.expectedStem, stem)
			}
		})
	}
}

func TestRename_DirectoryTemplateSplitting(t *testing.T) {
	tests := []struct {
		name         string
		stem         string
		expectedDir  string
		expectedFile string
	}{
		{"no directory", "The Matrix (1999)", "", "The Matrix (1999)"},
		{"with directory", "The Matrix (1999)/The Matrix (1999) - 1080p", "The Matrix (1999)", "The Matrix (1999) - 1080p"},
		{"nested directory", "Movies/The Matrix (1999)/The Matrix (1999)", "Movies/The Matrix (1999)", "The Matrix (1999)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dirTemplate := ""
			fileTemplate := tt.stem
			if idx := strings.LastIndex(tt.stem, "/"); idx != -1 {
				dirTemplate = tt.stem[:idx]
				fileTemplate = tt.stem[idx+1:]
			}

			if dirTemplate != tt.expectedDir {
				t.Errorf("expected dir %q, got %q", tt.expectedDir, dirTemplate)
			}
			if fileTemplate != tt.expectedFile {
				t.Errorf("expected file %q, got %q", tt.expectedFile, fileTemplate)
			}
		})
	}
}

func TestRename_CaseInsensitiveCollisionDetection(t *testing.T) {
	plannedPaths := make(map[string]bool)
	
	path1 := "Movies/The Matrix (1999)/The Matrix (1999).mkv"
	path2 := "movies/the matrix (1999)/the matrix (1999).mkv"

	lower1 := strings.ToLower(path1)
	plannedPaths[lower1] = true

	lower2 := strings.ToLower(path2)
	isConflict := plannedPaths[lower2]

	if !isConflict {
		t.Errorf("expected conflict to be detected for case-insensitive duplicate")
	}
}

func TestRename_TVTokensAndPresets(t *testing.T) {
	tokens := map[string]string{
		"showTitle":        "Breaking Bad",
		"seasonNumber":     "1",
		"seasonNumberPad":  "01",
		"episodeNumber":    "1",
		"episodeNumberPad": "01",
		"episodeTitle":     "Pilot",
		"year":             "2008",
		"resolution":       "1080p",
		"videoCodec":       "HEVC",
		"audioCodec":       "EAC3",
	}

	tests := []struct {
		name         string
		pattern      string
		expectedStem string
	}{
		{
			name:         "Kodi Standard",
			pattern:      "${showTitle}/Season ${seasonNumber}/${showTitle} - S${seasonNumberPad}E${episodeNumberPad}",
			expectedStem: "Breaking Bad/Season 1/Breaking Bad - S01E01",
		},
		{
			name:         "Kodi with Title",
			pattern:      "${showTitle}/Season ${seasonNumber}/${showTitle} - S${seasonNumberPad}E${episodeNumberPad} - ${episodeTitle}",
			expectedStem: "Breaking Bad/Season 1/Breaking Bad - S01E01 - Pilot",
		},
		{
			name:         "Plex Standard",
			pattern:      "${showTitle} (Season ${seasonNumberPad})/${showTitle} - s${seasonNumberPad}e${episodeNumberPad} - ${episodeTitle}",
			expectedStem: "Breaking Bad (Season 01)/Breaking Bad - s01e01 - Pilot",
		},
		{
			name:         "Jellyfin Standard",
			pattern:      "${showTitle}/Season ${seasonNumberPad}/${showTitle} S${seasonNumberPad}E${episodeNumberPad} ${episodeTitle}",
			expectedStem: "Breaking Bad/Season 01/Breaking Bad S01E01 Pilot",
		},
		{
			name:         "Flat Pattern",
			pattern:      "${showTitle} - S${seasonNumberPad}E${episodeNumberPad} - ${episodeTitle}",
			expectedStem: "Breaking Bad - S01E01 - Pilot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stem := namingTokenPattern.ReplaceAllStringFunc(tt.pattern, func(token string) string {
				name := namingTokenPattern.FindStringSubmatch(token)[1]
				return tokens[name]
			})

			if stem != tt.expectedStem {
				t.Errorf("got %q, want %q", stem, tt.expectedStem)
			}
		})
	}
}

