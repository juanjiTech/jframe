---
name: jframe-module-dev
description: "Implement, wire, and test code inside an existing jframe module. Invoke this skill when the user wants to: write handler/service/dao/model layer code, implement CRUD or business logic, register HTTP routes with jin (binding.JSON, binding.Query, render.JSON, DI-injected handler functions), register gRPC endpoints via gateway, use stdao generic DAO for GORM queries, use the settings system for dynamic config, wire module layers in mod.go Load(), write tests for handlers or services, or troubleshoot DI wiring issues (hub.Map/Load/Invoke). Trigger on phrases like 'implement X in module', 'write handler for', 'add API endpoint', 'use stdao', 'binding.JSON', 'DI inject', or any request to write/modify code within an existing mod/ directory. Do NOT use this skill for designing new modules from scratch or scaffolding (use jframe-module-design instead), or for framework-level changes to kernel/core."
---

# jframe Module Development

This skill guides you through implementing the internal code of a jframe module — the handler, service, dao, and model layers, plus DI wiring, route registration, and testing.

**Prerequisite:** The module skeleton should already exist (created via `jframe create -n <name>` or manually). If not, use the **jframe-module-design** skill first — it walks through requirements gathering, lifecycle phase selection, config struct design, DI planning, scaffolding, and module registration.

## Module internal architecture

jframe modules follow a layered architecture convention:

```
mod/<name>/
├── mod.go       # Lifecycle entry point — DI wiring, route registration
├── handler/     # HTTP/gRPC request handling — parse input, call service, format response
├── service/     # Business logic — orchestrates dao calls, enforces rules
├── dao/         # Data access — GORM queries, wraps stdao
├── model/       # Data models — GORM structs, DTOs
└── e/           # Error codes — domain-specific error definitions
```

**Data flow:** `handler → service → dao → model`

The handler receives requests, calls service methods with domain types, the service implements business rules and calls dao for persistence, and the dao talks to the database through GORM. This separation keeps each layer testable and replaceable.

## Implementing the model layer

Models define your data structures. For database-backed models, embed `stdao.Model` which provides ULID primary key, timestamps, and soft delete:

```go
// mod/<name>/model/user.go
package model

import "github.com/juanjiTech/jframe/pkg/stdao"

type User struct {
    stdao.Model
    Username string `json:"username" gorm:"uniqueIndex;size:64"`
    Email    string `json:"email" gorm:"size:256"`
    Status   int    `json:"status" gorm:"default:1"`
}
```

`stdao.Model` gives you:

```go
type Model struct {
    ID        string          `json:"id" gorm:"primaryKey;size:26"`  // ULID
    CreatedAt time.Time       `json:"createdAt"`
    UpdatedAt time.Time       `json:"updatedAt"`
    DeletedAt *gorm.DeletedAt `json:"deletedAt" gorm:"index"`       // soft delete
}
```

For request/response DTOs, define plain structs (no GORM tags):

```go
type CreateUserReq struct {
    Username string `json:"username"`
    Email    string `json:"email"`
}

type UserResp struct {
    ID       string `json:"id"`
    Username string `json:"username"`
    Email    string `json:"email"`
}
```

## Implementing the DAO layer

Use `stdao.Std[T]` as a generic base. It provides Create, List, Update, Delete, and context-based transaction propagation out of the box. Note that `Delete()` returns `*gorm.DB` (not `error`), so check `.Error` on the result:

```go
// mod/<name>/dao/user.go
package dao

import (
    "context"
    "github.com/juanjiTech/jframe/pkg/stdao"
    "github.com/juanjiTech/jframe/mod/<name>/model"
    "gorm.io/gorm"
)

type UserDao struct {
    stdao.Std[*model.User]  // Note: pointer type for GORM compatibility
}

func NewUserDao(db *gorm.DB) (*UserDao, error) {
    d := &UserDao{}
    if err := d.Init(db); err != nil {  // Init calls AutoMigrate
        return nil, err
    }
    return d, nil
}

// Custom queries beyond CRUD
func (d *UserDao) GetByUsername(ctx context.Context, username string) (*model.User, error) {
    var user model.User
    err := d.GetTxFromCtx(ctx).WithContext(ctx).Where("username = ?", username).First(&user).Error
    return &user, err
}
```

**Transaction propagation:** stdao uses context to carry transactions. Any dao method that calls `GetTxFromCtx(ctx)` will automatically participate in an active transaction if one exists in the context:

```go
// Start a transaction in the service layer
tx := d.Begin()
ctx = stdao.SetTxToCtx(ctx, tx)
// All dao calls with this ctx now use the same transaction
err1 := userDao.Create(ctx, user)
err2 := profileDao.Create(ctx, profile)
if err1 != nil || err2 != nil {
    tx.Rollback()
} else {
    tx.Commit()
}
```

