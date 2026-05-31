package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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

	stdout, _, err := runBump(t, dir, "--dry-run", "--as", "1.0.0")
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

	_, _, err := runBump(t, dir, "--as", "1.0.0")
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
		"### Features and enhancements",
		"login",
		"### Bug fixes",
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

// --- non-conventional commit history ---

// Non-conventional commits in history are skipped: they neither contribute to
// the bump nor break the run. Only conventional commits drive the version.
func TestBump_NonConventionalCommitsSkipped(t *testing.T) {
	gitAvailable(t)
	dir := initRepo(t)
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: init")
	gitRun(t, dir, "tag", "v1.0.0")
	gitRun(t, dir, "commit", "--allow-empty", "-m", "WIP")
	gitRun(t, dir, "commit", "--allow-empty", "-m", "fix: a bug")
	gitRun(t, dir, "commit", "--allow-empty", "-m", "updated readme")
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: a feature")
	gitRun(t, dir, "commit", "--allow-empty", "-m", "Merge pull request #1")

	stdout, stderr, err := runBump(t, dir, "--dry-run", "--verbose")
	if err != nil {
		t.Fatal(err)
	}
	// 1 feat + 1 fix => minor over v1.0.0.
	if strings.TrimSpace(stdout) != "v1.1.0" {
		t.Fatalf("got %q, want v1.1.0", strings.TrimSpace(stdout))
	}
	// 3 non-conventional messages skipped (WIP, updated readme, Merge ...).
	if !strings.Contains(stderr, "commits=2") || !strings.Contains(stderr, "skipped=3") {
		t.Fatalf("verbose plan missing commits=2 skipped=3:\n%s", stderr)
	}
}

