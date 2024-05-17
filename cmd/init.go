package cmd

import (
	"github.com/juanjiTech/jframe/cmd/config"
	"github.com/juanjiTech/jframe/cmd/create"
	"github.com/juanjiTech/jframe/cmd/server"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	Use:          "mod",
	Short:        "mod",
	SilenceUsage: true,
	Long:         `mod`,
}

func init() {
	rootCmd.AddCommand(server.StartCmd)
	rootCmd.AddCommand(config.StartCmd)
	rootCmd.AddCommand(create.StartCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}
