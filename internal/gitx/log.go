// Package gitx wraps the `git` CLI for reading history and writing tags.
package gitx

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Commit is one git commit record.
type Commit struct {
	Hash    string
	Message string
}

// ErrNothingStaged is returned by CreateCommit when `git commit` reports
// nothing to commit.
var ErrNothingStaged = errors.New("gitx: nothing staged")

// LastTag returns the most recent annotated/lightweight tag reachable from
// HEAD via `git describe --tags --abbrev=0`. found=false when no tag exists.
func LastTag(repoDir string) (tag string, found bool, err error) {
	stdout, stderr, runErr := runGit(repoDir, "describe", "--tags", "--abbrev=0")
	if runErr != nil {
		if _, ok := errors.AsType[*exec.ExitError](runErr); ok {
			msg := stderr.String()
			if strings.Contains(msg, "No names found") || strings.Contains(msg, "No tags can describe") {
				return "", false, nil
			}
		}
		return "", false, fmt.Errorf("git describe: %w: %s", runErr, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), true, nil
}

// Log returns Commits reachable from HEAD but not from ref. When ref is empty,
// all commits reachable from HEAD are returned. Merge commits are excluded.
func Log(repoDir, ref string) ([]Commit, error) {
	args := []string{"log", "--no-merges", "--format=%H%x1f%B%x00"}
	if ref != "" {
		args = append(args, ref+"..HEAD")
	}

	stdout, stderr, err := runGit(repoDir, args...)
	if err != nil {
		return nil, fmt.Errorf("git log: %w: %s", err, stderr.String())
	}

	raw := bytes.TrimRight(stdout.Bytes(), "\x00\n")
	if len(raw) == 0 {
		return nil, nil
	}

	records := bytes.Split(raw, []byte{0})
	out := make([]Commit, 0, len(records))
	for _, r := range records {
		fields := bytes.SplitN(r, []byte{0x1f}, 2)
		if len(fields) != 2 {
			continue
		}
		msg := strings.TrimSpace(string(fields[1]))
		if msg == "" {
			continue
		}
		out = append(out, Commit{
			Hash:    strings.TrimSpace(string(fields[0])),
			Message: msg,
		})
	}

	return out, nil
}

// CommitsSince returns commit messages reachable from HEAD but not from ref.
// Thin wrapper around Log.
func CommitsSince(repoDir, ref string) ([]string, error) {
	cs, err := Log(repoDir, ref)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Message
	}
	return out, nil
}

// CreateCommit stages the given paths (relative to repoDir) and creates a
// commit with the given message. Returns ErrNothingStaged when there is
// nothing to commit.
func CreateCommit(repoDir, message string, paths []string) error {
	if len(paths) > 0 {
		args := append([]string{"add", "--"}, paths...)
		if _, stderr, err := runGit(repoDir, args...); err != nil {
			return fmt.Errorf("git add: %w: %s", err, stderr.String())
		}
	}

	stdout, stderr, err := runGit(repoDir, "commit", "-m", message)
	if err != nil {
		combined := stdout.String() + stderr.String()
		if strings.Contains(combined, "nothing to commit") || strings.Contains(combined, "no changes added") {
			return ErrNothingStaged
		}
		return fmt.Errorf("git commit: %w: %s", err, combined)
	}

	return nil
}

// CreateTag creates an annotated tag at HEAD with the given message.
func CreateTag(repoDir, name, message string) error {
	if _, stderr, err := runGit(repoDir, "tag", "-a", name, "-m", message); err != nil {
		return fmt.Errorf("git tag: %w: %s", err, stderr.String())
	}

	return nil
}

// Push pushes the given refs to remote.
func Push(repoDir, remote string, refs ...string) error {
	args := append([]string{"push", remote}, refs...)
	if _, stderr, err := runGit(repoDir, args...); err != nil {
		return fmt.Errorf("git push: %w: %s", err, stderr.String())
	}

	return nil
}

func runGit(repoDir string, args ...string) (stdout, stderr bytes.Buffer, err error) {
	full := append([]string{"-C", repoDir}, args...)
	cmd := exec.Command("git", full...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	return
}
