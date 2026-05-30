package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/yoyrandao/autotag/internal/changelog"
	"github.com/yoyrandao/autotag/internal/conventional"
	"github.com/yoyrandao/autotag/internal/gitx"
	"github.com/yoyrandao/autotag/internal/semver"
)

type bumpOptions struct {
	*rootOptions
	preFlag       string
	releaseAs     string
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
commit, an annotated tag, and pushes them to the remote.

--dry-run skips all side effects (no file write, no commit, no tag, no push)
and prints the planned release to stderr.

If no commits contributed a bump (and no --pre / --release-as is set), bump
exits 0 and prints "nothing to release" to stderr.`,
		RunE: options.runE,
	}

	cmd.Flags().StringVar(&options.preFlag, "pre", "", "prerelease suffix (e.g. rc, beta)")
	cmd.Flags().StringVar(&options.releaseAs, "release-as", "", "force next version (e.g. 1.0.0); must be > last tag")
	cmd.Flags().BoolVar(&options.noPush, "no-push", false, "do not push commit/tag to remote")
	cmd.Flags().StringVar(&options.remote, "remote", "origin", "git remote name for push")
	cmd.Flags().StringVar(&options.changelogPath, "changelog", "CHANGELOG.md", "changelog path relative to --repo")

	return cmd
}

func (o *bumpOptions) runE(cmd *cobra.Command, args []string) error {
	repoDir, err := os.Getwd()
	if err != nil {
		return err
	}

	tagStr, found, err := gitx.LastTag(repoDir)
	if err != nil {
		return err
	}

	var last *semver.Version
	if found {
		v, err := semver.Parse(tagStr)
		if err != nil {
			return fmt.Errorf("last tag %q is not semver: %w", tagStr, err)
		}
		last = &v
	}

	ref := ""
	if found {
		ref = tagStr
	}
	rawCommits, err := gitx.Log(repoDir, ref)
	if err != nil {
		return err
	}

	parsed := make([]conventional.Commit, 0, len(rawCommits))
	entries := make([]changelog.Entry, 0, len(rawCommits))
	skipped := 0
	for _, rc := range rawCommits {
		c, ok := conventional.Parse(rc.Message)
		if !ok {
			skipped++
			continue
		}
		parsed = append(parsed, c)
		entries = append(entries, changelog.ClassifyEntry(c, rc.Hash))
	}

	agg := semver.Aggregate(parsed)

	opts := semver.Options{PreSuffix: o.preFlag}
	if o.releaseAs != "" {
		ra, err := semver.Parse(o.releaseAs)
		if err != nil {
			return fmt.Errorf("--release-as %q is not semver: %w", o.releaseAs, err)
		}
		opts.ReleaseAs = &ra
	}

	if o.verbose {
		lastStr := "<none>"
		if last != nil {
			lastStr = "v" + last.String()
		}
		fmt.Fprintf(cmd.ErrOrStderr(),
			"last=%s commits=%d skipped=%d bump=%v pre=%q release-as=%q dry-run=%v no-push=%v remote=%s\n",
			lastStr, len(parsed), skipped, agg, o.preFlag, o.releaseAs, o.dryRun, o.noPush, o.remote,
		)
	}

	if last != nil && agg == semver.BumpNone && opts.PreSuffix == "" && opts.ReleaseAs == nil && last.Prerelease == "" {
		fmt.Fprintln(cmd.ErrOrStderr(), "nothing to release")
		return nil
	}

	next, err := semver.Next(last, agg, opts)
	if err != nil {
		return err
	}

	version := next.String()
	tag := "v" + version
	date := time.Now().UTC().Format("2006-01-02")
	section := changelog.Render(version, date, entries)

	if o.dryRun {
		fmt.Fprintln(cmd.OutOrStdout(), tag)
		fmt.Fprintf(cmd.ErrOrStderr(),
			"dry-run: would update %s, commit \"chore(release): %s\", tag %s",
			o.changelogPath, tag, tag,
		)
		if !o.noPush {
			fmt.Fprintf(cmd.ErrOrStderr(), ", push HEAD and %s to %s", tag, o.remote)
		}
		fmt.Fprintln(cmd.ErrOrStderr())
		fmt.Fprintln(cmd.ErrOrStderr(), "--- CHANGELOG section ---")
		fmt.Fprint(cmd.ErrOrStderr(), section)
		return nil
	}

	absChangelog := o.changelogPath
	if !filepath.IsAbs(absChangelog) {
		absChangelog = filepath.Join(repoDir, o.changelogPath)
	}
	if err := changelog.Update(absChangelog, section); err != nil {
		return err
	}

	if err := gitx.CreateCommit(repoDir, "chore(release): "+tag, []string{o.changelogPath}); err != nil {
		return err
	}

	tagMessage := fmt.Sprintf("Release %s\n\n%s", tag, section)
	if err := gitx.CreateTag(repoDir, tag, tagMessage); err != nil {
		return err
	}

	if !o.noPush {
		if err := gitx.Push(repoDir, o.remote, "HEAD", tag); err != nil {
			return err
		}
	}

	fmt.Fprintln(cmd.OutOrStdout(), tag)
	return nil
}
