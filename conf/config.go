package conf

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var SysVersion = "dev"

var serveConfig *GlobalConfig

func LoadConfig(configPath ...string) error {
	noCustomConfigPath := len(configPath) == 0 || (len(configPath) >= 1 && configPath[0] == "")
	if noCustomConfigPath {
		viper.SetConfigName("config")
		viper.AddConfigPath(".")
		viper.AddConfigPath("./config")
	} else {
		viper.SetConfigFile(configPath[0])
	}

	serveConfig = new(GlobalConfig)
	
	loadConfig := func() error {
		newConf := new(GlobalConfig)
		err := viper.ReadInConfig()
		if err != nil {
			if noCustomConfigPath && errors.As(err, &viper.ConfigFileNotFoundError{}) {
				// 没指定配置文件路径，且不是配置文件未找到错误
				return nil
			}
			return errors.Wrap(err, "config read failed")
		}
		err = viper.Unmarshal(newConf)
		if err != nil {
			return errors.Wrap(err, "config unmarshal failed")
		}
		serveConfig = newConf
		return nil
	}

	err := loadConfig()
	if err != nil {
		return err
	}

	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config fileHandle changed: ", e.Name)
		err = loadConfig()
		if err != nil {
			zap.S().Error(err)
		}
	})
	viper.WatchConfig()
	return nil
}

func Get() *GlobalConfig {
	return serveConfig
}
