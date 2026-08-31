package metadata

import "testing"

func TestReleaseYear(t *testing.T) {
	if year := releaseYear("1999-03-30"); year == nil || *year != 1999 {
		t.Fatalf("unexpected year: %v", year)
	}
	if releaseYear("bad") != nil {
		t.Fatal("invalid date returned a year")
	}
}
