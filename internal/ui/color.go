// Package ui provides small terminal helpers for colored output.
package ui

import (
	"io"
	"os"

	"golang.org/x/term"
)

// ANSI SGR codes.
const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
)

// Colorizer wraps text in ANSI codes when enabled; otherwise returns it as-is.
type Colorizer struct {
	enabled bool
}

// New returns a Colorizer that emits color only when w is a terminal and the
// NO_COLOR environment variable (https://no-color.org) is unset.
func New(w io.Writer) Colorizer {
	if os.Getenv("NO_COLOR") != "" {
		return Colorizer{}
	}
	f, ok := w.(*os.File)
	if !ok {
		return Colorizer{}
	}
	return Colorizer{enabled: term.IsTerminal(int(f.Fd()))}
}

func (c Colorizer) paint(code, s string) string {
	if !c.enabled {
		return s
	}
	return code + s + reset
}

func (c Colorizer) Bold(s string) string   { return c.paint(bold, s) }
func (c Colorizer) Dim(s string) string    { return c.paint(dim, s) }
func (c Colorizer) Red(s string) string    { return c.paint(red, s) }
func (c Colorizer) Green(s string) string  { return c.paint(green, s) }
func (c Colorizer) Yellow(s string) string { return c.paint(yellow, s) }
func (c Colorizer) Cyan(s string) string   { return c.paint(cyan, s) }
