package pyroscope

import (
	"context"
	"os"
	"sync"

	"github.com/grafana/pyroscope-go"
	"github.com/juanjiTech/jframe/core/kernel"
)

type Config struct {
	ApplicationName string `yaml:"applicationName"`
	ServerAddress   string `yaml:"serverAddress"`
	BasicAuthUser   string `yaml:"basicAuthUser"`
	BasicAuthPass   string `yaml:"basicAuthPass"`
	TenantID        string `yaml:"tenantID"`
}

var _ kernel.Module = (*Mod)(nil)

type Mod struct {
	kernel.UnimplementedModule // 请为所有Module引入UnimplementedModule

	config Config

	profiler *pyroscope.Profiler
}

func (m *Mod) Name() string {
	return "pyroscope"
}

func (m *Mod) Config() any {
	return &m.config
}

func (m *Mod) PreInit(hub *kernel.Hub) error {
	pyroscopeConf := m.config
	if pyroscopeConf.ServerAddress == "" {
		hub.Log.Info("pyroscope server address is empty, skip init pyroscope")
		return nil
	}
	var err error
	m.profiler, err = pyroscope.Start(pyroscope.Config{
		ApplicationName: pyroscopeConf.ApplicationName,

		Tags: map[string]string{
			"hostname": os.Getenv("HOSTNAME"),
		},

		// replace this with the address of pyroscope server
		ServerAddress: pyroscopeConf.ServerAddress,

		// you can disable logging by setting this to nil
		Logger: nil,

		// Optional HTTP Basic authentication (Grafana Cloud)
		BasicAuthUser:     pyroscopeConf.BasicAuthUser,
		BasicAuthPassword: pyroscopeConf.BasicAuthPass,
		// Optional Pyroscope tenant ID (only needed if using multi-tenancy). Not needed for Grafana Cloud.
		TenantID: pyroscopeConf.TenantID,

		// by default all profilers are enabled,
		// but you can select the ones you want to use:
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,
		},
	})
	if err != nil {
		hub.Log.Error(err)
		return err
	}
	return nil
}

func (m *Mod) Stop(wg *sync.WaitGroup, _ context.Context) error {
	defer wg.Done()
	if m.profiler != nil {
		return m.profiler.Stop()
	}
	return nil
}
