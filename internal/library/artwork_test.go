package library

import "testing"

func TestFindLocalArtwork(t *testing.T) {
	sidecars := []Sidecar{
		{RelativePath: "folder.jpg", Kind: "image"},
		{RelativePath: "fanart.png", Kind: "image"},
		{RelativePath: "movie.nfo", Kind: "nfo"},
	}

	if !HasLocalArtwork(sidecars, "poster") {
		t.Fatal("expected poster to be found")
	}
	if !HasLocalArtwork(sidecars, "fanart") {
		t.Fatal("expected fanart to be found")
	}
	if HasLocalArtwork(sidecars, "banner") {
		t.Fatal("expected banner to not be found")
	}

	poster := FindLocalArtwork(sidecars, "poster")
	if poster == nil || poster.RelativePath != "folder.jpg" {
		t.Fatalf("unexpected poster: %+v", poster)
	}

	fanart := FindLocalArtwork(sidecars, "fanart")
	if fanart == nil || fanart.RelativePath != "fanart.png" {
		t.Fatalf("unexpected fanart: %+v", fanart)
	}
}
