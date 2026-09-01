package library

import (
	"path/filepath"
	"strings"
)

var (
	// PosterFilenames are standard filenames representing a poster or folder image.
	PosterFilenames = []string{"poster.jpg", "poster.png", "poster.jpeg", "folder.jpg", "cover.jpg"}
	// FanartFilenames are standard filenames representing background or backdrop artwork.
	FanartFilenames = []string{"fanart.jpg", "fanart.png", "landscape.jpg", "backdrop.jpg"}
)

// HasLocalArtwork checks whether the item sidecars contain a local image for the given artwork kind.
func HasLocalArtwork(sidecars []Sidecar, kind string) bool {
	return FindLocalArtwork(sidecars, kind) != nil
}

// FindLocalArtwork returns the first matching sidecar for the given artwork kind, or nil if none found.
func FindLocalArtwork(sidecars []Sidecar, kind string) *Sidecar {
	for i := range sidecars {
		asset := &sidecars[i]
		if asset.Kind != "image" {
			continue
		}
		name := strings.ToLower(strings.TrimSuffix(filepath.Base(asset.RelativePath), filepath.Ext(asset.RelativePath)))
		if kind == "poster" && (name == "poster" || name == "folder" || name == "cover") {
			return asset
		}
		if kind == "fanart" && (name == "fanart" || name == "backdrop" || name == "landscape") {
			return asset
		}
	}
	return nil
}
