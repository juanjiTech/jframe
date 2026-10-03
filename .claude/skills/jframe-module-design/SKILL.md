---
name: jframe-module-design
description: >-
  Design and scaffold new jframe kernel modules: lifecycle phases, Config,
  Map/Load plan, jframe create, modList registration. Trigger on new module,
  create mod, integrate DB/cache/MQ/SDK as a module, or architecture planning
  before code. Do NOT use for implementing inside an existing mod/ (use
  jframe-module-dev) or kernel/core changes.
---

# jframe Module Design

Plan and scaffold a new `kernel.Module`. For writing handler/dao/service code after the skeleton exists, switch to **jframe-module-dev**.

## Progressive reading

| Need | Read |
|------|------|
| Default | This file |
| DI / lifecycle detail | [`references/di-and-lifecycle.md`](references/di-and-lifecycle.md) |
| Hand-written mod.go template | [`references/module-template.md`](references/module-template.md) |
| Shared Load types | `docs/di-reference.md` |
| Filled CRUD gold standard | `mod/example/` |

## Steps

### 1. Requirements

Name (`Name()` → YAML key), Config?, Map outputs?, Load inputs?, long-running Start?, HTTP vs worker vs infra client.

### 2. Lifecycle

Pick only needed phases. Common:

- **Infra client:** Config + PreInit (Map) + Init + Stop  
- **HTTP API:** Load (Load jin + DB, routes)  
- **Worker:** Load/PostInit + Start + Stop  

Details: `references/di-and-lifecycle.md`.

### 3. Config

Struct with **both** `yaml` and `mapstructure` tags; `Config() any { return &m.config }`.

### 4. DI plan

Map in PreInit with pointers; Load in Load/Init+ from `docs/di-reference.md`. Modules must not construct peers' clients — only Hub. Importing a type package for `Load` is fine.

### 5. Scaffold

```bash
go run . create -n <moduleName>
```

Copies **`mod/example/`** (Item CRUD + `pkg/stdao`), replaces `example` → `<moduleName>`, skips `embed.go`. Rename types/routes to the domain. Drop unused layers for infra-only modules.

### 6. Register

Add `&<moduleName>.Mod{}` to `cmd/server/modList/list.go`. Same-phase order matters if B.PreInit Loads what A.PreInit Maps.

### 7. Config YAML

Add section under `Name()`, or `go run . config`.

## Next

Implement layers with **jframe-module-dev** (start from generated code / `mod/example`).

## Checklist

- [ ] Embeds `UnimplementedModule`; `var _ kernel.Module = (*Mod)(nil)`
- [ ] Config dual tags; Map/Load pointers + error checks
- [ ] `Stop`: `defer wg.Done()`
- [ ] In `modList` + config defaults
- [ ] `go build ./...`