## Implementing the service layer

Services contain business logic. They receive dao instances (injected) and operate on domain types:

```go
// mod/<name>/service/user.go
package service

import (
    "context"
    "github.com/juanjiTech/jframe/mod/<name>/dao"
    "github.com/juanjiTech/jframe/mod/<name>/model"
    "github.com/pkg/errors"
)

type UserService struct {
    userDao *dao.UserDao
}

func NewUserService(userDao *dao.UserDao) *UserService {
    return &UserService{userDao: userDao}
}

func (s *UserService) CreateUser(ctx context.Context, username, email string) (*model.User, error) {
    existing, err := s.userDao.GetByUsername(ctx, username)
    if err == nil && existing.ID != "" {
        return nil, errors.New("username already exists")
    }

    user := &model.User{
        Username: username,
        Email:    email,
    }
    if err := s.userDao.Create(ctx, user); err != nil {
        return nil, errors.Wrap(err, "failed to create user")
    }
    return user, nil
}
```

## Implementing the handler layer

jframe uses `jin` (a gin fork that **removes the binding package** and replaces it with dependency injection). This is the core design philosophy:

**Handler = pure function. Input parameters are auto-injected request data. Return values are auto-registered as response data.**

jin's key differences from gin:
- **`HandlerFunc` is `interface{}`** — any function signature works, not just `func(*Context)`
- **`Context.Next()` calls handlers via `inject.Invoke()`** — parameters are auto-resolved from DI
- **Handler return values are auto-registered by return type** — `c.Set(fnType.Out(i), val)` (context.go:82) stores each return value keyed by its declared return type
- **`jin.Context` embeds `inject.Injector`** — each request has its own DI scope

This means you can build a middleware pipeline where:
1. **Binding middleware** parses the request → `c.Map(parsedStruct)` into request-scoped DI
2. **Handler** is a pure function that receives the parsed struct as a parameter and returns a response struct
3. **Render middleware** reads the returned response struct from DI and writes HTTP response

### The pure function handler pattern

```go
// Handler is a pure function — no HTTP plumbing, just business logic
func (h *UserHandler) createUser(req model.CreateUserReq) (*model.UserResp, error) {
    user, err := h.svc.CreateUser(context.Background(), req.Username, req.Email)
    if err != nil {
        return nil, err
    }
    return &model.UserResp{ID: user.ID, Username: user.Username}, nil
}

// jin's Invoke mechanism:
// 1. Resolves `model.CreateUserReq` from DI (put there by binding middleware)
// 2. Calls the function
// 3. Auto-Maps return values back into DI (for render middleware to consume)
```

### Binding middleware — parse request into DI

jin provides official binding middleware at `jin/middleware/binding`. The binding package is the bridge between raw HTTP requests and jin's DI system — it transforms unstructured request data into typed Go structs and registers them into the request-scoped DI container via `ctx.Map()`.

**`binding.JSON[T](T{})`** — JSON body binding:
- Parses `r.Body` using `json.Decoder` with `UseNumber()` for precision
- Uses `io.TeeReader` to preserve the body — downstream handlers can still read `r.Body`
- On success: `ctx.Map(t)` registers the parsed struct into DI
- On decode error (except EOF): `ctx.Error(err)` + `ctx.Abort()`
- On empty body (EOF): silently continues without mapping (handler can check)

```go
import "github.com/juanjiTech/jin/middleware/binding"

// Usage: middleware in route chain
engine.POST("/users", binding.JSON(model.CreateUserReq{}), handler.createUser)
// handler receives CreateUserReq as a function parameter — auto-injected
```

**`binding.Query[T](T{})`** — URL query params binding:
- Uses `query:"key"` struct tag (not `form:"key"` like gin)
- Supports: string, int/int8-64, uint/uint8-64, float32/64, bool, slices, nested structs
- Nested structs use dot notation: `?parent.child=value` (format configurable via `binding.DefaultQueryFormat`)
- Slices use comma separation: `?ids=1,2,3`
- Bool accepts: `true`, `1`, `True` → true; everything else → false
- Anonymous fields are skipped, fields without `query` tag are skipped

```go
type ListQuery struct {
    Page     int      `query:"page"`
    PageSize int      `query:"page_size"`
    Search   string   `query:"search"`
    Tags     []string `query:"tags"`        // ?tags=go,web,api
    Filter   struct {
        Status int    `query:"status"`
        Sort   string `query:"sort"`
    } `query:"filter"`                       // ?filter.status=1&filter.sort=name
}

engine.GET("/users", binding.Query(ListQuery{}), handler.listUsers)
// handler receives ListQuery as parameter
```

