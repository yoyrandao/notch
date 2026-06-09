package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/yoyrandao/notch/internal/changelog"
	"github.com/yoyrandao/notch/internal/gitx"
	"github.com/yoyrandao/notch/internal/project"
	"github.com/yoyrandao/notch/internal/publisher"
	"github.com/yoyrandao/notch/internal/semconv"
	"github.com/yoyrandao/notch/internal/semver"
	"github.com/yoyrandao/notch/internal/ui"
)

type bumpOptions struct {
	*rootOptions
	repository    string
	pre           string
	as            string
	noPush        bool
	remote        string
	changelogPath string
}

func newBumpOptions(o *rootOptions) *bumpOptions {
	return &bumpOptions{rootOptions: o}
}

func NewBumpCmd(root *rootOptions) *cobra.Command {
	options := newBumpOptions(root)
	cmd := &cobra.Command{
		Use:   "bump",
		Short: "Compute next version, update CHANGELOG, tag and push",
		Long: `bump reads conventional commits since the last reachable git tag,
computes the next semantic version, updates CHANGELOG.md, creates a release
commit, an annotated tag, which can be pushed to a remote.

If no commits contributed a bump (and no --pre / --as is set), only "nothing to release" will be printed.`,
		RunE: options.runE,
	}

	cmd.Flags().StringVar(&options.repository, "repo", "", "path to git repository (default: .)")
	cmd.Flags().StringVar(&options.pre, "pre", "", "prerelease suffix (e.g. rc, beta)")
	cmd.Flags().StringVar(&options.as, "as", "", "force next version (e.g. 1.0.0); must be > last tag")
	cmd.Flags().BoolVar(&options.noPush, "no-push", false, "do not push commit/tag to remote")
	cmd.Flags().StringVar(&options.remote, "remote", "origin", "git remote name for push")
	cmd.Flags().StringVar(&options.changelogPath, "changelog", "", "changelog path relative to --repo")

	return cmd
}

// release holds everything computed for a planned release.
type release struct {
	tag     string
	version string
	section string
	last    *semver.Version
}

func (o *bumpOptions) runE(cmd *cobra.Command, args []string) error {
	o.applyConfig(cmd)

	repoDir, err := o.resolveRepositoryDirectory()
	if err != nil {
		return err
	}

	last, lastTag, err := o.lastVersion(repoDir)
	if err != nil {
		return err
	}

	pattern, err := o.commitPattern()
	if err != nil {
		return err
	}

	parsed, entries, skipped, err := collectCommits(repoDir, lastTag, pattern)
	if err != nil {
		return err
	}

	opts, err := o.semverOptions()
	if err != nil {
		return err
	}

	agg := semver.Aggregate(parsed)
	if o.verbose {
		o.logPlan(cmd, last, len(parsed), skipped, agg)
	}

	if nothingToRelease(last, agg, opts) {
		c := ui.New(cmd.ErrOrStderr())
		fmt.Fprintln(cmd.ErrOrStderr(), c.Yellow("nothing to release"))
		return nil
	}

	rel, err := o.computeRelease(last, agg, opts, entries)
	if err != nil {
		return err
	}

	if o.dryRun {
		o.printDryRun(cmd, repoDir, rel)
		return nil
	}

	return o.executeRelease(cmd, repoDir, rel)
}

// applyConfig fills options from config for flags not explicitly set on the CLI.
func (o *bumpOptions) applyConfig(cmd *cobra.Command) {
	cfg := o.config
	if !cmd.Flags().Changed("repo") {
		o.repository = cfg.Repository
	}
	if !cmd.Flags().Changed("changelog") {
		o.changelogPath = cfg.Changelog.Path
	}
	if !cmd.Flags().Changed("no-push") {
		o.noPush = !cfg.Tag.Push
	}
}

// resolveRepositoryDirectory turns the configured repo path into an absolute directory.
func (o *bumpOptions) resolveRepositoryDirectory() (string, error) {
	if o.repository != "." && o.repository != "" {
		if filepath.IsAbs(o.repository) {
			return o.repository, nil
		}
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return filepath.Join(cwd, o.repository), nil
	}
	return os.Getwd()
}

