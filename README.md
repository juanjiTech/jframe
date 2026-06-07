![jFrame](https://github.com/juanjiTech/jframe/raw/main/docs/header.webp)

# jFrame

> AI 友好型 Go 开发框架，面向服务端与桌面端应用——模块化领域边界、固定生命周期与分层约定，支持快速迭代与可审查的变更。

jFrame 是一个基于模块化内核与依赖注入的 Go 应用脚手架。业务按模块组织，模块内部再按 handler / service / dao / model 分层；启动顺序由内核统一编排，模块内通过 DI 按类型取用依赖——多数情况下查 [DI 参考](docs/di-reference.md) 并在当前模块内开发即可，不必为了取依赖而通读其他模块。配合 `jframe create` 脚手架与仓库内置的 Agent 指南，人类开发者与 Cursor、Claude 等 AI 助手都能在同一套结构约束下协作，少耗上下文、审查 diff 也更可预期。

除服务端场景外，jFrame 也已在团队内部大量用于桌面端应用开发（闭源项目，此处不展开细节）。

[框架文档（DeepWiki）](https://deepwiki.com/juanjiTech/jframe/) · [使用指南](docs/usage.md) · [AI 开发指南](docs/ai-development.md)

## 为什么 AI 友好

- **模块化领域边界 + 分层约定** — 业务模块按类别高度内聚，模块内固定 handler → service → dao → model 分层；AI 只需聚焦当前模块与层级，减少无关上下文干扰。
- **固定生命周期 + DI** — 内核按 Config → PreInit → … → Stop 六阶段统一编排启动，AI 不必把注意力耗在启动顺序与模块调度上。写业务模块时在约定阶段（如 `Load()`）通过 `hub.Load` 按类型取用依赖；优先查 [DI 参考](docs/di-reference.md)，上下文可集中在当前模块内。
- **脚手架 + 金标准模板** — `jframe create -n <name>` 基于 `mod/example/` 生成一致目录结构，AI 生成代码有明确参照。
- **Agent 指南开箱即用** — 仓库提供 [`CLAUDE.md`](CLAUDE.md) 与 [Agent Skills](.claude/skills/)（模块设计 / 模块实现），Cursor、Claude Code 可直接读取框架约定。
- **结构约束，审查友好** — 模块边界、DI 规则、配置 tag 均有硬性约定，便于 AI 辅助开发与人工 / AI 代码审查。

## 特性

- **模块化内核** — 所有功能以 `Module` 为单位组织，六阶段生命周期管理启动与关闭
- **依赖注入** — 基于 `inject/v2`，模块间通过 `Map` / `Load` / `Invoke` 共享依赖，零直接耦合
- **反射驱动配置** — 每个模块声明 Config，Viper 按模块名自动映射 YAML / 环境变量，支持热重载
- **脚手架命令** — `jframe create -n <name>` 一键生成模块骨架
- **协议复用** — HTTP 与 gRPC 通过 cmux 共享同一 TCP 端口
- **可观测性** — OpenTelemetry (uptrace)、Pyroscope、Sentry、腾讯云 CLS 日志
- **AI 工具链** — `CLAUDE.md` + Agent Skills，适配 Cursor / Claude 团队工作流

## 快速开始

### 前置条件

- Go 1.23+
- MySQL / PostgreSQL（可选）
- Redis（可选）

### 启动

```bash
# 克隆项目
git clone https://github.com/juanjiTech/jframe.git
cd jframe

# 复制配置
cp config.example.yaml config.yaml
# 编辑 config.yaml 填入实际配置

# 启动开发环境依赖（可选：本地 Redis + MySQL）
docker compose -f docker-compose-dev.yml up -d

# 运行
go run . server -c config.yaml
```

### 常用命令

```bash
go run . server -c ./config.yaml    # 启动服务
go run . config                     # 生成配置模板
go run . create -n users            # 创建新模块
```

模块注册、配置说明、项目结构等详见 [使用指南](docs/usage.md)。

## 用 AI 开发

1. 让 Agent 先读 [`CLAUDE.md`](CLAUDE.md) — 涵盖 Module 生命周期、DI 规则、jin HTTP 约定等；查可 Load 的类型用 [`docs/di-reference.md`](docs/di-reference.md)。
2. 新建功能时执行 `go run . create -n <moduleName>`，再按 `mod/example/` 结构实现。
3. Cursor / Claude 可自动加载 `.claude/skills/jframe-module-design`（设计新模块）与 `jframe-module-dev`（实现 handler / service / dao）。

更多场景与提示词建议见 [AI 开发指南](docs/ai-development.md)。

## 文档

| 文档 | 说明 |
|------|------|
| [docs/di-reference.md](docs/di-reference.md) | DI 共享类型表（权威维护位置） |
| [docs/usage.md](docs/usage.md) | 架构概览、CLI、模块创建、配置与 Docker |
| [docs/ai-development.md](docs/ai-development.md) | 面向 Cursor / Claude 团队的开发工作流 |
| [CLAUDE.md](CLAUDE.md) | Agent 开发约定（生命周期、分层、jin 等；DI 类型表见 di-reference） |
| [DeepWiki](https://deepwiki.com/juanjiTech/jframe/) | 在线框架文档 |

## License

[MIT](LICENSE)
