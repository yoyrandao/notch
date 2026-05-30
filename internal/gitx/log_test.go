package gitx

import (
	"os/exec"
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

func mustRun(t *testing.T, dir string, args ...string) {
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
	mustRun(t, dir, "init", "-q", "-b", "main")
	mustRun(t, dir, "config", "commit.gpgsign", "false")
	mustRun(t, dir, "config", "tag.gpgsign", "false")
	return dir
}

func TestLastTag_NoTags(t *testing.T) {
	gitAvailable(t)
	dir := initRepo(t)
	mustRun(t, dir, "commit", "--allow-empty", "-m", "feat: initial")

	tag, found, err := LastTag(dir)
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatalf("expected no tag, got %q", tag)
	}
}

func TestLastTag_Returns(t *testing.T) {
	gitAvailable(t)
	dir := initRepo(t)
	mustRun(t, dir, "commit", "--allow-empty", "-m", "feat: initial")
	mustRun(t, dir, "tag", "v0.1.0")
	mustRun(t, dir, "commit", "--allow-empty", "-m", "fix: a")

	tag, found, err := LastTag(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !found || tag != "v0.1.0" {
		t.Fatalf("got tag=%q found=%v, want v0.1.0", tag, found)
	}
}

func TestCommitsSince(t *testing.T) {
	gitAvailable(t)
	dir := initRepo(t)
	mustRun(t, dir, "commit", "--allow-empty", "-m", "feat: initial")
	mustRun(t, dir, "tag", "v0.1.0")
	mustRun(t, dir, "commit", "--allow-empty", "-m", "fix: a")
	mustRun(t, dir, "commit", "--allow-empty", "-m", "feat: b")

	msgs, err := CommitsSince(dir, "v0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("got %d messages: %v", len(msgs), msgs)
	}
	if !slices.Contains(msgs, "fix: a") || !slices.Contains(msgs, "feat: b") {
		t.Fatalf("missing expected messages: %v", msgs)
	}
}

func TestCommitsSince_NoRefReturnsAll(t *testing.T) {
	gitAvailable(t)
	dir := initRepo(t)
	mustRun(t, dir, "commit", "--allow-empty", "-m", "feat: initial")
	mustRun(t, dir, "commit", "--allow-empty", "-m", "fix: a")

	msgs, err := CommitsSince(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("got %d, want 2: %v", len(msgs), msgs)
	}
}

func TestCommitsSince_ExcludesMerges(t *testing.T) {
	gitAvailable(t)
	dir := initRepo(t)
	mustRun(t, dir, "commit", "--allow-empty", "-m", "feat: initial")
	mustRun(t, dir, "checkout", "-b", "branch")
	mustRun(t, dir, "commit", "--allow-empty", "-m", "feat: on branch")
	mustRun(t, dir, "checkout", "main")
	mustRun(t, dir, "commit", "--allow-empty", "-m", "fix: on main")
	mustRun(t, dir, "merge", "--no-ff", "-m", "merge: branch", "branch")

	msgs, err := CommitsSince(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range msgs {
		if strings.HasPrefix(m, "merge:") {
			t.Fatalf("merge commit leaked: %v", msgs)
		}
	}
}
