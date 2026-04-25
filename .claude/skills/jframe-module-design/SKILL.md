---
name: jframe-module-design
description: "Design, plan, and scaffold new jframe kernel modules. Invoke this skill when the user wants to: create a new module (jframe create), add a new feature/service/integration as a module, scaffold module structure, plan module lifecycle phases, design module config structs, plan DI dependencies (Map/Load), or integrate an external service (DB, cache, MQ, S3, SDK) into the jframe architecture. Trigger on phrases like 'new module', 'add module', 'create mod', 'integrate X into jframe', 'design a module for X', or any request to add new functionality that should be encapsulated as a jframe kernel.Module. This skill covers architecture design — from requirements through to a registered module skeleton. Do NOT use this skill for implementing code inside an existing module (use jframe-module-dev instead), modifying existing modules, or debugging module issues."
---

# jframe Module Design

This skill guides you through designing and creating a new module for the jframe Go scaffolding framework. jframe uses a kernel-based modular architecture where every feature is a `kernel.Module` with a structured lifecycle managed by a dependency injection container.

## When to use this skill

- User wants to add a new feature/service to the jframe application
- User asks to create/scaffold/design a new module
- User wants to integrate an external service (database, cache, API, etc.) as a module

## Overview of the module system

Every module implements `kernel.Module` (defined in `core/kernel/module.go`). The kernel runs modules through 6 sequential lifecycle phases, each receiving a `*kernel.Hub` that wraps the DI container (`inject.Injector`) plus a namespaced logger:

```
Config loading → PreInit → Init → PostInit → Load → Start (goroutine)
                                                         ↓
                                                     Stop (concurrent via WaitGroup)
```

Modules interact exclusively through the Hub's DI container — they never import each other directly. This is the core decoupling mechanism.

## Step 1: Gather requirements

Before writing any code, clarify these with the user:

1. **What does the module do?** — Is it a service client (DB, cache, API), a protocol server (HTTP routes, gRPC), a background worker, or business logic?
2. **What name should it have?** — The `Name()` return value determines the YAML config key and log namespace. Convention: camelCase for infrastructure (`myDB`, `rds`), descriptive for business modules (`users`, `orders`).
3. **Does it need configuration?** — External services almost always do. Business modules may or may not.
4. **What does it produce for other modules?** — What will it `Map` into the DI container? (e.g., `*gorm.DB`, `*redis.Client`, a custom service interface)
5. **What does it consume from other modules?** — What will it `Load` from the DI container? (e.g., `*jin.Engine` for HTTP routes, `*gorm.DB` for data access)
6. **Does it need a long-running goroutine?** — Only if it runs a server or background loop.

## Step 2: Choose lifecycle phases

Not every module needs every phase. Here's the decision guide:

| Phase | When to implement | Examples |
|-------|------------------|----------|
| `Config()` | Module has external configuration | DB connection params, API keys, feature flags |
| `PreInit()` | Module creates clients/resources and Maps them into DI | Connect to DB, create HTTP engine, init SDK client |
| `Init()` | Module needs to verify its own dependencies work | Ping DB, health check, validate config values |
| `PostInit()` | Module needs dependencies from other modules that were set up in PreInit | Load jin.Engine to mount routes, load DB to set up tracing |
| `Load()` | Late wiring — register HTTP routes, enable plugins, cross-module integration | Register `/api/users/*` routes, enable OTel on DB driver |
| `Start()` | Module runs a long-lived task (runs in its own goroutine) | HTTP server `Serve()`, gRPC server `Serve()`, background worker |
| `Stop()` | Module needs graceful shutdown | Close connections, drain queues, shutdown HTTP server with timeout |

**Common patterns by module type:**

- **Service client** (DB, Redis, S3): `Config` + `PreInit` (connect & Map) + `Init` (verify) + `Stop` (close)
- **HTTP route group** (users API, admin API): `Load` (load jin.Engine & DB, register routes)
- **Protocol server** (HTTP, gRPC): `PreInit` (create engine & Map) + `Start` (listen & serve) + `Stop` (shutdown)
- **Background worker**: `PostInit` (load deps) + `Start` (run loop) + `Stop` (signal & wait)

## Step 3: Design the config struct

If the module needs configuration, define a Config struct with `yaml` and `mapstructure` tags. The kernel uses `reflect.StructOf` to dynamically wrap this struct and unmarshal from Viper, keyed by `Name()`.

```go
type Config struct {
    Addr     string `yaml:"addr" mapstructure:"addr"`
    Port     string `yaml:"port" mapstructure:"port"`
    APIKey   string `yaml:"apiKey" mapstructure:"apiKey"`
    Debug    bool   `yaml:"debug" mapstructure:"debug"`
}
```

The corresponding YAML section uses the module's `Name()`:

```yaml
moduleName:
    addr: "localhost"
    port: "8080"
    apiKey: ""
    debug: false
```

Environment variable override: `MODULENAME_ADDR=localhost` (Viper replaces `.` with `_`, case-insensitive).

Store the config as a field on the Mod struct and return a pointer from `Config()`:

```go
type Mod struct {
    kernel.UnimplementedModule
    config Config
}

func (m *Mod) Config() any { return &m.config }
```

## Step 4: Plan DI dependencies

The DI container is a type-map — one value per concrete type. Plan what types your module will Map and Load.

**Mapping (producing):**
```go
// In PreInit — always Map pointer types
hub.Map(&db)        // stores as **gorm.DB
hub.Map(&rdb)       // stores as **redis.Client
hub.Map(&m.engine)  // stores as **jin.Engine
```

