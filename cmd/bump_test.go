package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
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
	gitRun(t, dir, "config", "user.name", "test")
	gitRun(t, dir, "config", "user.email", "test@example.com")
	return dir
}

func setGitEnv(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_AUTHOR_NAME", "test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
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

// --- compute-only tests via --dry-run (no side effects) ---

func TestBump_NoTag_Feat(t *testing.T) {
	gitAvailable(t)
	dir := initRepo(t)
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: init")

	stdout, _, err := runBump(t, dir, "--dry-run")
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

	stdout, _, err := runBump(t, dir, "--dry-run")
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

	stdout, _, err := runBump(t, dir, "--dry-run")
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

	stdout, _, err := runBump(t, dir, "--dry-run")
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

	stdout, _, err := runBump(t, dir, "--dry-run", "--pre", "rc")
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

	stdout, _, err := runBump(t, dir, "--dry-run", "--release-as", "1.0.0")
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

// --- full-release tests (side effects, no remote) ---

func TestBump_FullRelease_NoPush(t *testing.T) {
	gitAvailable(t)
	setGitEnv(t)
	dir := initRepo(t)
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: login")
	gitRun(t, dir, "commit", "--allow-empty", "-m", "fix(api): nil ptr")
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat!: drop legacy")

	stdout, _, err := runBump(t, dir, "--no-push")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(stdout) != "v0.1.0" {
		t.Fatalf("stdout = %q", stdout)
	}

	b, err := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	cl := string(b)
	for _, want := range []string{
		"# Changelog",
		"## [0.1.0]",
		"### BREAKING CHANGES",
		"drop legacy",
		"### Added",
		"login",
		"### Fixed",
		"api: nil ptr",
	} {
		if !strings.Contains(cl, want) {
			t.Errorf("CHANGELOG missing %q:\n%s", want, cl)
		}
	}

	tag, found, err := lastTag(dir)
	if err != nil || !found || tag != "v0.1.0" {
		t.Fatalf("expected tag v0.1.0, got %q found=%v err=%v", tag, found, err)
	}

	headMsg := headCommitSubject(t, dir)
	if headMsg != "chore(release): v0.1.0" {
		t.Fatalf("head subject = %q", headMsg)
	}
}

func TestBump_DryRun_NoSideEffects(t *testing.T) {
	gitAvailable(t)
	setGitEnv(t)
	dir := initRepo(t)
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: init")

	headBefore := headSHA(t, dir)

	_, stderr, err := runBump(t, dir, "--dry-run", "--no-push")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr, "dry-run") || !strings.Contains(stderr, "CHANGELOG section") {
		t.Fatalf("stderr missing dry-run preview:\n%s", stderr)
	}

	if _, err := os.Stat(filepath.Join(dir, "CHANGELOG.md")); !os.IsNotExist(err) {
		t.Fatalf("CHANGELOG.md should not exist, err = %v", err)
	}
	if got := headSHA(t, dir); got != headBefore {
		t.Fatalf("HEAD moved: before=%s after=%s", headBefore, got)
	}
	_, found, _ := lastTag(dir)
	if found {
		t.Fatal("tag was created during dry-run")
	}
}

func TestBump_FullRelease_WithPush(t *testing.T) {
	gitAvailable(t)
	setGitEnv(t)
	dir := initRepo(t)
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: init")

	bare := t.TempDir()
	gitRun(t, bare, "init", "--bare", "-q", "-b", "main")
	gitRun(t, dir, "remote", "add", "origin", bare)

	stdout, _, err := runBump(t, dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(stdout) != "v0.1.0" {
		t.Fatalf("stdout = %q", stdout)
	}

	tag, found, err := lastTag(bare)
	if err != nil || !found || tag != "v0.1.0" {
		t.Fatalf("remote tag = %q found=%v err=%v", tag, found, err)
	}
}

// --- small helpers using git CLI directly to avoid importing the gitx package ---

func lastTag(dir string) (string, bool, error) {
	out, err := exec.Command("git", "-C", dir, "describe", "--tags", "--abbrev=0").CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "No names found") {
			return "", false, nil
		}
		return "", false, err
	}
	return strings.TrimSpace(string(out)), true, nil
}

func headSHA(t *testing.T, dir string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").CombinedOutput()
	if err != nil {
		t.Fatalf("rev-parse: %v: %s", err, out)
	}
	return strings.TrimSpace(string(out))
}

func headCommitSubject(t *testing.T, dir string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "log", "-1", "--format=%s").CombinedOutput()
	if err != nil {
		t.Fatalf("log: %v: %s", err, out)
	}
	return strings.TrimSpace(string(out))
}
