package renamepattern

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRenderOptionalExpressions(t *testing.T) {
	allowed := Allowed("title", "edition", "year", "resolution")
	got, err := Render("${title}${ - ,edition,}${ (,year,)}", map[string]string{"title": "Film", "edition": "", "year": "0"}, allowed)
	if err != nil || got != "Film (0)" {
		t.Fatalf("render=%q err=%v", got, err)
	}
	got, err = Render("${title}${ - ,edition,}${ [,resolution,]}", map[string]string{"title": "Film", "edition": "Cut", "resolution": "1080p"}, allowed)
	if err != nil || got != "Film - Cut [1080p]" {
		t.Fatalf("render=%q err=%v", got, err)
	}
}

func TestStrictMissingAndMalformedOptionalExpressions(t *testing.T) {
	allowed := Allowed("title", "edition")
	for _, pattern := range []string{"${edition}", "${unknown}", "${a,b}", "${x,edition,/}", "${, edition,${title}}"} {
		if _, err := Render(pattern, map[string]string{}, allowed); err == nil {
			t.Fatalf("accepted invalid pattern %q", pattern)
		}
	}
	if got, err := Render("${title}${,edition,}", map[string]string{"title": "Film", "edition": "0"}, allowed); err != nil || got != "Film0" {
		t.Fatalf("zero value render=%q err=%v", got, err)
	}
	if _, err := Render("${title}", map[string]string{"title": " \t\n"}, allowed); err == nil {
		t.Fatal("accepted whitespace-only required value")
	}
	if got, err := Render("${,edition,}", map[string]string{"edition": " \t\n"}, allowed); err != nil || got != "" {
		t.Fatalf("whitespace-only optional render=%q err=%v", got, err)
	}
}

func TestSharedFixtureMatrix(t *testing.T) {
	_, sourceFile, _, _ := runtime.Caller(0)
	fixturePath := filepath.Join(filepath.Dir(sourceFile), "../../web/src/composables/renamePatternFixtures.json")
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct {
		Name     string            `json:"name"`
		Pattern  string            `json:"pattern"`
		Allowed  []string          `json:"allowed"`
		Values   map[string]string `json:"values"`
		Rendered string            `json:"rendered"`
		Error    bool              `json:"error"`
	}
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			got, err := Render(fixture.Pattern, fixture.Values, Allowed(fixture.Allowed...))
			if fixture.Error {
				if err == nil {
					t.Fatalf("accepted invalid fixture, rendered %q", got)
				}
				return
			}
			if err != nil || got != fixture.Rendered {
				t.Fatalf("render=%q err=%v want=%q", got, err, fixture.Rendered)
			}
		})
	}
}

func TestValidatePathLimits(t *testing.T) {
	allowed := Allowed("title", "edition")
	for _, pattern := range []string{"", "a//b", "../movie", "${x,edition,/}", "${title}{", "${title}$"} {
		if err := Validate(pattern, allowed); err == nil {
			t.Fatalf("accepted unsafe pattern %q", pattern)
		}
	}
	if err := Validate("${title}/${,edition,}", allowed); err != nil {
		t.Fatalf("rejected optional path segment syntax: %v", err)
	}
	if err := Validate(strings.Repeat("影", 342), allowed); err == nil {
		t.Fatal("accepted a pattern over the UTF-8 byte limit")
	}
}
