package project

import (
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type pom struct{}

const pomFile = "pom.xml"

func (pom) Name() string { return "maven" }

func (pom) Detect(repoDir string) (string, bool) {
	if fi, err := os.Stat(filepath.Join(repoDir, pomFile)); err == nil && !fi.IsDir() {
		return pomFile, true
	}
	return "", false
}

// SetVersion replaces only the <version> that is a direct child of <project>
// (depth 2 in the token stream). Parent, dependency, and plugin versions are
// at depth 3+ and are left untouched.
func (pom) SetVersion(absPath, version string) (bool, error) {
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return false, err
	}

	content := string(raw)
	dec := xml.NewDecoder(strings.NewReader(content))

	depth := 0
	inVersion := false
	var valueStart, valueEnd int64
	found := false

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return false, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			if depth == 2 && t.Name.Local == "version" {
				inVersion = true
			}
		case xml.CharData:
			if inVersion {
				end := dec.InputOffset()
				valueStart = end - int64(len(t))
				valueEnd = end
				found = true
				inVersion = false
			}
		case xml.EndElement:
			if depth == 2 && t.Name.Local == "version" {
				inVersion = false
			}
			depth--
		}

		if found {
			break
		}
	}

	if !found {
		return false, nil
	}

	out := content[:valueStart] + version + content[valueEnd:]
	if out == content {
		return false, nil
	}
	return true, writeFilePreservingMode(absPath, []byte(out))
}
