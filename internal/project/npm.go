package project

import (
	"os"
	"path/filepath"
	"regexp"
)

// npm patches the "version" field of a package.json.
type npm struct{}

const npmFile = "package.json"

// npmVersionRe matches the `"version": "..."` member. Groups: 1 = key through
// opening quote, 2 = value, 3 = closing quote + trailing.
var npmVersionRe = regexp.MustCompile(`(?m)^(\s*"version"\s*:\s*")([^"]*)(".*)$`)

func (npm) Name() string { return "npm" }

func (npm) Detect(repoDir string) (string, bool) {
	if fi, err := os.Stat(filepath.Join(repoDir, npmFile)); err == nil && !fi.IsDir() {
		return npmFile, true
	}
	return "", false
}

func (npm) SetVersion(absPath, version string) (bool, error) {
	return replaceVersion(absPath, npmVersionRe, version)
}
