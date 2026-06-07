# DI 参考

本文档维护 jFrame DI 容器中**可由业务模块 Load 的共享类型**。新增基础设施模块 Map 新类型时，请同步更新此表。

## 基本用法

```go
hub.Map(&db)              // 注册（传指针）
var db *gorm.DB
hub.Load(&db)             // 获取（须检查 error）
hub.Invoke(func(db *gorm.DB) { ... })
```

**规则摘要：**

- Map 总是传指针：`hub.Map(&value)`
- 同类型只能 Map 一个值；多个同类依赖需用包装类型区分
- Map 使用 `hub.Map(&value)` 时，容器内存储类型常为 `**T`，Load 时写 `var v *T; hub.Load(&v)`

业务模块通常在 `Load()` 阶段 Load 依赖并组装 handler / service；基础设施模块在 `PreInit()` 连接资源并 Map。

## 共享类型一览

以下类型由内核或内置模块 Map，业务模块可按类型 Load 使用：

| 类型 | Map 来源 | 可用阶段 |
|------|----------|----------|
| `*net.Listener` | 内核 (`cmd/server/server.go`) | PreInit 起 |
| `cmux.CMux` | 内核 (`cmd/server/server.go`) | PreInit 起 |
| `**jin.Engine` | `jinx` PreInit | Init 起 |
| `**gorm.DB` | `myDB` / `pgsql` PreInit | Init 起 |
| `**redis.Client` | `rds` PreInit | Init 起 |
| `*grpc.Server` | `grpcGateway` PreInit | Init 起 |
| `*gateway.Gateway` | `grpcGateway` PostInit | Load 起 |
| `**b2.Client`, `**b2.Bucket` | `b2x` PreInit | Init 起 |

> **Load 示例：** Map 的是 `&db`（`*gorm.DB`），Load 时 `var db *gorm.DB; hub.Load(&db)`。

## 何时读基础设施模块源码

**优先查本表** — 多数业务开发只需知道类型与可用阶段，在约定生命周期阶段 `hub.Load` 即可。

**仍不确定时** — 可以阅读对应模块（如 `mod/myDB/`、`mod/jinx/`）的 `mod.go`，确认 Map 时机、配置项或边界行为；这不违背 jFrame 的模块边界，只是多消耗一些上下文。

**不要**在业务模块里重复创建已有基础设施提供的连接（例如再 `gorm.Open` 一次），应 Load 容器内已有实例。

## 延伸阅读

- [使用指南](usage.md) — 架构与 Module 生命周期
- [AI 开发指南](ai-development.md) — Agent 工作流
- [CLAUDE.md](../CLAUDE.md) — Agent 开发约定（链至本文档维护 DI 类型表）
