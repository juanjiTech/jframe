---
name: jframe-module-dev
description: >-
  Implement code inside an existing jframe module (handler/service/dao/model),
  wire Load(), register jin routes, use pkg/stdao, fix DI Load/Map issues, or
  write module tests. Trigger when editing mod/<name>/, implementing CRUD,
  AutoMigrate/DAO, binding.JSON, render.JSON, hub.Load errors, or stdao.
  Do NOT use for greenfield module design/scaffolding (jframe-module-design)
  or kernel/core framework changes.
---

# jframe Module Development

Implement layers inside an **existing** module. For new module design/scaffold, use **jframe-module-design**.

## Progressive reading

| Need | Read |
|------|------|
| Default path | This file + **`mod/example/`** |
| DAO / transactions | [`references/stdao.md`](references/stdao.md) |
| jin / binding / params | [`references/jin-handlers.md`](references/jin-handlers.md) |
| Load wiring / errors / checklist | [`references/wiring.md`](references/wiring.md) |
| gRPC, settings, testing, DI ops | [`references/advanced.md`](references/advanced.md) |
| Human docs | `docs/module-implementation.md`, `docs/stdao.md` |

**Do not** load every reference up front — open only what the current task needs.

## Gold standard

Copy from **`mod/example/`** (Item CRUD, compilable, not in `modList` by default):

```
mod/example/
├── mod.go           # Load: db + jin → dao → service → handler → routes
├── model/item.go
├── dao/item.go      # stdao.Std[*model.Item]
├── service/item.go
├── handler/item.go
└── e/e.go
```

Data flow: `handler → service → dao → model`. Assemble bottom-up in `Load()`.

## Implementation order

1. **model** — GORM struct (prefer `stdao.Model`) + DTOs. See example `model/item.go`.
2. **dao** — `stdao.Std[*model.T]` + `NewXxxDao` + `Init`. Details: `references/stdao.md`.
3. **service** — business rules; return `e/` errors.
4. **handler** — jin routes. Details: `references/jin-handlers.md`.
5. **mod.go Load** — `hub.Load` deps, wire layers, register routes. Details: `references/wiring.md`.

## Minimal dao sketch

```go
type ItemDao struct {
    stdao.Std[*model.Item]
}
func NewItemDao(db *gorm.DB) (*ItemDao, error) {
    d := &ItemDao{}
    return d, d.Init(db)
}
// queries: d.GetTxFromCtx(ctx).WithContext(ctx).Where(...).First(...)
```

Import path: `github.com/juanjiTech/jframe/pkg/stdao`.

## Minimal Load sketch

```go
itemDao, err := dao.NewItemDao(db)
svc := service.NewItemService(itemDao)
h := handler.NewItemHandler(svc)
h.RegisterRoutes(j.Group("/api/<name>"))
```

## Pitfalls (short)

1. `Stop`: `defer wg.Done()`.
2. `hub.Map` / `hub.Load`: always pointers; check Load errors.
3. Load deps in **Load/Init+**, not PreInit (peers may not have Mapped yet).
4. jin: `c.Params.ByName`, `render.JSON` — not gin's `c.Param` / `c.JSON`.
5. Config fields need both `yaml` and `mapstructure` tags.

## Done when

- [ ] Matches `mod/example` layering
- [ ] DAO via `pkg/stdao` + `GetTxFromCtx` for custom SQL
- [ ] `go build ./...` passes