// A tagged repo whose only new commits are non-conventional contributes no
// bump => nothing to release.
func TestBump_AllNonConventional_NothingToRelease(t *testing.T) {
	gitAvailable(t)
	dir := initRepo(t)
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: init")
	gitRun(t, dir, "tag", "v1.0.0")
	gitRun(t, dir, "commit", "--allow-empty", "-m", "WIP")
	gitRun(t, dir, "commit", "--allow-empty", "-m", "fixed stuff")

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

// With no prior tag, the first release is 0.1.0 regardless of commit content,
// so non-conventional commits still yield the initial version.
func TestBump_AllNonConventional_NoTag_InitialRelease(t *testing.T) {
	gitAvailable(t)
	dir := initRepo(t)
	gitRun(t, dir, "commit", "--allow-empty", "-m", "initial import")
	gitRun(t, dir, "commit", "--allow-empty", "-m", "WIP")

	stdout, stderr, err := runBump(t, dir, "--dry-run", "--verbose")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(stdout) != "v0.1.0" {
		t.Fatalf("got %q, want v0.1.0", strings.TrimSpace(stdout))
	}
	if !strings.Contains(stderr, "commits=0") || !strings.Contains(stderr, "skipped=2") {
		t.Fatalf("verbose plan missing commits=0 skipped=2:\n%s", stderr)
	}
}

// --- merge-commit subject unwrapping (Azure DevOps etc.) ---

func TestBump_UnwrapAzurePrefix(t *testing.T) {
	gitAvailable(t)
	dir := initRepo(t)
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: init")
	gitRun(t, dir, "tag", "v1.0.0")
	gitRun(t, dir, "commit", "--allow-empty", "-m", "Merged PR 5: feat: a feature")
	gitRun(t, dir, "commit", "--allow-empty", "-m", "Merged PR 6: fix: a bug")
	cfg := writeConfig(t, dir, "commit:\n  subject_pattern: '^Merged PR \\d+: (.+)$'\n")

	stdout, stderr, err := runBump(t, dir, "--dry-run", "--verbose", "--config", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(stdout) != "v1.1.0" {
		t.Fatalf("got %q, want v1.1.0", strings.TrimSpace(stdout))
	}
	if !strings.Contains(stderr, "commits=2") || !strings.Contains(stderr, "skipped=0") {
		t.Fatalf("verbose plan missing commits=2 skipped=0:\n%s", stderr)
	}
}

// Without the pattern configured, Azure-wrapped commits stay non-conventional.
func TestBump_NoUnwrapConfig_Skipped(t *testing.T) {
	gitAvailable(t)
	dir := initRepo(t)
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: init")
	gitRun(t, dir, "tag", "v1.0.0")
	gitRun(t, dir, "commit", "--allow-empty", "-m", "Merged PR 5: feat: a feature")
	gitRun(t, dir, "commit", "--allow-empty", "-m", "Merged PR 6: fix: a bug")

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

// Pattern matches but the extracted payload is not conventional -> skipped.
func TestBump_UnwrapInvalidPayload_Skipped(t *testing.T) {
	gitAvailable(t)
	dir := initRepo(t)
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: init")
	gitRun(t, dir, "tag", "v1.0.0")
	gitRun(t, dir, "commit", "--allow-empty", "-m", "Merged PR 7: random text")
	cfg := writeConfig(t, dir, "commit:\n  subject_pattern: '^Merged PR \\d+: (.+)$'\n")

	_, stderr, err := runBump(t, dir, "--dry-run", "--verbose", "--config", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr, "commits=0") || !strings.Contains(stderr, "skipped=1") {
		t.Fatalf("verbose plan missing commits=0 skipped=1:\n%s", stderr)
	}
}

func TestBump_InvalidCommitPattern(t *testing.T) {
	gitAvailable(t)
	dir := initRepo(t)
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: init")

	t.Run("bad regex", func(t *testing.T) {
		cfg := writeConfig(t, dir, "commit:\n  subject_pattern: '^Merged PR ([0-9'\n")
		_, _, err := runBump(t, dir, "--dry-run", "--config", cfg)
		if err == nil {
			t.Fatal("expected error for invalid regex")
		}
	})
	t.Run("no capture group", func(t *testing.T) {
		cfg := writeConfig(t, dir, "commit:\n  subject_pattern: '^Merged PR \\d+: .+$'\n")
		_, _, err := runBump(t, dir, "--dry-run", "--config", cfg)
		if err == nil || !strings.Contains(err.Error(), "capture group") {
			t.Fatalf("expected capture-group error, got %v", err)
		}
	})
}

// --- config-driven bump tests ---

func writeConfig(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "test-notch.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestBump_ConfigPrefix(t *testing.T) {
	gitAvailable(t)
	dir := initRepo(t)
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: init")
	cfg := writeConfig(t, dir, "tag:\n  prefix: \"rel-\"\n")

	stdout, _, err := runBump(t, dir, "--dry-run", "--config", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(stdout) != "rel-0.1.0" {
		t.Fatalf("got %q, want %q", strings.TrimSpace(stdout), "rel-0.1.0")
	}
}

func TestBump_ConfigChangelogPath(t *testing.T) {
	gitAvailable(t)
	setGitEnv(t)
	dir := initRepo(t)
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: init")
	cfg := writeConfig(t, dir, "changelog:\n  path: \"CHANGES.md\"\n")

	_, _, err := runBump(t, dir, "--no-push", "--config", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "CHANGES.md")); statErr != nil {
		t.Fatal("CHANGES.md not created")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "CHANGELOG.md")); !os.IsNotExist(statErr) {
		t.Fatal("CHANGELOG.md should not exist")
	}
}

func TestBump_ConfigNoPush(t *testing.T) {
	gitAvailable(t)
	setGitEnv(t)
	dir := initRepo(t)
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: init")

	bare := t.TempDir()
	gitRun(t, bare, "init", "--bare", "-q", "-b", "main")
	gitRun(t, dir, "remote", "add", "origin", bare)

	cfg := writeConfig(t, dir, "tag:\n  push: false\n")

	_, _, err := runBump(t, dir, "--config", cfg)
	if err != nil {
		t.Fatal(err)
	}
	_, found, _ := lastTag(bare)
	if found {
		t.Fatal("tag was pushed despite config push: false")
	}
}

func TestBump_FlagOverridesConfig(t *testing.T) {
	gitAvailable(t)
	setGitEnv(t)
	dir := initRepo(t)
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: init")
	cfg := writeConfig(t, dir, "changelog:\n  path: \"CHANGES.md\"\n")

	_, _, err := runBump(t, dir, "--no-push", "--config", cfg, "--changelog", "CHANGELOG.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "CHANGELOG.md")); statErr != nil {
		t.Fatal("CHANGELOG.md not created (flag should override config)")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "CHANGES.md")); !os.IsNotExist(statErr) {
		t.Fatal("CHANGES.md should not exist")
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

func headFiles(t *testing.T, dir string) []string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "show", "HEAD", "--name-only", "--format=").CombinedOutput()
	if err != nil {
		t.Fatalf("show: %v: %s", err, out)
	}
	var files []string
	for l := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			files = append(files, l)
		}
	}
	return files
}

