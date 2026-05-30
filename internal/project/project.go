// Package project patches the version stored in tool-specific project files
// (Helm Chart.yaml, npm package.json, ...). Detection is implicit: if the
// target directory holds no recognised project, or the recognised file lacks a
// version field, nothing is changed and no error is returned — the caller
// behaves exactly as for a plain repository.
package project

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Patcher knows how to find and rewrite the version of one project type.
type Patcher interface {
	// Name is a short identifier ("helm", "npm").
	Name() string
	// Detect reports the version file (relative to repoDir) if this project
	// type is present, with ok=false otherwise.
	Detect(repoDir string) (relPath string, ok bool)
	// SetVersion rewrites the version in absPath. changed reports whether the
	// file content actually changed (false when the version field is absent or
	// already equal). A missing field is not an error.
	SetVersion(absPath, version string) (changed bool, err error)
}

// registry is the ordered set of supported project types. Adding a type later
// is one struct plus one entry here.
var registry = []Patcher{helm{}, npm{}}

// Detect returns the version files (relative to repoDir) of every recognised
// project type present. Read-only; safe for dry-run.
func Detect(repoDir string) []string {
	var out []string
	for _, p := range registry {
		if rel, ok := p.Detect(repoDir); ok {
			out = append(out, rel)
		}
	}
	return out
}

// Patch rewrites the version (plain semver, no tag prefix) in every recognised
// project file under repoDir. It returns the relative paths of files that
// actually changed, for staging. A project that is absent, or whose version
// field is missing, contributes nothing and is not an error.
func Patch(repoDir, version string) (changedRel []string, err error) {
	for _, p := range registry {
		rel, ok := p.Detect(repoDir)
		if !ok {
			continue
		}
		changed, err := p.SetVersion(joinRepo(repoDir, rel), version)
		if err != nil {
			return nil, err
		}
		if changed {
			changedRel = append(changedRel, rel)
		}
	}
	return changedRel, nil
}

// replaceVersion rewrites the value captured by re's group 2 with version,
// keeping groups 1 and 3 (and the rest of the file) byte-for-byte. It writes
// only when content changes. A non-matching file changes nothing and is not an
// error. re must have exactly three capture groups.
func replaceVersion(absPath string, re *regexp.Regexp, version string) (bool, error) {
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return false, err
	}
	repl := "${1}" + escapeReplacement(version) + "${3}"
	out := re.ReplaceAllString(string(raw), repl)
	if out == string(raw) {
		return false, nil
	}
	if err := writeFilePreservingMode(absPath, []byte(out)); err != nil {
		return false, err
	}
	return true, nil
}

// escapeReplacement neutralises `$` so version is inserted literally by
// Regexp.ReplaceAllString.
func escapeReplacement(s string) string { return strings.ReplaceAll(s, "$", "$$") }

// joinRepo joins a repo-relative path onto repoDir.
func joinRepo(repoDir, rel string) string { return filepath.Join(repoDir, rel) }

// writeFilePreservingMode rewrites path with content, keeping the file's
// existing permission bits (0644 fallback).
func writeFilePreservingMode(path string, content []byte) error {
	mode := os.FileMode(0o644)
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}
	return os.WriteFile(path, content, mode)
}
