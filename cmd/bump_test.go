package cmd

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func gitAvailable(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(cmd.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitRun(t, dir, "init", "-q", "-b", "main")
	gitRun(t, dir, "config", "commit.gpgsign", "false")
	gitRun(t, dir, "config", "tag.gpgsign", "false")
	return dir
}

func runBump(t *testing.T, repoDir string, extra ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := NewRootCmd()
	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	args := append([]string{"bump", "--repo", repoDir}, extra...)
	root.SetArgs(args)
	err = root.Execute()
	return outBuf.String(), errBuf.String(), err
}

func TestBump_NoTag_Feat(t *testing.T) {
	gitAvailable(t)
	dir := initRepo(t)
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: init")

	stdout, _, err := runBump(t, dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(stdout) != "v0.1.0" {
		t.Fatalf("got %q", stdout)
	}
}

func TestBump_Tagged_Feat(t *testing.T) {
	gitAvailable(t)
	dir := initRepo(t)
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: init")
	gitRun(t, dir, "tag", "v1.2.3")
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: more")

	stdout, _, err := runBump(t, dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(stdout) != "v1.3.0" {
		t.Fatalf("got %q", stdout)
	}
}

func TestBump_Tagged_BreakingPost1(t *testing.T) {
	gitAvailable(t)
	dir := initRepo(t)
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: init")
	gitRun(t, dir, "tag", "v1.2.3")
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat!: break")

	stdout, _, err := runBump(t, dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(stdout) != "v2.0.0" {
		t.Fatalf("got %q", stdout)
	}
}

func TestBump_Tagged_BreakingPre1Degrades(t *testing.T) {
	gitAvailable(t)
	dir := initRepo(t)
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: init")
	gitRun(t, dir, "tag", "v0.5.0")
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat!: break")

	stdout, _, err := runBump(t, dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(stdout) != "v0.6.0" {
		t.Fatalf("got %q", stdout)
	}
}

func TestBump_NothingToRelease(t *testing.T) {
	gitAvailable(t)
	dir := initRepo(t)
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: init")
	gitRun(t, dir, "tag", "v1.2.3")

	stdout, stderr, err := runBump(t, dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("stdout should be empty, got %q", stdout)
	}
	if !strings.Contains(stderr, "nothing to release") {
		t.Fatalf("stderr missing 'nothing to release': %q", stderr)
	}
}

func TestBump_PreFlag(t *testing.T) {
	gitAvailable(t)
	dir := initRepo(t)
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: init")
	gitRun(t, dir, "tag", "v1.2.3")
	gitRun(t, dir, "commit", "--allow-empty", "-m", "fix: a")

	stdout, _, err := runBump(t, dir, "--pre", "rc")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(stdout) != "v1.2.4-rc.1" {
		t.Fatalf("got %q", stdout)
	}
}

func TestBump_ReleaseAs(t *testing.T) {
	gitAvailable(t)
	dir := initRepo(t)
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: init")
	gitRun(t, dir, "tag", "v0.5.0")
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: more")

	stdout, _, err := runBump(t, dir, "--release-as", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(stdout) != "v1.0.0" {
		t.Fatalf("got %q", stdout)
	}
}

func TestBump_ReleaseAs_NotGreater(t *testing.T) {
	gitAvailable(t)
	dir := initRepo(t)
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: init")
	gitRun(t, dir, "tag", "v1.2.3")

	_, _, err := runBump(t, dir, "--release-as", "1.0.0")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "greater") {
		t.Fatalf("err missing 'greater': %v", err)
	}
}

func TestBump_InvalidLastTag(t *testing.T) {
	gitAvailable(t)
	dir := initRepo(t)
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: init")
	gitRun(t, dir, "tag", "not-a-tag")

	_, _, err := runBump(t, dir)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not semver") {
		t.Fatalf("err missing 'not semver': %v", err)
	}
}