You can also write custom binding middleware for other sources (headers, path params, etc.):

```go
// Custom binding middleware pattern
func BindPathParam(key string) jin.HandlerFunc {
    return func(c *jin.Context) {
        val, _ := c.Params.Get(key)
        c.Map(val) // Maps as string type
    }
}
```

### Route registration with middleware chain

```go
import "github.com/juanjiTech/jin/middleware/binding"

func (h *UserHandler) RegisterRoutes(g *jin.RouterGroup) {
    // binding middleware → pure handler (receives parsed request via DI)
    g.POST("/users", binding.JSON(model.CreateUserReq{}), h.createUser)
    g.GET("/users", binding.Query(model.ListUserQuery{}), h.listUsers)
    g.GET("/users/:id", h.getUser)
}
```

### Handler function signatures

jin supports multiple handler signatures via `inject.Invoke`:

```go
// Classic gin-style (fast path, no reflection)
func(c *jin.Context) { ... }

// Standard http handler (fast path)
func(w http.ResponseWriter, r *http.Request) { ... }

// Pure function with DI injection — any combo of mapped types
func(req model.CreateUserReq) (*model.UserResp, error) { ... }
func(req model.CreateUserReq, c *jin.Context) { ... }
func(db *gorm.DB, c *jin.Context) { ... }
```

### Accessing request data directly

For handlers that don't use the binding middleware pattern:

```go
// Path params
id, _ := c.Params.Get("id")

// Query params
page := c.Request.URL.Query().Get("page")

// Headers
auth := c.Request.Header.Get("Authorization")

// Manual JSON body
var req model.CreateUserReq
json.NewDecoder(c.Request.Body).Decode(&req)
```

### Response rendering

jin provides a `render` package (no `c.JSON` shorthand — use `c.Render`):

```go
import "github.com/juanjiTech/jin/render"

c.Render(http.StatusOK, render.JSON{Data: user})
c.Render(http.StatusOK, render.IndentedJSON{Data: user})

// Or write directly
c.Writer.Header().Set("Content-Type", "application/json")
json.NewEncoder(c.Writer).Encode(user)
```

### Complete handler example

```go
// mod/<name>/handler/user.go
package handler

import (
    "net/http"
    "github.com/juanjiTech/jin"
    "github.com/juanjiTech/jin/middleware/binding"
    "github.com/juanjiTech/jin/render"
    "github.com/juanjiTech/jframe/mod/<name>/model"
    "github.com/juanjiTech/jframe/mod/<name>/service"
    "github.com/juanjiTech/jframe/mod/<name>/e"
    "errors"
)

type UserHandler struct {
    svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
    return &UserHandler{svc: svc}
}

func (h *UserHandler) RegisterRoutes(g *jin.RouterGroup) {
    // binding.JSON parses request body → Maps model.CreateUserReq into DI
    // handler's `req` parameter is auto-injected
    g.POST("/users", binding.JSON(model.CreateUserReq{}), h.createUser)
    g.GET("/users", binding.Query(model.ListUserQuery{}), h.listUsers)
    g.GET("/users/:id", h.getUser)
}

// createUser — receives parsed request via DI injection (pure function style)
func (h *UserHandler) createUser(req model.CreateUserReq, c *jin.Context) {
    user, err := h.svc.CreateUser(c.Request.Context(), req.Username, req.Email)
    if err != nil {
        c.Render(http.StatusInternalServerError, render.JSON{Data: map[string]string{"error": err.Error()}})
        return
    }
    c.Render(http.StatusOK, render.JSON{Data: user})
}

func (h *UserHandler) listUsers(query model.ListUserQuery, c *jin.Context) {
    users, err := h.svc.ListUsers(c.Request.Context(), query.Page, query.PageSize)
    if err != nil {
        c.Render(http.StatusInternalServerError, render.JSON{Data: map[string]string{"error": err.Error()}})
        return
    }
    c.Render(http.StatusOK, render.JSON{Data: users})
}

func (h *UserHandler) getUser(c *jin.Context) {
    id, _ := c.Params.Get("id")
    user, err := h.svc.GetUser(c.Request.Context(), id)
    if err != nil {
        if errors.Is(err, e.ErrUserNotFound) {
            c.Render(http.StatusNotFound, render.JSON{Data: map[string]string{"error": err.Error()}})
            return
        }
        c.Render(http.StatusInternalServerError, render.JSON{Data: map[string]string{"error": "internal error"}})
        return
    }
    c.Render(http.StatusOK, render.JSON{Data: user})
}
```

## Wiring everything in mod.go

The module's lifecycle methods connect all layers through the DI container:

