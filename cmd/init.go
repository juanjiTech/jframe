package cmd

import (
	"github.com/juanjiTech/jframe/cmd/config"
	"github.com/juanjiTech/jframe/cmd/create"
	"github.com/juanjiTech/jframe/cmd/server"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	Use:          "jframe",
	SilenceUsage: true,
	Short:        "jframe is an AI-friendly Golang framework for server and desktop development with quality control",
	Example:      "jframe server -c ./config.yaml",
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
