package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestNew_NonTerminalDisabled(t *testing.T) {
	// bytes.Buffer is not an *os.File, so color must be disabled.
	c := New(&bytes.Buffer{})
	if got := c.Green("ok"); got != "ok" {
		t.Fatalf("expected plain %q, got %q", "ok", got)
	}
}

func TestNew_NoColorEnv(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	c := New(&bytes.Buffer{})
	if got := c.Red("x"); strings.Contains(got, "\033") {
		t.Fatalf("NO_COLOR set but ANSI emitted: %q", got)
	}
}

func TestColorizer_EnabledPaints(t *testing.T) {
	c := Colorizer{enabled: true}
	got := c.Green("ok")
	if !strings.HasPrefix(got, green) || !strings.HasSuffix(got, reset) {
		t.Fatalf("expected wrapped string, got %q", got)
	}
}
