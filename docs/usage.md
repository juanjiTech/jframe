# jFrame 使用指南

本文档介绍 jFrame 的架构、CLI 与日常开发流程。产品定位与 AI 友好特性见 [README](../README.md)。

## 架构概览

jFrame 的核心是 `core/kernel/` 中的 Engine：启动时加载 `cmd/server/modList/list.go` 中注册的所有 `kernel.Module`，按固定顺序执行生命周期，并通过 DI 容器（`inject/v2`）在模块间传递依赖。

```
Config 反序列化
    → PreInit（创建资源，Map 到容器）
    → Init（校验依赖）
    → PostInit（跨模块装配）
    → Load（注册路由等）
    → Start（各模块独立 goroutine）
    → Stop（优雅关闭）
```

业务模块按领域/类别划分，彼此不直接 import，只通过 Hub 的 `Map` / `Load` 通信。基础设施模块在较早阶段 Map 连接与引擎，业务模块在 `Load()` 等阶段 Load 所需类型即可——优先查阅 [DI 参考](di-reference.md) 中的共享类型表；若仍不确定，再阅读对应基础设施模块源码。模块内部推荐分层：

```
handler/  → HTTP 请求解析与响应
service/  → 业务逻辑
dao/      → 数据访问（可基于 pkg/stdao）
model/    → 数据模型与 DTO
e/        → 领域错误码
```

基础设施模块（数据库、Redis、HTTP 网关、可观测性等）与业务模块使用同一套 Module 接口，在 `modList` 中统一注册。

## 项目结构

```
jframe/
├── main.go
├── cmd/
│   ├── server/              # 启动服务
│   │   └── modList/list.go  # ★ 模块注册清单
│   ├── config/              # 生成配置模板
│   └── create/              # 脚手架生成新模块
├── conf/                    # 全局配置
├── core/kernel/             # 内核与 Module 接口
├── mod/                     # 内置与业务模块
│   └── example/             # create 命令的模板来源
└── pkg/                     # 公共工具（auth、stdao、settings 等）
```

## CLI 命令

### `server`

加载配置、初始化内核、启动所有已注册模块。

```bash
go run . server -c ./config.yaml
```

### `config`

扫描所有已注册模块的 `Config()`，生成 YAML 配置模板。

```bash
go run . config              # 默认写入 ./config.yaml
go run . config -p ./config.example.yaml -f   # 指定路径并强制覆盖
```

### `create`

基于 `mod/example/` 生成新模块目录。

```bash
go run . create -n users            # 生成 mod/users/
go run . create -n users -p mymod   # 指定输出目录
go run . create -n users -f         # 强制覆盖
```

生成后在 `cmd/server/modList/list.go` 注册：

```go
var ModList = []kernel.Module{
    // ...
    &users.Mod{},
}
```

## Module 接口

所有模块实现 `kernel.Module`，并嵌入 `kernel.UnimplementedModule`：

```go
type Mod struct {
    kernel.UnimplementedModule
}

func (m *Mod) Name() string { return "myMod" }

// Config()  — 返回配置结构体指针，nil 表示无配置
// PreInit() — 创建客户端，hub.Map 依赖
// Init()    — 校验依赖
// PostInit()— 跨模块装配
// Load()    — 注册路由
// Start()   — 长驻任务（独立 goroutine）
// Stop()    — 优雅关闭（须 defer wg.Done()）
```

## 配置系统

每个模块通过 `Config()` 返回配置结构体，字段须同时带 `yaml` 与 `mapstructure` tag。内核按模块 `Name()` 映射 YAML 节点：

```yaml
myMod:
    addr: "localhost"
    port: "3306"
```

等价环境变量：`MYMOD_ADDR=localhost`、`MYMOD_PORT=3306`。优先级：环境变量 > config.yaml > 默认值。支持 Viper + fsnotify 热重载。

## jin HTTP 框架

jFrame 使用 `jin`（gin fork），用 DI 替代传统 binding：

- Handler 为任意函数签名，参数由 `inject.Invoke` 注入
- `binding.JSON(T{})` / `binding.Query(T{})` 解析请求并 Map 到 DI
- 响应：`c.Render(code, render.JSON{Data: data})`

```go
j.POST("/api/users", binding.JSON(CreateReq{}), func(req CreateReq, c *jin.Context) {
    c.Render(http.StatusOK, render.JSON{Data: req.Name})
})
```

## DI 容器

Map / Load 基本用法与**共享类型一览表**见 [DI 参考](di-reference.md)。

## Docker

```bash
docker build -t jframe .
docker compose up -d
```

本地开发依赖（Redis + MySQL）：

```bash
docker compose -f docker-compose-dev.yml up -d
```
