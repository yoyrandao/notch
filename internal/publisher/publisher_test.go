package publisher_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/yoyrandao/notch/internal/publisher"
)

func TestRun_EmptySteps(t *testing.T) {
	err := publisher.Run(nil, publisher.Env{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_ScriptSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash script tests not supported on Windows")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "ok.sh")
	os.WriteFile(script, []byte("#!/bin/sh\nexit 0\n"), 0o755)

	err := publisher.Run([]publisher.Step{{Name: "ok", Script: script}}, publisher.Env{Tag: "v1.0.0"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_ScriptFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash script tests not supported on Windows")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "fail.sh")
	os.WriteFile(script, []byte("#!/bin/sh\nexit 1\n"), 0o755)

	err := publisher.Run([]publisher.Step{{Name: "failing-step", Script: script}}, publisher.Env{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failing-step") {
		t.Errorf("error should contain step name, got: %v", err)
	}
}

func TestRun_EnvVarsInjected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash script tests not supported on Windows")
	}
	dir := t.TempDir()
	outFile := filepath.Join(dir, "out.txt")
	script := filepath.Join(dir, "check.sh")
	os.WriteFile(script, []byte(fmt.Sprintf(
		"#!/bin/sh\nprintf '%%s\\n%%s\\n%%s\\n%%s\\n%%s' \"$NOTCH_TAG\" \"$NOTCH_VERSION\" \"$NOTCH_COMMIT\" \"$NOTCH_CHANGELOG_PATH\" \"$NOTCH_REPOSITORY\" > %s\n",
		outFile,
	)), 0o755)

	env := publisher.Env{
		Tag:           "v2.3.4",
		Version:       "2.3.4",
		Commit:        "abc123def456abc123def456abc123def456abc1",
		ChangelogPath: "/repo/CHANGELOG.md",
		Repository:    "/repo",
	}
	if err := publisher.Run([]publisher.Step{{Name: "check", Script: script}}, env); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(outFile)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 5 {
		t.Fatalf("expected 5 lines, got %d: %q", len(lines), string(data))
	}
	wants := []string{"v2.3.4", "2.3.4", "abc123def456abc123def456abc123def456abc1", "/repo/CHANGELOG.md", "/repo"}
	for i, want := range wants {
		if lines[i] != want {
			t.Errorf("line[%d]: got %q, want %q", i, lines[i], want)
		}
	}
}

func TestRun_StopsOnFirstFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash script tests not supported on Windows")
	}
	dir := t.TempDir()

	failScript := filepath.Join(dir, "fail.sh")
	os.WriteFile(failScript, []byte("#!/bin/sh\nexit 1\n"), 0o755)

	sentinel := filepath.Join(dir, "ran")
	touchScript := filepath.Join(dir, "touch.sh")
	os.WriteFile(touchScript, []byte(fmt.Sprintf("#!/bin/sh\ntouch %s\n", sentinel)), 0o755)

	steps := []publisher.Step{
		{Name: "fail", Script: failScript},
		{Name: "should-not-run", Script: touchScript},
	}
	if err := publisher.Run(steps, publisher.Env{}); err == nil {
		t.Fatal("expected error, got nil")
	}
	if _, err := os.Stat(sentinel); err == nil {
		t.Error("second step should not have run after first failure")
	}
}