// --- tool-specific project patching ---

func TestBump_PatchHelm(t *testing.T) {
	gitAvailable(t)
	setGitEnv(t)
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "Chart.yaml"),
		[]byte("apiVersion: v2\nname: app\nversion: 0.0.0\nappVersion: \"1.0\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, dir, "add", "Chart.yaml")
	gitRun(t, dir, "commit", "-m", "feat: init")

	if _, _, err := runBump(t, dir, "--no-push"); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "Chart.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	want := "apiVersion: v2\nname: app\nversion: 0.1.0\nappVersion: \"1.0\"\n"
	if string(got) != want {
		t.Fatalf("Chart.yaml:\n%q\nwant:\n%q", got, want)
	}
	if files := headFiles(t, dir); !slices.Contains(files, "Chart.yaml") || !slices.Contains(files, "CHANGELOG.md") {
		t.Fatalf("release commit files = %v", files)
	}
}

func TestBump_PatchNpm(t *testing.T) {
	gitAvailable(t)
	setGitEnv(t)
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "package.json"),
		[]byte("{\n  \"name\": \"app\",\n  \"version\": \"0.0.0\"\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, dir, "add", "package.json")
	gitRun(t, dir, "commit", "-m", "feat: init")

	if _, _, err := runBump(t, dir, "--no-push"); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"name\": \"app\",\n  \"version\": \"0.1.0\"\n}\n"
	if string(got) != want {
		t.Fatalf("package.json:\n%q\nwant:\n%q", got, want)
	}
	if files := headFiles(t, dir); !slices.Contains(files, "package.json") {
		t.Fatalf("release commit files = %v", files)
	}
}

func TestBump_NoToolProject_OnlyChangelog(t *testing.T) {
	gitAvailable(t)
	setGitEnv(t)
	dir := initRepo(t)
	gitRun(t, dir, "commit", "--allow-empty", "-m", "feat: init")

	if _, _, err := runBump(t, dir, "--no-push"); err != nil {
		t.Fatal(err)
	}
	if files := headFiles(t, dir); len(files) != 1 || files[0] != "CHANGELOG.md" {
		t.Fatalf("expected only CHANGELOG.md, got %v", files)
	}
}

func TestBump_DryRun_PatchPreview(t *testing.T) {
	gitAvailable(t)
	setGitEnv(t)
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "Chart.yaml"),
		[]byte("version: 0.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, dir, "add", "Chart.yaml")
	gitRun(t, dir, "commit", "-m", "feat: init")

	_, stderr, err := runBump(t, dir, "--dry-run", "--no-push")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr, "patch Chart.yaml") {
		t.Fatalf("stderr missing patch step:\n%s", stderr)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "Chart.yaml")); string(got) != "version: 0.0.0\n" {
		t.Fatalf("dry-run mutated file: %q", got)
	}
}
