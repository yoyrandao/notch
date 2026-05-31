package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yoyrandao/notch/internal/config"
)

func runInit(t *testing.T, dir string, extra ...string) (string, error) {
	t.Helper()
	root := NewRootCmd()
	var errBuf bytes.Buffer
	root.SetErr(&errBuf)
	args := append([]string{"init"}, extra...)
	root.SetArgs(args)
	// Change to dir so init writes there
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
	err = root.Execute()
	return errBuf.String(), err
}

func TestInit_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	_, err := runInit(t, dir)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, config.DefaultPath())
	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("file not created: %v", statErr)
	}
	cfg, err := config.LoadFile(path)
	if err != nil {
		t.Fatalf("created file not parseable: %v", err)
	}
	if cfg.Tag.Prefix != "v" {
		t.Errorf("Tag.Prefix = %q", cfg.Tag.Prefix)
	}
}

func TestInit_ErrorIfExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, config.DefaultPath())
	if err := os.WriteFile(path, []byte("repo: ."), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := runInit(t, dir)
	if err == nil {
		t.Fatal("expected error when file already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error: %v", err)
	}
}
