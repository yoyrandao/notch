// Package config loads notch configuration via koanf.
package config

import (
	"errors"
	"os"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type TagConfig struct {
	Prefix string `koanf:"prefix"`
	Push   bool   `koanf:"push"`
}

type ChangelogConfig struct {
	Path string `koanf:"path"`
}

// CommitConfig tunes how raw commit messages are interpreted.
//
// SubjectPattern, when non-empty, is a regular expression with at least one
// capture group. It is applied to each commit's subject line; capture group 1
// becomes the message fed to the conventional-commit parser. This peels merge
// wrappers such as Azure DevOps's "Merged PR 123: <message>". Empty disables it.
type CommitConfig struct {
	SubjectPattern string `koanf:"subject_pattern"`
}

type Config struct {
	Repository string          `koanf:"repository"`
	Tag        TagConfig       `koanf:"tag"`
	Changelog  ChangelogConfig `koanf:"changelog"`
	Commit     CommitConfig    `koanf:"commit"`
}

func DefaultPath() string { return ".notch.yaml" }

func DefaultYAML() string {
	return `repository: .

tag:
  prefix: "v"
  push: true

changelog:
  path: CHANGELOG.md

# Extract the conventional message from wrapped merge-commit subjects.
# Capture group 1 is parsed as the commit. Disabled when unset.
# Example for Azure DevOps squash merges:
# commit:
#   subject_pattern: '^Merged PR \d+: (.+)$'
`
}

// LoadFile loads config from path; returns error if file is missing.
func LoadFile(path string) (*Config, error) {
	return load(path, true)
}

// LoadOptional loads config from path; missing file returns defaults without error.
func LoadOptional(path string) (*Config, error) {
	return load(path, false)
}

func defaultConfig() Config {
	return Config{
		Repository: ".",
		Tag:        TagConfig{Prefix: "v", Push: true},
		Changelog:  ChangelogConfig{Path: "CHANGELOG.md"},
	}
}

func load(path string, mustExist bool) (*Config, error) {
	k := koanf.New(".")

	err := k.Load(file.Provider(path), yaml.Parser())
	if err != nil {
		if mustExist || !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	// Pre-populate with defaults so fields absent from file keep their values.
	cfg := defaultConfig()
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
