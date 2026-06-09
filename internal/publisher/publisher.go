// Package publisher executes user-defined publication steps after a release.
package publisher

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/yoyrandao/notch/internal/ui"
)

// Step is a named publication step with an associated script path.
type Step struct {
	Name   string
	Script string
}

// Env carries release context injected into each script as environment variables:
//
//	NOTCH_TAG           full tag (e.g. v1.2.3)
//	NOTCH_VERSION       version without prefix (e.g. 1.2.3)
//	NOTCH_COMMIT        release commit SHA
//	NOTCH_CHANGELOG_PATH absolute path to CHANGELOG.md
//	NOTCH_REPOSITORY    absolute path to repository root
type Env struct {
	Tag           string
	Version       string
	Commit        string
	ChangelogPath string
	Repository    string
}

// Run executes each step sequentially. Returns on first failure, wrapping the
// error with the step name.
func Run(steps []Step, env Env) error {
	c := ui.New(os.Stderr)

	for _, step := range steps {
		fmt.Fprintf(os.Stderr, "running %s...\n", c.Green(step.Name))
		if err := runStep(step, env); err != nil {
			return fmt.Errorf("publish step %q: %w", step.Name, err)
		}
		fmt.Fprintf(os.Stderr, "%s finished.\n", c.Green(step.Name))
	}
	return nil
}

func runStep(step Step, env Env) error {
	cmd := buildCommand(step.Script)
	cmd.Env = append(os.Environ(),
		"NOTCH_TAG="+env.Tag,
		"NOTCH_VERSION="+env.Version,
		"NOTCH_COMMIT="+env.Commit,
		"NOTCH_CHANGELOG_PATH="+env.ChangelogPath,
		"NOTCH_REPOSITORY="+env.Repository,
	)
	outW := newPrefixWriter(os.Stdout)
	errW := newPrefixWriter(os.Stderr)
	cmd.Stdout = outW
	cmd.Stderr = errW
	err := cmd.Run()
	outW.flush()
	errW.flush()
	return err
}

type prefixWriter struct {
	w   io.Writer
	buf []byte
}

func newPrefixWriter(w io.Writer) *prefixWriter { return &prefixWriter{w: w} }

func (p *prefixWriter) Write(b []byte) (int, error) {
	p.buf = append(p.buf, b...)
	for {
		idx := bytes.IndexByte(p.buf, '\n')
		if idx < 0 {
			break
		}
		if _, err := fmt.Fprintf(p.w, "  > %s", p.buf[:idx+1]); err != nil {
			return 0, err
		}
		p.buf = p.buf[idx+1:]
	}
	return len(b), nil
}

func (p *prefixWriter) flush() {
	if len(p.buf) > 0 {
		fmt.Fprintf(p.w, "  > %s\n", p.buf)
		p.buf = p.buf[:0]
	}
}

func buildCommand(script string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("powershell.exe", "-File", script)
	}
	return exec.Command("bash", script)
}
