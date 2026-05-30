package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print autotag version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("autotag %s (commit %s, built %s)\n", version, commit, date)
		},
	}
}
