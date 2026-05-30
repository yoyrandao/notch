// Package gitx wraps the `git` CLI for reading history and writing tags.
package gitx

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// LastTag returns the most recent annotated/lightweight tag reachable from
// HEAD via `git describe --tags --abbrev=0`. found=false when no tag exists.
func LastTag(repoDir string) (tag string, found bool, err error) {
	stdout, stderr, runErr := runGit(repoDir, "describe", "--tags", "--abbrev=0")
	if runErr != nil {
		var ee *exec.ExitError
		if errors.As(runErr, &ee) {
			msg := stderr.String()
			if strings.Contains(msg, "No names found") || strings.Contains(msg, "No tags can describe") {
				return "", false, nil
			}
		}
		return "", false, fmt.Errorf("git describe: %w: %s", runErr, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), true, nil
}

// CommitsSince returns raw commit messages reachable from HEAD but not from
// ref, oldest-first not guaranteed (git default is newest-first; order is
// irrelevant for bump aggregation). Merge commits are excluded. When ref is
// empty, all commits reachable from HEAD are returned.
func CommitsSince(repoDir, ref string) ([]string, error) {
	args := []string{"log", "--no-merges", "--format=%B%x00"}
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

	parts := bytes.Split(raw, []byte{0})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(string(p))
		if s != "" {
			out = append(out, s)
		}
	}
	return out, nil
}

func runGit(repoDir string, args ...string) (stdout, stderr bytes.Buffer, err error) {
	full := append([]string{"-C", repoDir}, args...)
	cmd := exec.Command("git", full...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	return
}
