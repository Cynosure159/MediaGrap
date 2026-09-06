package metadata

import (
	"path/filepath"
	"testing"
)

func TestTVArtworkTargetUsesKodiSeasonFilename(t *testing.T) {
	season := 1
	target, err := tvArtworkTarget("/media/Example", TVArtworkCandidate{Scope: "season", SeasonNumber: &season, Kind: "season_poster", MimeType: "image/jpeg"})
	if err != nil {
		t.Fatal(err)
	}
	if target != filepath.Join("/media/Example", "season01-poster.jpg") {
		t.Fatalf("unexpected target: %s", target)
	}
}

func TestTVArtworkScopeRejectsMissingSeason(t *testing.T) {
	if err := validateTVArtworkScope("season", nil); err == nil {
		t.Fatal("expected missing season to be rejected")
	}
}
