package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/yoyrandao/autotag/internal/config"
)

func NewInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create default .autotag.yaml in current directory",
		Long:  `init writes .autotag.yaml with all default values. Exits with error if file already exists.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := config.DefaultPath()
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("%s already exists", path)
			}
			return os.WriteFile(path, []byte(config.DefaultYAML()), 0o644)
		},
	}
}
