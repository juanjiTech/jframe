# Load wiring, errors, checklist (Agent)

## Load assembly (bottom-up)

```go
func (m *Mod) Load(hub *kernel.Hub) error {
    var j *jin.Engine
    if err := hub.Load(&j); err != nil {
        return errors.Wrap(err, "load jin.Engine")
    }
    var db *gorm.DB
    if err := hub.Load(&db); err != nil {
        return errors.Wrap(err, "load gorm.DB")
    }

    itemDao, err := dao.NewItemDao(db)
    if err != nil {
        return err
    }
    svc := service.NewItemService(itemDao)
    h := handler.NewItemHandler(svc)
    h.RegisterRoutes(j.Group("/api/<name>"))
    return nil
}
```

Shared types: `docs/di-reference.md`. Always check `hub.Load` errors. Use `hub.Log`, not `fmt.Println`.

## Domain errors (`e/`)

```go
var ErrItemNotFound = errors.New("item not found")
```

Service returns these; handler maps with `errors.Is` to HTTP status.

## Worker modules (no HTTP)

`Load`: construct dao/service from Hub.  
`Start`: block on consume loop (kernel already runs Start in a goroutine).  
`Stop`: `defer wg.Done()`; cancel context. Skip empty handler package if unused.

## Checklist

- [ ] Models: GORM + JSON tags; DAO uses `stdao.Std[*model.T]` + `Init`
- [ ] Custom DAO queries use `GetTxFromCtx(ctx)`
- [ ] Handlers: `binding.JSON`/`Query`, `render.JSON`, `c.Params.ByName`
- [ ] Routes registered in `Load()`
- [ ] `go build ./...` / `go vet ./...` pass
- [ ] Module not inventing its own DB/Redis connections
