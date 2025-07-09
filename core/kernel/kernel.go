package kernel

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/juanjiTech/inject/v2"
	"github.com/juanjiTech/jframe/core/logx"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Engine struct {
	config Config

	Ctx    context.Context
	Cancel context.CancelFunc

	inject.Injector
	modules   map[string]Module
	modulesMu sync.Mutex
}

type Config struct {
	EnableSentry bool
}

func New(config ...Config) *Engine {
	if len(config) == 0 {
		panic("config can't be empty")
	}
	return &Engine{
		config:   config[0],
		Injector: inject.New(),
		modules:  make(map[string]Module),
	}
}

func (e *Engine) Init() {
	e.Ctx, e.Cancel = context.WithCancel(context.Background())
}

func (e *Engine) StartModule() error {
	hub := Hub{
		Injector: e.Injector,
	}
	for _, module := range e.modules {
		c := module.Config()
		if c == nil {
			continue
		}
		zap.S().Info("Module " + module.Name() + " has config, try to load it")
		ct := reflect.TypeOf(c)
		if ct.Kind() != reflect.Pointer {
			zap.S().Errorf("The config exported by module %s is not a pointer.", module.Name())
		}

		// The Viper can't unmarshal config from env directly into a map, so we need to
		// create a dynamic struct that has a single field with the mapstructure tag
		// corresponding to the module's name. This allows Viper to unmarshal the
		// configuration for a specific module based on its name.
		// The created struct is equivalent to:
		//
		// type DynamicConfig struct {
		//     Config *ModuleConfigType `mapstructure:"module_name"`
		// }
		structType := reflect.StructOf([]reflect.StructField{
			{
				Name: "Config",
				Type: reflect.TypeOf(c),
				Tag:  reflect.StructTag(fmt.Sprintf(`mapstructure:"%s"`, module.Name())),
			},
		})

		// Create a new pointer to an instance of the dynamic struct.
		instance := reflect.New(structType)
		// Set the 'Config' field of our dynamic struct to point to the module's config object.
		instance.Elem().Field(0).Set(reflect.ValueOf(c))

		// Unmarshal the configuration into our dynamically created struct instance.
		if err := viper.Unmarshal(instance.Interface()); err != nil {
			zap.S().Error("Config Unmarshal failed: " + err.Error())
		}
		fmt.Println(module.Config())
	}
	for _, m := range e.modules {
		h4m := hub
		h4m.Log = logx.NameSpace("module." + m.Name())
		if err := m.PreInit(&h4m); err != nil {
			h4m.Log.Error(err)
			panic(err)
		}
	}
	for _, m := range e.modules {
		h4m := hub
		h4m.Log = logx.NameSpace("module." + m.Name())
		if err := m.Init(&h4m); err != nil {
			h4m.Log.Error(err)
			panic(err)
		}
	}
	for _, m := range e.modules {
		h4m := hub
		h4m.Log = logx.NameSpace("module." + m.Name())
		if err := m.PostInit(&h4m); err != nil {
			h4m.Log.Error(err)
			panic(err)
		}
	}
	for _, m := range e.modules {
		h4m := hub
		h4m.Log = logx.NameSpace("module." + m.Name())
		if err := m.Load(&h4m); err != nil {
			h4m.Log.Error(err)
			panic(err)
		}
	}
	for _, m := range e.modules {
		go func(m Module) {
			h4m := hub
			h4m.Log = logx.NameSpace("module." + m.Name())
			if err := m.Start(&h4m); err != nil {
				h4m.Log.Error(err)
				panic(err)
			}
		}(m)
	}
	return nil
}

func (e *Engine) Serve() {
}

func (e *Engine) Stop() error {
	wg := sync.WaitGroup{}
	wg.Add(len(e.modules))
	for _, m := range e.modules {
		err := m.Stop(&wg, e.Ctx)
		if err != nil {
			return err
		}
	}
	wg.Wait()

	return nil
}
