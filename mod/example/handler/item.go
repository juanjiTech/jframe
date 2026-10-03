package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/juanjiTech/jframe/mod/example/e"
	"github.com/juanjiTech/jframe/mod/example/model"
	"github.com/juanjiTech/jframe/mod/example/service"
	"github.com/juanjiTech/jin"
	"github.com/juanjiTech/jin/middleware/binding"
	"github.com/juanjiTech/jin/render"
)

type ItemHandler struct {
	svc *service.ItemService
}

func NewItemHandler(svc *service.ItemService) *ItemHandler {
	return &ItemHandler{svc: svc}
}

func (h *ItemHandler) RegisterRoutes(g jin.IRoutes) {
	g.GET("/items", h.list)
	g.GET("/items/:id", h.get)
	g.POST("/items", binding.JSON(model.CreateItemReq{}), h.create)
}

func (h *ItemHandler) list(c *jin.Context) {
	items, err := h.svc.List(context.Background())
	if err != nil {
		c.Render(http.StatusInternalServerError, render.JSON{Data: map[string]string{"error": err.Error()}})
		return
	}
	out := make([]model.ItemResp, 0, len(items))
	for _, it := range items {
		out = append(out, model.ItemResp{ID: it.ID, Name: it.Name, Status: it.Status})
	}
	c.Render(http.StatusOK, render.JSON{Data: out})
}

func (h *ItemHandler) get(c *jin.Context) {
	item, err := h.svc.Get(context.Background(), c.Params.ByName("id"))
	if err != nil {
		if errors.Is(err, e.ErrItemNotFound) {
			c.Render(http.StatusNotFound, render.JSON{Data: map[string]string{"error": err.Error()}})
			return
		}
		c.Render(http.StatusInternalServerError, render.JSON{Data: map[string]string{"error": err.Error()}})
		return
	}
	c.Render(http.StatusOK, render.JSON{Data: model.ItemResp{ID: item.ID, Name: item.Name, Status: item.Status}})
}

func (h *ItemHandler) create(req model.CreateItemReq, c *jin.Context) {
	item, err := h.svc.Create(context.Background(), req.Name)
	if err != nil {
		if errors.Is(err, e.ErrInvalidName) {
			c.Render(http.StatusBadRequest, render.JSON{Data: map[string]string{"error": err.Error()}})
			return
		}
		c.Render(http.StatusInternalServerError, render.JSON{Data: map[string]string{"error": err.Error()}})
		return
	}
	c.Render(http.StatusCreated, render.JSON{Data: model.ItemResp{ID: item.ID, Name: item.Name, Status: item.Status}})
}
