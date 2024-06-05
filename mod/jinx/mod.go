package jinx

import (
	"context"
	"errors"
	"fmt"
	"github.com/juanjiTech/jframe/conf"
	"github.com/juanjiTech/jframe/core/kernel"
	"github.com/juanjiTech/jin"
	"github.com/juanjiTech/jin/middleware/cors"
	sentryjin "github.com/juanjiTech/sentry-jin"
	"github.com/soheilhy/cmux"
	"net"
	"net/http"
	"sync"
	"time"
)

var _ kernel.Module = (*Mod)(nil)

type Mod struct {
	kernel.UnimplementedModule // 请为所有Module引入UnimplementedModule

	listener net.Listener
	j        *jin.Engine
	httpSrv  *http.Server
}

func (m *Mod) Name() string {
	return "jinx"
}

func (m *Mod) Init(hub *kernel.Hub) error {
	m.j = jin.New()
	corsConf := cors.DefaultConfig()
	corsConf.AllowAllOrigins = true
	corsConf.AllowCredentials = true
	corsConf.AddAllowHeaders("Authorization")
	m.j.Use(
		jin.Recovery(),
		cors.New(corsConf),
	)
	if conf.Get().SentryDsn != "" {
		m.j.Use(sentryjin.New(sentryjin.Options{Repanic: true}))
	}

	hub.Map(m.j)
	return nil
}

func (m *Mod) Load(hub *kernel.Hub) error {
	var jinE jin.Engine
	err := hub.Load(&jinE)
	if err != nil {
		return errors.New("can't load jin.Engine from kernel")
	}
	return nil
}

func (m *Mod) Start(hub *kernel.Hub) error {
	var tcpMux cmux.CMux
	err := hub.Load(&tcpMux)
	if err != nil {
		return errors.New("can't load tcpMux from kernel")
	}

	httpL := tcpMux.Match(cmux.HTTP1Fast())
	m.listener = httpL
	m.httpSrv = &http.Server{
		Handler: m.j,
	}

	if err := m.httpSrv.Serve(httpL); err != nil && !errors.Is(err, http.ErrServerClosed) {
		hub.Log.Infow("failed to start to listen and serve", "error", err)
	}
	return nil
}

func (m *Mod) Stop(wg *sync.WaitGroup, ctx context.Context) error {
	defer wg.Done()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := m.httpSrv.Shutdown(ctx); err != nil {
		fmt.Println("Server forced to shutdown: " + err.Error())
		return err
	}
	return nil
}
