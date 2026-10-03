package example

import (
	"context"
	"sync"

	"github.com/juanjiTech/jframe/core/kernel"
	"github.com/juanjiTech/jframe/mod/example/dao"
	"github.com/juanjiTech/jframe/mod/example/handler"
	"github.com/juanjiTech/jframe/mod/example/service"
	"github.com/juanjiTech/jin"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

var _ kernel.Module = (*Mod)(nil)

// Mod 是 create 脚手架的金标准模板：演示 Load 中 dao → service → handler 组装，
// 以及用 github.com/juanjiTech/jframe/pkg/stdao 做数据访问。
// 默认不注册进 modList；需要联调时再挂上。
type Mod struct {
	kernel.UnimplementedModule
}

func (m *Mod) Name() string { return "example" }

func (m *Mod) Load(h *kernel.Hub) error {
	var j *jin.Engine
	if err := h.Load(&j); err != nil {
		return errors.Wrap(err, "load jin.Engine")
	}
	var db *gorm.DB
	if err := h.Load(&db); err != nil {
		return errors.Wrap(err, "load gorm.DB")
	}

	itemDao, err := dao.NewItemDao(db)
	if err != nil {
		return errors.Wrap(err, "init item dao")
	}
	itemSvc := service.NewItemService(itemDao)
	itemHandler := handler.NewItemHandler(itemSvc)

	g := j.Group("/api/example")
	itemHandler.RegisterRoutes(g)
	h.Log.Infow("routes registered", "group", "/api/example")
	return nil
}

func (m *Mod) Stop(wg *sync.WaitGroup, _ context.Context) error {
	defer wg.Done()
	return nil
}
