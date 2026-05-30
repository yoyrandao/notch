// Package conventional parses Conventional Commits messages.
package conventional

import (
	"regexp"
	"strings"
)

// Commit is a parsed Conventional Commit message.
type Commit struct {
	Type        string
	Scope       string
	Description string
	Body        string
	Breaking    bool
}

var headerRE = regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9]*)(?:\(([^)]+)\))?(!?): (.+)$`)

// Parse returns (commit, true) if msg matches Conventional Commits format,
// otherwise (zero, false). Non-conv messages are not errors — they are simply
// skipped by callers when classifying.
func Parse(msg string) (Commit, bool) {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return Commit{}, false
	}

	header, body, _ := strings.Cut(msg, "\n")
	m := headerRE.FindStringSubmatch(header)
	if m == nil {
		return Commit{}, false
	}

	c := Commit{
		Type:        strings.ToLower(m[1]),
		Scope:       m[2],
		Description: strings.TrimSpace(m[4]),
		Breaking:    m[3] == "!",
		Body:        strings.TrimSpace(body),
	}

	if !c.Breaking && c.Body != "" {
		for ln := range strings.SplitSeq(c.Body, "\n") {
			ln = strings.TrimSpace(ln)
			if strings.HasPrefix(ln, "BREAKING CHANGE:") || strings.HasPrefix(ln, "BREAKING-CHANGE:") {
				c.Breaking = true
				break
			}
		}
	}

	return c, true
}
