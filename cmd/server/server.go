package server

import (
	"fmt"
	"github.com/juanjiTech/jframe/conf"
	"github.com/juanjiTech/jframe/core/kernel"
	"github.com/juanjiTech/jframe/core/logx"
	"github.com/juanjiTech/jframe/mod/grpcGateway"
	"github.com/juanjiTech/jframe/mod/jinPprof"
	"github.com/juanjiTech/jframe/mod/jinx"
	"github.com/juanjiTech/jframe/mod/myDB"
	"github.com/juanjiTech/jframe/mod/rds"
	"github.com/juanjiTech/jframe/mod/uptrace"
	"github.com/juanjiTech/jframe/pkg/ip"
	"github.com/juanjiTech/jframe/pkg/sentry"
	"github.com/soheilhy/cmux"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
	"net"
	"os"
	"os/signal"
	"syscall"
)

var log = logx.NameSpace("cmd.server")

var (
	configYml string
	StartCmd  = &cobra.Command{
		Use:     "server",
		Short:   "Start server",
		Example: "jframe server -c ./config.yaml",
		Run: func(cmd *cobra.Command, args []string) {
			log.Info("loading config...")
			conf.LoadConfig(configYml)
			log.Info("loading config complete")

			log.Info("init dep...")
			if conf.Get().SentryDsn != "" {
				sentry.Init()
			}
			if conf.Get().MODE == "" || conf.Get().MODE == "debug" {
				logx.Init(zapcore.DebugLevel)
			} else {
				logx.Init(zapcore.InfoLevel)
			}
			defer func() {
				if err := recover(); err != nil {
					_ = log.Sync()
				}
			}()
			log.Info("init dep complete")

			log.Info("init kernel...")
			conn, err := net.Listen("tcp", fmt.Sprintf(":%s", conf.Get().Port))
			if err != nil {
				log.Fatalw("failed to listen", "error", err)
			}
			tcpMux := cmux.New(conn)
			log.Infow("start listening", "port", conf.Get().Port)
			k := kernel.New(kernel.Config{})
			k.Map(&conn, &tcpMux)
			k.RegMod(
				//&b2x.Mod{},
				&uptrace.Mod{},
				&grpcGateway.Mod{},
				&jinPprof.Mod{},
				&jinx.Mod{},
				&myDB.Mod{},
				&rds.Mod{},
			)
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
	StartCmd.PersistentFlags().StringVarP(&configYml, "config", "c", "", "Start server with provided configuration file")
}