// lastVersion reads the most recent tag and parses it. Returns the parsed
// version, the raw tag string (git ref), and nil version if no tag exists.
func (o *bumpOptions) lastVersion(repoDir string) (*semver.Version, string, error) {
	tag, found, err := gitx.LastTag(repoDir)
	if err != nil || !found {
		return nil, "", err
	}
	v, err := semver.Parse(strings.TrimPrefix(tag, o.config.Tag.Prefix))
	if err != nil {
		return nil, "", fmt.Errorf("last tag %q is not semver: %w", tag, err)
	}
	return &v, tag, nil
}

// commitPattern compiles the configured subject-unwrap regex. Returns nil when
// unset (feature disabled). Errors when the regex is invalid or lacks a capture
// group, since capture group 1 supplies the message to parse.
func (o *bumpOptions) commitPattern() (*regexp.Regexp, error) {
	p := o.config.Commit.SubjectPattern
	if p == "" {
		return nil, nil
	}
	re, err := regexp.Compile(p)
	if err != nil {
		return nil, fmt.Errorf("commit.subject_pattern %q is not a valid regexp: %w", p, err)
	}
	if re.NumSubexp() < 1 {
		return nil, fmt.Errorf("commit.subject_pattern %q must have a capture group for the payload", p)
	}
	return re, nil
}

// collectCommits gathers conventional commits since the given ref (empty = all).
// When pattern is non-nil, each commit subject is unwrapped through it before
// parsing (see semconv.Unwrap).
func collectCommits(repoDir, ref string, pattern *regexp.Regexp) (parsed []semconv.Commit, entries []changelog.Entry, skipped int, err error) {
	rawCommits, err := gitx.Log(repoDir, ref)
	if err != nil {
		return nil, nil, 0, err
	}

	parsed = make([]semconv.Commit, 0, len(rawCommits))
	entries = make([]changelog.Entry, 0, len(rawCommits))
	for _, rc := range rawCommits {
		c, ok := semconv.Parse(semconv.Unwrap(rc.Message, pattern))
		if !ok {
			skipped++
			continue
		}
		parsed = append(parsed, c)
		entries = append(entries, changelog.ClassifyEntry(c, rc.Hash))
	}
	return parsed, entries, skipped, nil
}

// semverOptions builds bump options from the --pre / --as flags.
func (o *bumpOptions) semverOptions() (semver.Options, error) {
	opts := semver.Options{PreSuffix: o.pre}
	if o.as != "" {
		ra, err := semver.Parse(o.as)
		if err != nil {
			return opts, fmt.Errorf("--as %q is not semver: %w", o.as, err)
		}
		opts.ReleaseAs = &ra
	}
	return opts, nil
}

func nothingToRelease(last *semver.Version, agg semver.Bump, opts semver.Options) bool {
	return last != nil && agg == semver.BumpNone &&
		opts.PreSuffix == "" && opts.ReleaseAs == nil && last.PreRelease == ""
}

// computeRelease derives the next version and renders the changelog section.
func (o *bumpOptions) computeRelease(last *semver.Version, agg semver.Bump, opts semver.Options, entries []changelog.Entry) (release, error) {
	next, err := semver.Next(last, agg, opts)
	if err != nil {
		return release{}, err
	}
	version := next.String()
	date := time.Now().UTC().Format("2006-01-02")
	return release{
		tag:     o.config.Tag.Prefix + version,
		version: version,
		section: changelog.Render(version, date, entries),
		last:    last,
	}, nil
}

// releaseCommitMessage renders the configured release-commit message, substituting
// {tag} (prefixed, e.g. v1.2.3) and {version} (plain semver, e.g. 1.2.3).
func (o *bumpOptions) releaseCommitMessage(rel release) string {
	return strings.NewReplacer("{tag}", rel.tag, "{version}", rel.version).
		Replace(o.config.Commit.ReleaseMessage)
}

