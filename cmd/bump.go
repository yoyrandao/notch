package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/yoyrandao/autotag/internal/conventional"
	"github.com/yoyrandao/autotag/internal/gitx"
	"github.com/yoyrandao/autotag/internal/semver"
)

func NewBumpCmd(root *rootFlags) *cobra.Command {
	var (
		repoPath  string
		preFlag   string
		releaseAs string
	)

	cmd := &cobra.Command{
		Use:   "bump",
		Short: "Compute the next semantic version from commits since the last tag",
		Long: `bump reads conventional commits since the last reachable git tag, computes
the next semantic version per SemVer rules, and prints it on stdout.

If no tag is found, the next version is v0.1.0. When no conventional commits
contributed a bump (and no --pre / --release-as is set), bump exits 0 and
prints "nothing to release" to stderr.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			repoDir, err := filepath.Abs(repoPath)
			if err != nil {
				return fmt.Errorf("resolve repo path: %w", err)
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
			msgs, err := gitx.CommitsSince(repoDir, ref)
			if err != nil {
				return err
			}

			commits := make([]conventional.Commit, 0, len(msgs))
			skipped := 0
			for _, m := range msgs {
				if c, ok := conventional.Parse(m); ok {
					commits = append(commits, c)
				} else {
					skipped++
				}
			}

			agg := semver.Aggregate(commits)

			opts := semver.Options{PreSuffix: preFlag}
			if releaseAs != "" {
				ra, err := semver.Parse(releaseAs)
				if err != nil {
					return fmt.Errorf("--release-as %q is not semver: %w", releaseAs, err)
				}
				opts.ReleaseAs = &ra
			}

			if root.verbose {
				lastStr := "<none>"
				if last != nil {
					lastStr = "v" + last.String()
				}
				fmt.Fprintf(cmd.ErrOrStderr(),
					"last=%s commits=%d skipped=%d bump=%v pre=%q release-as=%q\n",
					lastStr, len(commits), skipped, agg, preFlag, releaseAs,
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

			fmt.Fprintln(cmd.OutOrStdout(), "v"+next.String())
			return nil
		},
	}

	cmd.Flags().StringVar(&repoPath, "repo", ".", "path to git repository")
	cmd.Flags().StringVar(&preFlag, "pre", "", "prerelease suffix (e.g. rc, beta)")
	cmd.Flags().StringVar(&releaseAs, "release-as", "", "force next version (e.g. 1.0.0); must be > last tag")

	return cmd
}
