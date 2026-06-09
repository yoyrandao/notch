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

	// ReleaseMessage is the commit message for the release commit. Tokens {tag}
	// and {version} are substituted with the prefixed tag (e.g. v1.2.3) and the
	// plain semver (e.g. 1.2.3). Append "[skip ci]" to keep CI from triggering
	// on the release commit.
	ReleaseMessage string `koanf:"release_message"`
}

// PublishStep is a named step in the post-release publication pipeline.
type PublishStep struct {
	Name   string `koanf:"name"`
	Script string `koanf:"script"`
}

// PublishConfig defines the post-release publication pipeline.
// Steps run sequentially after push; any non-zero exit aborts the pipeline.
type PublishConfig struct {
	Steps []PublishStep `koanf:"steps"`
}

type Config struct {
	Repository string          `koanf:"repository"`
	Tag        TagConfig       `koanf:"tag"`
	Changelog  ChangelogConfig `koanf:"changelog"`
	Commit     CommitConfig    `koanf:"commit"`
	Publish    PublishConfig   `koanf:"publish"`
}

func DefaultPath() string { return ".notch.yaml" }

func DefaultYAML() string {
	return `repository: .

tag:
  prefix: "v"
  push: true

changelog:
  path: CHANGELOG.md

# commit:
#   # Extract the conventional message from wrapped merge-commit subjects.
#   # Capture group 1 is parsed as the commit. Disabled when unset.
#   # Example for Azure DevOps squash merges:
#   subject_pattern: '^Merged PR \d+: (.+)$'
#
#   # Release commit message. Tokens: {tag} (e.g. v1.2.3), {version} (e.g. 1.2.3).
#   # Append "[skip ci]" to stop CI triggering on the release commit.
#   release_message: "chore(release): {tag}"

# publish:
#   # Steps run sequentially after push. Any non-zero exit aborts the pipeline.
#   # Environment variables injected into each script:
#   #   NOTCH_TAG           full tag (e.g. v1.2.3)
#   #   NOTCH_VERSION       version without prefix (e.g. 1.2.3)
#   #   NOTCH_COMMIT        release commit SHA
#   #   NOTCH_CHANGELOG_PATH absolute path to CHANGELOG.md
#   #   NOTCH_REPOSITORY    absolute path to repository root
#   steps:
#     - name: upload-artifacts
#       script: ./scripts/upload.sh
#     - name: notify-slack
#       script: ./scripts/notify.sh
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
		Commit:     CommitConfig{ReleaseMessage: "chore(release): {tag}"},
		Publish:    PublishConfig{Steps: []PublishStep{}},
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
