package project

import (
	"os"
	"path/filepath"
	"regexp"
)

// helm patches the `version` field of a Helm chart's Chart.yaml.
type helm struct{}

const helmFile = "Chart.yaml"

// helmVersionRe matches a top-level `version:` key (no leading indentation, so
// nested keys and `appVersion:` are not touched). Groups: 1 = key + optional
// opening quote, 2 = value, 3 = optional closing quote + trailing comment.
var helmVersionRe = regexp.MustCompile(`(?m)^(version:[ \t]*["']?)([^"'\r\n]*?)(["']?[ \t]*(?:#.*)?)$`)

func (helm) Name() string { return "helm" }

func (helm) Detect(repoDir string) (string, bool) {
	if fi, err := os.Stat(filepath.Join(repoDir, helmFile)); err == nil && !fi.IsDir() {
		return helmFile, true
	}
	return "", false
}

func (helm) SetVersion(absPath, version string) (bool, error) {
	return replaceVersion(absPath, helmVersionRe, version)
}