**Loading (consuming):**
```go
// In PostInit/Load/Start — Load into a matching pointer type
var db *gorm.DB
if err := hub.Load(&db); err != nil {
    return errors.New("can't load gorm.DB from kernel")
}
```

**Ordering rule:** A module can only Load what another module (or the kernel) has already Mapped. Since phases run sequentially across all modules:
- PreInit Maps are available to all modules' Init, PostInit, Load, and Start
- Init Maps are available from PostInit onwards
- The kernel Maps `*net.Listener` and `cmux.CMux` before any module runs

## Step 5: Create the module structure

Use the CLI scaffolding command:

```bash
go run . create -n <moduleName>
# or if built: jframe create -n <moduleName>
```

This generates from the `mod/example/` template:

```
mod/<moduleName>/
├── mod.go          # Module struct + lifecycle methods
├── handler/        # HTTP handlers (jin routes)
│   └── <moduleName>.go
├── service/        # Business logic
│   └── <moduleName>.go
├── dao/            # Data access (GORM queries)
│   └── <moduleName>.go
├── model/          # Data models (GORM structs)
│   └── <moduleName>.go
└── e/              # Error codes / custom errors
    └── e.go
```

The scaffolding replaces all occurrences of "example" with the module name. After generation, you'll need to customize `mod.go` with the actual lifecycle logic designed in steps 2-4.

If the module is simple (e.g., a pure service client with no HTTP routes), some sub-packages may be unnecessary — it's fine to remove empty ones. Infrastructure modules like `myDB` and `rds` only have `mod.go`.

## Step 6: Register the module

Add the module to `cmd/server/modList/list.go`:

```go
import (
    // ...existing imports
    "<module-import-path>"
)

var ModList = []kernel.Module{
    // ...existing modules
    &<moduleName>.Mod{},
}
```

**Module ordering matters** for implicit dependency ordering within the same lifecycle phase. If module A's PreInit Maps something that module B's PreInit needs, A must appear before B in ModList. Across phases (A's PreInit → B's Init), ordering doesn't matter since all PreInits run before any Init.

## Step 7: Add config to config.yaml

If the module has config, add the default values to `config.yaml` and `config.example.yaml`:

```yaml
# config.yaml
moduleName:
    addr: "localhost"
    port: "8080"
```

You can also regenerate the full config template:

```bash
go run . config
```

This iterates all registered modules' `Config()` and outputs a complete YAML template.

## Complete module template

Here's a production-ready template for a typical module that connects to an external service and exposes HTTP routes:

```go
package myMod

import (
    "context"
    "errors"
    "sync"

    "github.com/juanjiTech/jframe/core/kernel"
    "github.com/juanjiTech/jin"
    "github.com/juanjiTech/jin/middleware/binding"
    "github.com/juanjiTech/jin/render"
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

func (m *Mod) Name() string    { return "myMod" }
func (m *Mod) Config() any     { return &m.config }

func (m *Mod) PreInit(hub *kernel.Hub) error {
    // Create clients, initialize resources
    // hub.Map(&client)
    return nil
}

func (m *Mod) Init(hub *kernel.Hub) error {
    // Verify dependencies
    return nil
}

func (m *Mod) Load(hub *kernel.Hub) error {
    // Load jin engine and register routes
    var j *jin.Engine
    if err := hub.Load(&j); err != nil {
        return errors.New("can't load jin.Engine from kernel")
    }

    var db *gorm.DB
    if err := hub.Load(&db); err != nil {
        return errors.New("can't load gorm.DB from kernel")
    }

    // Register route group
    // jin's HandlerFunc is interface{} — any function signature works
    // Parameters are auto-injected via inject.Invoke
    // binding.JSON/Query middleware parses request data → Maps to DI
    g := j.Group("/api/myMod")
    h := handler{db: db, config: m.config}
    g.GET("/list", h.list)
    g.POST("/create", binding.JSON(createReq{}), h.create)

    return nil
}

func (m *Mod) Stop(wg *sync.WaitGroup, ctx context.Context) error {
    defer wg.Done()
    // Cleanup resources
    return nil
}

type createReq struct {
    Name string `json:"name"`
}

type handler struct {
    db     *gorm.DB
    config Config
}

// list uses classic func(*jin.Context) signature
func (h *handler) list(c *jin.Context) {
    c.Render(http.StatusOK, render.JSON{Data: "ok"})
}

// create receives parsed request via DI injection (pure function pattern)
func (h *handler) create(req createReq, c *jin.Context) {
    c.Render(http.StatusOK, render.JSON{Data: req.Name})
}
```

## Next step: implement the module

Once the module skeleton is registered and compiles, switch to the **jframe-module-dev** skill to implement the internal layers (model → dao → service → handler), wire DI in `Load()`, and register routes. That skill covers jin handler patterns, binding middleware, stdao, testing, and the full implementation checklist.

## Checklist before finishing

- [ ] `mod.go` implements `kernel.Module` with `var _ kernel.Module = (*Mod)(nil)` compile-time check
- [ ] Embeds `kernel.UnimplementedModule`
- [ ] `Name()` returns a unique, camelCase identifier
- [ ] `Config()` returns `&m.config` (pointer) if config exists, or is omitted (defaults to nil via UnimplementedModule)
- [ ] Config struct has both `yaml` and `mapstructure` tags on every field
- [ ] All `Map` calls use pointer arguments: `hub.Map(&thing)`
- [ ] All `Load` calls check for errors
- [ ] `Stop()` calls `defer wg.Done()` as its first statement
- [ ] Module is added to `cmd/server/modList/list.go`
- [ ] Config defaults are in `config.yaml` and `config.example.yaml`
- [ ] `go build ./...` succeeds
