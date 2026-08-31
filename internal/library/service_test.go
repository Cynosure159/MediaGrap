package library

import "testing"

func TestMetadataSearchHintPrefersMovieDirectory(t *testing.T) {
	service := NewService(nil, nil)
	title, year, origin := service.MetadataSearchHint(MediaItem{
		RelativePath: "奥本海默 (2023)/Oppenheimer.2023.1080p.MA.WEB-DL.DUAL.DD+5.1.H.265-TheBiscuitMan.mkv",
		TitleHint:    "Oppenheimer 1080p MA WEB DL DUAL DD+5 1 H 265 TheBiscuitMan",
	})
	if title != "奥本海默" || year == nil || *year != 2023 || origin != "parent_directory" {
		t.Fatalf("unexpected directory search hint: title=%q year=%v origin=%q", title, year, origin)
	}
}

func TestMetadataSearchHintFallsBackToFilenameAtSourceRoot(t *testing.T) {
	service := NewService(nil, nil)
	year := 2010
	title, actualYear, origin := service.MetadataSearchHint(MediaItem{RelativePath: "Inception.2010.mkv", TitleHint: "Inception", YearHint: &year})
	if title != "Inception" || actualYear != &year || origin != "filename" {
		t.Fatalf("unexpected filename search hint: title=%q year=%v origin=%q", title, actualYear, origin)
	}
}
