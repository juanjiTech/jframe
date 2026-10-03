# DI & lifecycle details (Agent)

## Phases

```
Config → PreInit → Init → PostInit → Load → Start (goroutine)
                                              ↓
                                            Stop (concurrent, defer wg.Done())
```

| Phase | Use for |
|-------|---------|
| Config | External config struct (`yaml` + `mapstructure` tags) |
| PreInit | Create clients, `hub.Map` |
| Init | Verify own resources |
| PostInit | Cross-module assemble after peers' PreInit |
| Load | Routes, plugins, business wiring |
| Start | Blocking servers / workers |
| Stop | Cleanup |

**Patterns:** infra client = Config+PreInit+Init+Stop; HTTP API module = Load; worker = Load/PostInit + Start + Stop.

## DI rules

- `hub.Map(&x)` / `hub.Load(&x)` — pointers; check Load errors.
- One value per concrete type; wrap duplicates.
- Type table: `docs/di-reference.md`.
- Importing another module's **package for a type** used in `Load` is OK. Do **not** call other modules' constructors/internals — take instances from Hub.

## Config

```go
type Config struct {
    Addr string `yaml:"addr" mapstructure:"addr"`
}
func (m *Mod) Config() any { return &m.config }
```

YAML key = `Name()`. Env: `MODULENAME_ADDR` (`.` → `_`).
