package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := LoadOptional(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Repository != "." {
		t.Errorf("Repository = %q, want %q", cfg.Repository, ".")
	}
	if cfg.Tag.Prefix != "v" {
		t.Errorf("Tag.Prefix = %q, want %q", cfg.Tag.Prefix, "v")
	}
	if !cfg.Tag.Push {
		t.Error("Tag.Push = false, want true")
	}
	if cfg.Changelog.Path != "CHANGELOG.md" {
		t.Errorf("Changelog.Path = %q, want %q", cfg.Changelog.Path, "CHANGELOG.md")
	}
	if cfg.Commit.ReleaseMessage != "chore(release): {tag}" {
		t.Errorf("Commit.ReleaseMessage = %q, want %q", cfg.Commit.ReleaseMessage, "chore(release): {tag}")
	}
}

func TestLoad_PartialFile(t *testing.T) {
	f := filepath.Join(t.TempDir(), "cfg.yaml")
	if err := os.WriteFile(f, []byte(`tag:
  prefix: "rel-"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Tag.Prefix != "rel-" {
		t.Errorf("Tag.Prefix = %q, want %q", cfg.Tag.Prefix, "rel-")
	}
	// unset fields keep defaults
	if cfg.Repository != "." {
		t.Errorf("Repository = %q, want %q", cfg.Repository, ".")
	}
	if !cfg.Tag.Push {
		t.Error("Tag.Push = false, want true")
	}
	if cfg.Changelog.Path != "CHANGELOG.md" {
		t.Errorf("Changelog.Path = %q, want %q", cfg.Changelog.Path, "CHANGELOG.md")
	}
}

func TestLoad_CommitSubjectPattern(t *testing.T) {
	f := filepath.Join(t.TempDir(), "cfg.yaml")
	if err := os.WriteFile(f, []byte("commit:\n  subject_pattern: '^Merged PR \\d+: (.+)$'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Commit.SubjectPattern != `^Merged PR \d+: (.+)$` {
		t.Errorf("Commit.SubjectPattern = %q", cfg.Commit.SubjectPattern)
	}
	// default is empty (disabled)
	cfg2, _ := LoadOptional(filepath.Join(t.TempDir(), "none.yaml"))
	if cfg2.Commit.SubjectPattern != "" {
		t.Errorf("default SubjectPattern = %q, want empty", cfg2.Commit.SubjectPattern)
	}
}

func TestLoad_ExplicitMissing(t *testing.T) {
	_, err := LoadFile(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatal("expected error for missing file with LoadFile")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	f := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(f, []byte(`: : invalid`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadFile(f)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestDefaultYAML_RoundTrip(t *testing.T) {
	f := filepath.Join(t.TempDir(), "defaults.yaml")
	if err := os.WriteFile(f, []byte(DefaultYAML()), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Repository != "." {
		t.Errorf("Repository = %q", cfg.Repository)
	}
	if cfg.Tag.Prefix != "v" {
		t.Errorf("Tag.Prefix = %q", cfg.Tag.Prefix)
	}
	if !cfg.Tag.Push {
		t.Error("Tag.Push = false")
	}
	if cfg.Changelog.Path != "CHANGELOG.md" {
		t.Errorf("Changelog.Path = %q", cfg.Changelog.Path)
	}
}

func TestLoad_PublishSteps(t *testing.T) {
	f := filepath.Join(t.TempDir(), "cfg.yaml")
	if err := os.WriteFile(f, []byte(`
publish:
  steps:
    - name: upload
      script: ./scripts/upload.sh
    - name: notify
      script: ./scripts/notify.sh
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFile(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Publish.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(cfg.Publish.Steps))
	}
	if cfg.Publish.Steps[0].Name != "upload" {
		t.Errorf("step[0].Name = %q, want %q", cfg.Publish.Steps[0].Name, "upload")
	}
	if cfg.Publish.Steps[0].Script != "./scripts/upload.sh" {
		t.Errorf("step[0].Script = %q, want %q", cfg.Publish.Steps[0].Script, "./scripts/upload.sh")
	}
	if cfg.Publish.Steps[1].Name != "notify" {
		t.Errorf("step[1].Name = %q, want %q", cfg.Publish.Steps[1].Name, "notify")
	}
}
