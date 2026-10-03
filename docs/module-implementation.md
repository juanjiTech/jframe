# 业务模块实现层

本文说明 `mod/<name>/` 内部分层怎么写。架构与生命周期见 [usage.md](usage.md)；可 Load 的类型见 [di-reference.md](di-reference.md)。

## 先看金标准

[`mod/example/`](../mod/example/) 是可编译的完整范例（默认不挂 `modList`）：

| 路径 | 职责 |
|------|------|
| `model/item.go` | GORM 模型 + 请求/响应 DTO |
| `dao/item.go` | `pkg/stdao` DAO |
| `service/item.go` | 业务规则 |
| `handler/item.go` | jin 路由与 binding |
| `e/e.go` | 领域错误 |
| `mod.go` | `Load`：Load 依赖 → 组装 → 挂路由 |

```bash
go run . create -n orders   # 复制 example，并把 example 替换为 orders
```

## 数据流

```
handler → service → dao → model
```

`mod.go` 的 **Load** 阶段自底向上组装：`NewXxxDao(db)` → `NewXxxService(dao)` → `NewXxxHandler(svc)` → `j.Group(...).RegisterRoutes`。

## 各层要点

### model

- 持久化结构：GORM tag；常用 `stdao.Model`（详见 [stdao.md](stdao.md)）。
- 请求/响应 DTO：普通 struct，不要混进 GORM 模型文件以外的无关类型亦可分文件。

### dao

- 只用 `github.com/juanjiTech/jframe/pkg/stdao`。
- `Init(db)` 负责 AutoMigrate；自定义查询用 `GetTxFromCtx`。
- 详情：[stdao.md](stdao.md)。

### service

- 只依赖 dao 与领域类型；返回 `e/` 包错误，不直接写 HTTP 状态码。

### handler

- jin（不是 gin）：`binding.JSON` / `binding.Query`，响应用 `c.Render(..., render.JSON{Data: ...})`。
- 路径参数：`c.Params.ByName("id")`（无 `c.Param`）。
- 范例：`mod/example/handler/item.go`。

### e

- 领域错误用 `errors.New` / `fmt.Errorf`；handler 里 `errors.Is` 映射 HTTP 状态。

## 非 HTTP / Worker 模块

无 jin 时仍可保留 `dao` + `service`：在 `Load` 里 `hub.Load` 依赖并构造 service；在 `Start`（内核已放独立 goroutine）里跑消费循环；`Stop` 里 `cancel` + `wg.Done()`。不必强行建空 `handler/`。

## 文档怎么读（渐进）

1. 本页 + `mod/example/` — 多数实现足够  
2. [stdao.md](stdao.md) — 写 DAO / 事务时  
3. [di-reference.md](di-reference.md) — 不清楚 Load 什么类型时  
4. Agent Skills — 设计用 `jframe-module-design`，实现用 `jframe-module-dev`（细节在 skill 的 `references/`）