func (o *bumpOptions) logPlan(cmd *cobra.Command, last *semver.Version, commits, skipped int, agg semver.Bump) {
	lastStr := "<none>"
	if last != nil {
		lastStr = o.config.Tag.Prefix + last.String()
	}
	fmt.Fprintf(cmd.ErrOrStderr(),
		"last=%s commits=%d skipped=%d bump=%v pre=%q as=%q dry-run=%v no-push=%v remote=%s\n",
		lastStr, commits, skipped, agg, o.pre, o.as, o.dryRun, o.noPush, o.remote,
	)
}

func (o *bumpOptions) printDryRun(cmd *cobra.Command, repoDir string, rel release) {
	out, errOut := cmd.OutOrStdout(), cmd.ErrOrStderr()
	c := ui.New(errOut)

	steps := []string{
		fmt.Sprintf("update %s", o.changelogPath),
	}
	for _, f := range project.Detect(repoDir) {
		steps = append(steps, fmt.Sprintf("patch %s", f))
	}
	steps = append(steps,
		fmt.Sprintf("commit %q", o.releaseCommitMessage(rel)),
		fmt.Sprintf("tag %s", c.Green(rel.tag)),
	)
	if !o.noPush {
		steps = append(steps, fmt.Sprintf("push HEAD and %s to %s", c.Green(rel.tag), o.remote))
	}
	for _, s := range o.config.Publish.Steps {
		steps = append(steps, fmt.Sprintf("publish[%s] %s", s.Name, s.Script))
	}

	fmt.Fprintln(out, rel.tag)
	fmt.Fprintf(errOut, "%s would %s\n", c.Bold(c.Cyan("dry-run:")), strings.Join(steps, ", "))
	fmt.Fprintln(errOut, c.Dim("--- CHANGELOG section ---"))
	fmt.Fprint(errOut, rel.section)
}

// executeRelease writes the changelog, commits, tags and optionally pushes.
func (o *bumpOptions) executeRelease(cmd *cobra.Command, repoDir string, rel release) error {
	absChangelog := o.changelogPath
	if !filepath.IsAbs(absChangelog) {
		absChangelog = filepath.Join(repoDir, o.changelogPath)
	}
	if err := changelog.Update(absChangelog, rel.section); err != nil {
		return err
	}

	// Patch tool-specific project files (Chart.yaml, package.json, ...) with the
	// plain semver. Absent project / missing version field => nothing staged,
	// no error: behaves exactly like a plain repo.
	patched, err := project.Patch(repoDir, rel.version)
	if err != nil {
		return err
	}

	paths := append([]string{o.changelogPath}, patched...)
	if err := gitx.CreateCommit(repoDir, o.releaseCommitMessage(rel), paths); err != nil {
		return err
	}

	tagMessage := fmt.Sprintf("Release %s\n\n%s", rel.tag, rel.section)
	if err := gitx.CreateTag(repoDir, rel.tag, tagMessage); err != nil {
		return err
	}

	if !o.noPush {
		if err := gitx.Push(repoDir, o.remote, "HEAD", rel.tag); err != nil {
			return err
		}
	}

	if len(o.config.Publish.Steps) > 0 {
		commit, err := gitx.Head(repoDir)
		if err != nil {
			return err
		}
		steps := make([]publisher.Step, len(o.config.Publish.Steps))
		for i, s := range o.config.Publish.Steps {
			steps[i] = publisher.Step{Name: s.Name, Script: s.Script}
		}
		pubEnv := publisher.Env{
			Tag:           rel.tag,
			Version:       rel.version,
			Commit:        commit,
			ChangelogPath: absChangelog,
			Repository:    repoDir,
		}
		if err := publisher.Run(steps, pubEnv); err != nil {
			return err
		}
	}

	c := ui.New(cmd.ErrOrStderr())
	fmt.Fprintf(cmd.ErrOrStderr(), "%s released %s\n", c.Green("✓"), c.Bold(rel.tag))
	fmt.Fprintln(cmd.OutOrStdout(), rel.tag)
	return nil
}
