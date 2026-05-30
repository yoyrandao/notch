// Package changelog renders and updates CHANGELOG.md (Keep a Changelog).
package changelog

import (
	"fmt"
	"os"
	"strings"

	"github.com/yoyrandao/autotag/internal/semconv"
)

type Kind int

const (
	KindOther Kind = iota
	KindFeature
	KindFix
	KindBreakingChange
)

// Entry is one line in the changelog.
type Entry struct {
	Kind        Kind
	Scope       string
	Description string
	Hash        string // full SHA; render shortens to 7 chars
}

const fileHeader = "# Changelog\n\nAll notable changes to this project are documented in this file. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).\n\n"

// ClassifyEntry converts a parsed commit + its hash into an Entry.
func ClassifyEntry(c semconv.Commit, hash string) Entry {
	e := Entry{Scope: c.Scope, Description: c.Description, Hash: hash}
	switch {
	case c.Breaking:
		e.Kind = KindBreakingChange
	case c.Type == "feat":
		e.Kind = KindFeature
	case c.Type == "fix":
		e.Kind = KindFix
	default:
		e.Kind = KindOther
	}
	return e
}

// Render produces the markdown section for one release, starting with the
// `## [version] - date` header and ending with a trailing newline.
func Render(version, date string, entries []Entry) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## [%s] - %s\n\n", version, date)

	type group struct {
		title   string
		entries []Entry
	}
	groups := []group{
		{"BREAKING CHANGES", nil},
		{"Added", nil},
		{"Fixed", nil},
		{"Changed", nil},
	}
	for _, e := range entries {
		switch e.Kind {
		case KindBreakingChange:
			groups[0].entries = append(groups[0].entries, e)
		case KindFeature:
			groups[1].entries = append(groups[1].entries, e)
		case KindFix:
			groups[2].entries = append(groups[2].entries, e)
		default:
			groups[3].entries = append(groups[3].entries, e)
		}
	}

	for _, g := range groups {
		if len(g.entries) == 0 {
			continue
		}
		fmt.Fprintf(&b, "### %s\n\n", g.title)
		for _, e := range g.entries {
			b.WriteString(formatBullet(e))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func formatBullet(e Entry) string {
	var b strings.Builder
	b.WriteString("- ")
	if e.Scope != "" {
		fmt.Fprintf(&b, "%s: ", e.Scope)
	}
	b.WriteString(e.Description)
	if e.Hash != "" {
		h := e.Hash
		if len(h) > 7 {
			h = h[:7]
		}
		fmt.Fprintf(&b, " (%s)", h)
	}
	b.WriteString("\n")
	return b.String()
}

// Update writes section into path. Creates the file with a Keep-a-Changelog
// header if missing; otherwise prepends section before the first existing
// `## ` release header (or appends if none).
func Update(path, section string) error {
	existing, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("changelog: read %s: %w", path, err)
		}
		return os.WriteFile(path, []byte(fileHeader+section), 0644)
	}

	content := string(existing)
	idx := findFirstRelease(content)

	var out string
	if idx == -1 {
		// No prior release sections; append after existing content
		// (ensuring a separating blank line).
		trimmed := strings.TrimRight(content, "\n")
		out = trimmed + "\n\n" + section
	} else {
		out = content[:idx] + section + content[idx:]
	}
	return os.WriteFile(path, []byte(out), 0644)
}

// findFirstRelease returns the byte index of the first `## ` line, or -1.
func findFirstRelease(s string) int {
	if strings.HasPrefix(s, "## ") {
		return 0
	}
	if i := strings.Index(s, "\n## "); i >= 0 {
		return i + 1
	}
	return -1
}
