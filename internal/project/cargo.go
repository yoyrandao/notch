package project

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type cargo struct{}

const cargoFile = "Cargo.toml"

var cargoVersionRe = regexp.MustCompile(`^(version\s*=\s*")([^"]*)(".*)$`)

func (cargo) Name() string { return "cargo" }

func (cargo) Detect(repoDir string) (string, bool) {
	if fi, err := os.Stat(filepath.Join(repoDir, cargoFile)); err == nil && !fi.IsDir() {
		return cargoFile, true
	}
	return "", false
}

// SetVersion replaces version only under the [package] section to avoid
// touching dependency version fields in other sections.
func (cargo) SetVersion(absPath, version string) (bool, error) {
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return false, err
	}

	lines := strings.Split(string(raw), "\n")
	inPackage := false
	changed := false

	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			inPackage = strings.TrimSpace(line) == "[package]"
			continue
		}
		if inPackage {
			if m := cargoVersionRe.FindStringSubmatchIndex(line); m != nil {
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
