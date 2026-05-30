package changelog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRender_AllKinds(t *testing.T) {
	entries := []Entry{
		{Kind: KindFeature, Description: "add login", Hash: "abcdef1234"},
		{Kind: KindFix, Scope: "api", Description: "nil ptr", Hash: "1234567abc"},
		{Kind: KindOther, Description: "bump deps", Hash: "9999999"},
		{Kind: KindBreakingChange, Scope: "core", Description: "drop v1", Hash: "ffffffff"},
	}
	got := Render("0.2.0", "2026-05-30", entries)
	want := `## [0.2.0] - 2026-05-30

### BREAKING CHANGES

- core: drop v1 (fffffff)

### Added

- add login (abcdef1)

### Fixed

- api: nil ptr (1234567)

### Changed

- bump deps (9999999)

`
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRender_OmitsEmpty(t *testing.T) {
	entries := []Entry{{Kind: KindFix, Description: "x", Hash: "abcdefg"}}
	got := Render("0.0.1", "2026-01-01", entries)
	if strings.Contains(got, "BREAKING") || strings.Contains(got, "Added") || strings.Contains(got, "Changed") {
		t.Fatalf("empty sections leaked:\n%s", got)
	}
	if !strings.Contains(got, "### Fixed") {
		t.Fatalf("missing Fixed:\n%s", got)
	}
}

func TestRender_BreakingOnly(t *testing.T) {
	entries := []Entry{
		{Kind: KindBreakingChange, Description: "remove v1 api", Hash: "abcdef1"},
	}
	got := Render("1.0.0", "2026-05-30", entries)
	if !strings.Contains(got, "### BREAKING CHANGES") {
		t.Fatalf("missing BREAKING CHANGES:\n%s", got)
	}
	if strings.Contains(got, "### Added") {
		t.Fatalf("should not contain Added")
	}
}

func TestUpdate_CreatesNewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CHANGELOG.md")
	section := "## [0.1.0] - 2026-05-30\n\n### Added\n\n- x (abcdef1)\n\n"
	if err := Update(path, section); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.HasPrefix(s, "# Changelog\n") {
		t.Fatalf("missing top header:\n%s", s)
	}
	if !strings.Contains(s, section) {
		t.Fatalf("missing section:\n%s", s)
	}
}

func TestUpdate_PrependsToExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CHANGELOG.md")
	initial := "# Changelog\n\nintro\n\n## [0.1.0] - 2026-01-01\n\n- old\n"
	if err := os.WriteFile(path, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}
	section := "## [0.2.0] - 2026-05-30\n\n### Added\n\n- new\n\n"
	if err := Update(path, section); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	s := string(b)
	pos02 := strings.Index(s, "## [0.2.0]")
	pos01 := strings.Index(s, "## [0.1.0]")
	if pos02 < 0 || pos01 < 0 || pos02 >= pos01 {
		t.Fatalf("ordering wrong:\n%s", s)
	}
	if !strings.HasPrefix(s, "# Changelog\n") {
		t.Fatal("top header lost")
	}
}

func TestUpdate_AppendsWhenNoExistingSections(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CHANGELOG.md")
	initial := "# Changelog\n\nintro only, no releases yet\n"
	if err := os.WriteFile(path, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}
	section := "## [0.1.0] - 2026-05-30\n\n- thing\n\n"
	if err := Update(path, section); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	s := string(b)
	if !strings.Contains(s, "intro only") {
		t.Fatal("lost intro")
	}
	if !strings.Contains(s, "## [0.1.0]") {
		t.Fatal("missing new section")
	}
	if strings.Index(s, "intro only") > strings.Index(s, "## [0.1.0]") {
		t.Fatal("section should be after intro")
	}
}
