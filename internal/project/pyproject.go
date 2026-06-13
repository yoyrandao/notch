package project

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type pyproject struct{}

const pyprojectFile = "pyproject.toml"

var pyprojectVersionRe = regexp.MustCompile(`^(version\s*=\s*")([^"]*)(".*)$`)

func (pyproject) Name() string { return "pyproject" }

func (pyproject) Detect(repoDir string) (string, bool) {
	if fi, err := os.Stat(filepath.Join(repoDir, pyprojectFile)); err == nil && !fi.IsDir() {
		return pyprojectFile, true
	}
	return "", false
}

// SetVersion replaces version only under the [project] section to avoid
// touching version fields in [tool.*] or other sections.
func (pyproject) SetVersion(absPath, version string) (bool, error) {
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return false, err
	}

	lines := strings.Split(string(raw), "\n")
	inProject := false
	changed := false

	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			inProject = strings.TrimSpace(line) == "[project]"
			continue
		}
		if inProject {
			if m := pyprojectVersionRe.FindStringSubmatchIndex(line); m != nil {
				lines[i] = line[:m[4]] + version + line[m[5]:]
				changed = true
				break
			}
		}
	}

	if !changed {
		return false, nil
	}
	return true, writeFilePreservingMode(absPath, []byte(strings.Join(lines, "\n")))
}
