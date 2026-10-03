# Complete module template (optional deep dive)

Prefer scaffolding from `mod/example/` via `go run . create -n <name>`, then rename types. Use this only when writing `mod.go` by hand for a non-CRUD module.

```go
package myMod

import (
    "context"
    "sync"

    "github.com/juanjiTech/jframe/core/kernel"
    "github.com/juanjiTech/jin"
    "github.com/juanjiTech/jin/middleware/binding"
    "github.com/juanjiTech/jin/render"
    "github.com/pkg/errors"
    "gorm.io/gorm"
    "net/http"
)

var _ kernel.Module = (*Mod)(nil)

type Mod struct {
    kernel.UnimplementedModule
    config Config
}

type Config struct {
    SomeParam string `yaml:"someParam" mapstructure:"someParam"`
}

func (m *Mod) Name() string { return "myMod" }
func (m *Mod) Config() any  { return &m.config }

func (m *Mod) PreInit(hub *kernel.Hub) error {
    // hub.Map(&client)
    return nil
}

func (m *Mod) Load(hub *kernel.Hub) error {
    var j *jin.Engine
    if err := hub.Load(&j); err != nil {
        return errors.Wrap(err, "load jin.Engine")
    }
    var db *gorm.DB
    if err := hub.Load(&db); err != nil {
        return errors.Wrap(err, "load gorm.DB")
    }
    // Prefer: dao.NewXxxDao(db) → service → handler.RegisterRoutes
    // See mod/example/mod.go
    g := j.Group("/api/myMod")
    g.GET("/list", func(c *jin.Context) {
        c.Render(http.StatusOK, render.JSON{Data: "ok"})
    })
    g.POST("/create", binding.JSON(createReq{}), func(req createReq, c *jin.Context) {
        c.Render(http.StatusOK, render.JSON{Data: req.Name})
    })
    return nil
}

func (m *Mod) Stop(wg *sync.WaitGroup, _ context.Context) error {
    defer wg.Done()
    return nil
}

type createReq struct {
    Name string `json:"name"`
}
```

For layer implementation after scaffold, switch to **jframe-module-dev**.
