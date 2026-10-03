# jin handlers (Agent)

jin is a gin fork: **no gin binding**; use DI + `jin/middleware/binding`.  
Gold standard: `mod/example/handler/item.go`.

## Differences from gin

- `HandlerFunc` is `interface{}` — any signature; args via `inject.Invoke`.
- No `c.JSON` — use `c.Render(code, render.JSON{Data: data})`.
- Path params: `c.Params.ByName("id")` / `c.Params.Get("id")` — **not** `c.Param`.
- Query binding tag is `query:"key"` (not `form`).

## Routes

```go
import "github.com/juanjiTech/jin/middleware/binding"

g := j.Group("/api/<name>")
g.GET("/items", h.list)
g.GET("/items/:id", h.get)
g.POST("/items", binding.JSON(model.CreateItemReq{}), h.create)
```

```go
func (h *ItemHandler) create(req model.CreateItemReq, c *jin.Context) {
    item, err := h.svc.Create(context.Background(), req.Name)
    // map domain errors → status codes
    c.Render(http.StatusCreated, render.JSON{Data: resp})
}
```

## Query binding

```go
type ListQuery struct {
    Page     int    `query:"page"`
    PageSize int    `query:"page_size"`
    Tags     []string `query:"tags"` // ?tags=a,b
}
g.GET("/items", binding.Query(ListQuery{}), h.list)
```

`binding.JSON` / `binding.Query` Map parsed values into request-scoped DI; handler params are injected automatically.
