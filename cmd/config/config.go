package config

import (
	"fmt"
	"github.com/juanjiTech/jframe/conf"
	"github.com/spf13/cobra"
	"os"
)

var (
	configPath string
	forceGen   bool
	StartCmd   = &cobra.Command{
		Use:     "config",
		Short:   "Generate config file",
		Example: "jframe config -p ./config.yaml -f",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Generating config...")
			err := load()
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
		},
	}
)

func init() {
	StartCmd.PersistentFlags().StringVarP(&configPath, "path", "p", "./config.yaml", "Generate config in provided path")
	StartCmd.PersistentFlags().BoolVarP(&forceGen, "force", "f", false, "Force generate config in provided path")
}

func load() error {
	return conf.GenYamlConfig(configPath, forceGen)
}
