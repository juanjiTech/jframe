# AI 开发指南

jFrame 面向使用 Cursor、Claude Code 等 AI 编程助手的团队。本文说明如何让 Agent 高效、可控地参与开发。

## 核心思路

jFrame 的「AI 友好」来自可预测的结构，而非某个特定 AI 产品：

1. **模块即边界** — 一次改动通常只涉及 `mod/<name>/` 下的一个业务模块。
2. **生命周期由内核编排** — 启动顺序与阶段调度框架已固定，Agent 不必管理「谁先启动、谁 Map 谁 Load」；只需在约定阶段（基础设施 PreInit Map、业务 Load 组装）写当前模块代码。
3. **依赖按类型取用，优先查 DI 参考** — 业务模块在 `Load()` 等阶段 `hub.Load(&db)` 即可拿到所需依赖；先查 [DI 参考](di-reference.md) 中的共享类型表，通常不必为了取依赖而通读其他模块。若对 Map 时机、配置或行为仍不确定，再读对应基础设施模块（如 `mod/myDB/mod.go`）亦可。
4. **模板可复制** — `mod/example/` 与 `create` 命令输出即金标准。
5. **规则可读取** — `CLAUDE.md` 与 Skills 把约定写进仓库，减少臆测。

## 推荐工作流

### 1. 初始化上下文

在新对话或新任务开始时，提示 Agent：

> 请先阅读 CLAUDE.md，了解 jframe 的 Module 生命周期、DI 规则与分层约定；查依赖类型时用 docs/di-reference.md。

Cursor 会自动加载项目 rules；Claude Code 可直接 `@CLAUDE.md`、`@docs/di-reference.md`。

### 2. 新建模块

**设计阶段** — 说明需求，让 Agent 使用 `jframe-module-design` skill（或手动参照 `mod/example/`）规划：

- 模块名、`Config` 字段
- 各生命周期阶段做什么
- 需要从 DI 容器 Load 哪些依赖（对照 [DI 参考](di-reference.md)）

**脚手架** — 执行后再写业务代码：

```bash
go run . create -n order
```

**实现阶段** — 使用 `jframe-module-dev` skill，在 `handler/` → `service/` → `dao/` 自底向上实现，在 `mod.go` 的 `Load()` 中组装并注册路由。

**注册** — 在 `cmd/server/modList/list.go` 添加 `&order.Mod{}`。

### 3. 修改现有模块

提示 Agent 明确模块名与层级，例如：

> 在 `mod/users/` 的 service 层增加按邮箱查询用户，不要改动其他模块。

Module 边界 + 分层让 diff 范围自然受限，便于人工与 AI 审查。

### 4. 配置变更

新模块的 `Config()` 结构体写好 `yaml` / `mapstructure` tag 后：

```bash
go run . config -p config.example.yaml -f
```

## 仓库内 Agent 资源

| 资源 | 路径 | 用途 |
|------|------|------|
| DI 共享类型表 | [`docs/di-reference.md`](di-reference.md) | **权威维护位置**：可 Load 的类型、Map 来源、可用阶段 |
| Agent 开发参考 | [`CLAUDE.md`](../CLAUDE.md) | 生命周期、DI 规则、jin、create 流程、pkg 工具 |
| 模块设计 Skill | [`.claude/skills/jframe-module-design`](../.claude/skills/jframe-module-design/SKILL.md) | 新建模块、集成外部服务前的架构设计 |
| 模块实现 Skill | [`.claude/skills/jframe-module-dev`](../.claude/skills/jframe-module-dev/SKILL.md) | handler / service / dao 实现与路由注册 |
| 模板模块 | [`mod/example/`](../mod/example/) | create 命令复制的金标准 |

## 给 Agent 的约束提示（可复制）

```
- 新功能 = 新 kernel.Module，业务逻辑不要写在 cmd/ 或 main.go
- 启动顺序由内核编排，业务模块按约定阶段写代码即可
- 依赖优先查 docs/di-reference.md，在 Load() 等阶段 hub.Load 按类型取用；不确定时再读对应 mod/<infra>/mod.go
- 不要在业务模块里重复创建 DI 容器已有的连接
- 模块间通过 Hub Map/Load 通信，禁止跨模块 direct import
- Config 字段必须同时有 yaml 和 mapstructure tag
- hub.Map 传指针；Stop 必须 defer wg.Done()
- HTTP 用 jin + binding.JSON/Query，响应用 render.JSON
- 参考 mod/example/ 的分层与 mod.go Load() 组装方式
```

## 常见问题

**Agent 改了不该改的文件？**  
在 prompt 中限定 `mod/<target>/` 路径，并强调「只改当前模块」。

**Agent 不了解 jin 与 gin 的差异？**  
指向 `CLAUDE.md` 的 jin 章节：无 `c.JSON`，Handler 为 DI 注入函数。

**需要集成 MySQL / Redis / gRPC？**  
先查 [DI 参考](di-reference.md)，在当前模块的 `Load()` 里 `hub.Load` 取用；不要在业务模块里重复创建连接。若表中没有或行为不清楚，再读对应基础设施模块源码。

**Agent 加载了过多无关模块？**  
提示优先 `@docs/di-reference.md` 与当前 `mod/<name>/`；仅在 DI 表无法解答时再打开 `mod/myDB`、`mod/jinx` 等。启动编排由内核负责，一般无需分析其他模块的完整生命周期实现。

## 延伸阅读

- [DI 参考](di-reference.md) — 共享类型表
- [使用指南](usage.md) — CLI、配置、Docker
- [README](../README.md) — 产品定位与快速开始
- [DeepWiki](https://deepwiki.com/juanjiTech/jframe/) — 在线文档
