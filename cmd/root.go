package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type rootFlags struct {
	cfgFile string
	dryRun  bool
	verbose bool
}

func NewRootCmd() *cobra.Command {
	flags := &rootFlags{}

	cmd := &cobra.Command{
		Use:   "autotag",
		Short: "Automated semantic versioning from conventional commits",
		Long: `autotag reads conventional commit history, computes the next semantic
version, updates CHANGELOG.md, creates an annotated git tag, pushes it and
runs optional user-defined automations.`,
		SilenceUsage: true,
	}

	cmd.PersistentFlags().StringVarP(&flags.cfgFile, "config", "c", "", "config file (default: .autotag.yaml)")
	cmd.PersistentFlags().BoolVar(&flags.dryRun, "dry-run", false, "compute but do not write/tag/push")
	cmd.PersistentFlags().BoolVarP(&flags.verbose, "verbose", "v", false, "verbose output")

	cmd.AddCommand(NewVersionCmd())
	cmd.AddCommand(NewBumpCmd(flags))

	return cmd
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
