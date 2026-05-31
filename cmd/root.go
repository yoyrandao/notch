package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/yoyrandao/notch/internal/config"
)

type rootOptions struct {
	cfgFile string
	dryRun  bool
	verbose bool
	config  *config.Config
}

func NewRootCmd() *cobra.Command {
	options := &rootOptions{}
	cmd := &cobra.Command{
		Use:   "notch",
		Short: "Automated semantic versioning from conventional commits",
		Long: `notch computes next semantic version based on conventional commit history,
updates CHANGELOG.md and creates annotated tags for your git repository.`,
		SilenceUsage: true,
	}

	cmd.PersistentFlags().StringVarP(&options.cfgFile, "config", "c", "", "config file (default: .notch.yaml)")
	cmd.PersistentFlags().BoolVar(&options.dryRun, "dry-run", false, "display bump artifacts only (next version and changelog)")
	cmd.PersistentFlags().BoolVarP(&options.verbose, "verbose", "v", false, "verbose output")

	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		var cfg *config.Config
		var err error

		if options.cfgFile != "" {
			cfg, err = config.LoadFile(options.cfgFile)
		} else {
			cfg, err = config.LoadOptional(config.DefaultPath())
		}

		if err != nil {
			return err
		}

		options.config = cfg
		return nil
	}

	cmd.AddCommand(NewVersionCmd(), NewBumpCmd(options), NewInitCmd())
	return cmd
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
