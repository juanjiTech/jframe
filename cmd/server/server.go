package server

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/juanjiTech/jframe/cmd/server/modList"
	"github.com/juanjiTech/jframe/conf"
	"github.com/juanjiTech/jframe/core/kernel"
	"github.com/juanjiTech/jframe/core/logx"
	"github.com/juanjiTech/jframe/pkg/ip"
	"github.com/juanjiTech/jframe/pkg/sentry"
	"github.com/soheilhy/cmux"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	configPath string
	StartCmd   = &cobra.Command{
		Use:     "server",
		Short:   "Start server",
		Example: "jframe server -c ./config.yaml",
		Run: func(cmd *cobra.Command, args []string) {
			//logx.PreInit()
			log := logx.NameSpace("cmd.server")
			log.Info("loading config...")
			// Enable BindStruct to allow unmarshal env into a nested struct
			// https://github.com/spf13/viper/pull/1429
			viper.SetOptions(viper.ExperimentalBindStruct())
			// This line allows viper to use an env var like ORIGIN_VALUE to override the viper string "Origin.Value"
			viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
			viper.AutomaticEnv()
			conf.LoadConfig(configPath)
			log.Info("loading config complete")

			log.Info("init dep...")
			if conf.Get().SentryDsn != "" {
				sentry.Init()
			}
			//if conf.Get().MODE == "" || conf.Get().MODE == "debug" {
			//	logx.Init(zapcore.DebugLevel)
			//} else {
			//	logx.Init(zapcore.InfoLevel)
			//}
			defer func() {
				if err := recover(); err != nil {
					log.Errorw("panic", "error", err)
				}
			}()
			log.Info("init dep complete")

			log.Info("init kernel...")
			conn, err := net.Listen("tcp", fmt.Sprintf(":%s", conf.Get().Port))
			if err != nil {
				log.Errorw("failed to listen", "error", err)
				return
			}
			tcpMux := cmux.New(conn)
			log.Infow("start listening", "port", conf.Get().Port)
			k := kernel.New(kernel.Config{})
			k.Map(&conn, &tcpMux)
			// ModList is a list of module that you want to start
			// the place to add your module is in modList.go
			k.RegMod(modList.ModList...)
			k.Init()
			log.Info("init kernel complete")

			log.Info("init module...")
			err = k.StartModule()
			if err != nil {
				panic(err)
			}
			log.Info("init module complete")

			log.Info("starting Server...")
			k.Serve()
			go func() {
				_ = tcpMux.Serve()
			}()

			fmt.Println("Server run at:")
			fmt.Printf("-  Local:   http://localhost:%s\n", conf.Get().Port)
			for _, host := range ip.GetLocalHost() {
				fmt.Printf("-  Network: http://%s:%s\n", host, conf.Get().Port)
			}

			quit := make(chan os.Signal, 1)
			signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
			<-quit
			fmt.Println("Shutting down server...")

			err = k.Stop()
			if err != nil {
				panic(err)
			}
		},
	}
)

func init() {
	StartCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Start server with provided configuration file")
}