```go
// mod/<name>/mod.go
func (m *Mod) Load(hub *kernel.Hub) error {
    // Load dependencies from DI
    var j *jin.Engine
    if err := hub.Load(&j); err != nil {
        return errors.New("can't load jin.Engine from kernel")
    }
    var db *gorm.DB
    if err := hub.Load(&db); err != nil {
        return errors.New("can't load gorm.DB from kernel")
    }

    // Build layers bottom-up: dao → service → handler
    userDao, err := dao.NewUserDao(db)
    if err != nil {
        return errors.Wrap(err, "failed to init user dao")
    }
    userSvc := service.NewUserService(userDao)
    userHandler := handler.NewUserHandler(userSvc)

    // Register routes
    g := j.Group("/api/<name>")
    userHandler.RegisterRoutes(g)

    return nil
}
```

The convention is: **Load phase** is where business modules wire their layers and register routes, because by this point all infrastructure modules (DB, Redis, HTTP engine) have already Mapped their resources in PreInit.

## DI container, gRPC gateway, and settings

For detailed reference on these topics, read `references/advanced.md`. Quick summary:

- **DI Container**: `hub.Map(&val)` to register, `hub.Load(&var)` to retrieve (always check error), `hub.Invoke(func)` for one-off injection. One value per concrete type — wrap in distinct structs if you need multiples.
- **gRPC Gateway**: Register in PostInit via `hub.Load(&gw)` then `gw.Register(...)`. Routes mount at `/gapi/*`.
- **Settings**: `settings.NewItem[T]` for dynamic, DB-backed config with cache + env override.

## Error handling patterns

Define domain errors in the `e/` package:

```go
// mod/<name>/e/e.go
package e

import "errors"

var (
    ErrUserNotFound    = errors.New("user not found")
    ErrDuplicateUser   = errors.New("duplicate username")
    ErrInvalidInput    = errors.New("invalid input")
)
```

In handlers, map domain errors to HTTP status codes:

```go
user, err := h.svc.GetUser(ctx, id)
if err != nil {
    if errors.Is(err, e.ErrUserNotFound) {
        c.Render(http.StatusNotFound, render.JSON{Data: map[string]string{"error": err.Error()}})
        return
    }
    c.Render(http.StatusInternalServerError, render.JSON{Data: map[string]string{"error": "internal error"}})
    return
}
```

## Testing

For unit tests, integration tests, handler tests, and E2E verification patterns, read `references/advanced.md` (Testing Strategies section). Key points:

- **Service tests**: Mock the dao layer to test business rules in isolation
- **DAO tests**: Use a real test DB (SQLite/MySQL), call `dao.NewXxxDao(db)` directly
- **Handler tests**: Create `jin.New()`, register routes with binding middleware, use `httptest.NewRecorder()` + `engine.ServeHTTP(w, req)`
- **E2E**: `go build ./...` → `go vet ./...` → start server → curl endpoints

## Common pitfalls

1. **Forgetting `defer wg.Done()` in Stop()** — causes shutdown to hang forever.
2. **Mapping non-pointer types** — `hub.Map(db)` vs `hub.Map(&db)`. Always use `&`.
3. **Loading in PreInit** — other modules haven't run PreInit yet, so their dependencies aren't available. Load in Init or later.
4. **Duplicate type in DI** — only one value per type. If you need two `*gorm.DB` (read/write replicas), wrap them: `type ReadDB struct{ *gorm.DB }`.
5. **Missing mapstructure tags** — config fields won't populate from env vars without `mapstructure:"fieldName"` tags.
6. **Not checking Load errors** — a silent nil pointer will panic later. Always handle the error from `hub.Load()`.
7. **Blocking in Start()** — Start runs in a goroutine. If your Start doesn't block (e.g., it's just setup), you probably want Load instead. If it does block (e.g., `http.Serve()`), that's correct.

## Checklist before completing module implementation

- [ ] All model structs have appropriate GORM tags and JSON tags
- [ ] DAO uses `stdao.Std[T]` where applicable with `*model.Type` (pointer)
- [ ] DAO custom queries use `GetTxFromCtx(ctx)` for transaction support
- [ ] Service layer handles business errors with descriptive messages
- [ ] Handlers validate input via `binding.JSON` / `binding.Query` middleware or manual parsing
- [ ] Query structs use `query:"key"` struct tags for binding.Query
- [ ] Handlers map domain errors to appropriate HTTP status codes
- [ ] Routes are registered in `Load()` via jin router group
- [ ] `go build ./...` passes
- [ ] `go vet ./...` passes
- [ ] Manual curl test of key endpoints succeeds
- [ ] Module logging uses `hub.Log` (namespaced), not raw `fmt.Println`
