package project

import (
	"os"
	"regexp"
	"strings"
)

type dotnet struct{}

var dotnetVersionRe = regexp.MustCompile(`(?m)^(\s*<Version>)([^<]*)(<\/Version>.*)$`)

func (dotnet) Name() string { return "dotnet" }

func (dotnet) Detect(repoDir string) (string, bool) {
	entries, err := os.ReadDir(repoDir)
	if err != nil {
		return "", false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".csproj") {
			return e.Name(), true
		}
	}
	return "", false
}

func (dotnet) SetVersion(absPath, version string) (bool, error) {
	return replaceVersion(absPath, dotnetVersionRe, version)
}
