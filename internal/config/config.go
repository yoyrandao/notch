// Package config loads autotag configuration via koanf.
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

type Config struct {
	Repository string          `koanf:"repository"`
	Tag        TagConfig       `koanf:"tag"`
	Changelog  ChangelogConfig `koanf:"changelog"`
}

func DefaultPath() string { return ".autotag.yaml" }

func DefaultYAML() string {
	return `repository: .

tag:
  prefix: "v"
  push: true

changelog:
  path: CHANGELOG.md
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
