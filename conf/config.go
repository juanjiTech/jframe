package conf

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/juanjiTech/jframe/pkg/fsx"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"os"
)

var SysVersion = "dev"

var serveConfig *GlobalConfig

func LoadConfig(configPath ...string) {
	if len(configPath) == 0 || configPath[0] == "" {
		viper.SetConfigName("config")
		viper.AddConfigPath(".")
		viper.AddConfigPath("./config")
	} else {
		viper.SetConfigFile(configPath[0])
	}

	loadConfig := func() {
		newConf := new(GlobalConfig)
		err := viper.ReadInConfig()
		if err != nil {
			fmt.Println("Config Read failed: " + err.Error())
			os.Exit(1)
		}
		err = viper.Unmarshal(newConf)
		if err != nil {
			fmt.Println("Config Unmarshal failed: " + err.Error())
			os.Exit(1)
		}
		serveConfig = newConf
	}

	loadConfig()

	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config fileHandle changed: ", e.Name)
		loadConfig()
	})
	viper.WatchConfig()
}

func GenYamlConfig(path string, force bool) error {
	if !fsx.FileExist(path) || force {
		data, _ := yaml.Marshal(&GlobalConfig{MODE: "debug"})
		err := os.WriteFile(path, data, 0644)
		if err != nil {
			return errors.New("Generate file with error: " + err.Error())
		}
		fmt.Println("Config file `config.yaml` generate success in " + path)
	} else {
		return errors.New(path + " already exist, use -f to Force coverage")
	}
	return nil
}

func Get() *GlobalConfig {
	return serveConfig
}
