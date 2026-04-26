![jFrame](https://github.com/juanjiTech/jframe/raw/main/docs/header.webp)

# jFrame

> A modular Go application scaffolding framework with dependency injection kernel.
>
> 一个基于模块化内核和依赖注入的 Go 应用脚手架框架。

[框架文档](https://deepwiki.com/juanjiTech/jframe/)

## 特性

- **模块化内核** — 所有功能以 `Module` 为单位组织，通过 6 阶段生命周期（PreInit / Init / PostInit / Load / Start / Stop）管理启动与关闭
- **依赖注入** — 基于 `inject/v2` 的类型映射容器，模块间通过 `Map` / `Load` / `Invoke` 共享依赖，零耦合
- **反射驱动配置** — 内核利用 `reflect.StructOf` 动态构建配置结构体，每个模块只需声明自己的 Config，Viper 自动按模块名映射 YAML / 环境变量
- **脚手架命令** — `jframe create -n <name>` 基于 example 模板一键生成新模块目录结构
- **协议复用** — HTTP 与 gRPC 通过 cmux 共享同一 TCP 端口
- **可观测性** — 内置 OpenTelemetry (uptrace)、Pyroscope 持续 profiling、Sentry 异常追踪、腾讯云 CLS 日志
- **配置热重载** — Viper + fsnotify 实时监听配置变更

## 项目结构

```
jframe/
├── main.go                  # 入口，调用 cmd.Execute()
├── cmd/
│   ├── init.go              # Cobra 根命令注册
│   ├── server/              # jframe server — 启动服务
│   │   ├── server.go        # 加载配置 → 创建内核 → 注册模块 → 启动
│   │   └── modList/list.go  # 模块注册清单（新增模块在此添加）
│   ├── config/              # jframe config — 生成配置模板
│   └── create/              # jframe create — 脚手架生成新模块
│       └── createMod.go
├── conf/                    # 全局配置结构与加载逻辑
│   ├── config.go            # Viper 加载 + 热重载
│   └── vars.go              # GlobalConfig 定义
├── core/
│   ├── kernel/              # 框架内核
│   │   ├── kernel.go        # Engine (DI容器 + 模块管理 + 生命周期)
│   │   └── module.go        # Module 接口 / Hub / UnimplementedModule
│   └── logx/                # Zap 日志封装
├── mod/                     # 内置模块
│   ├── example/             # 模板模块 (脚手架模板 + DI 用法示例)
│   │   ├── mod.go           # 模块主体，演示完整生命周期
│   │   ├── embed.go         # go:embed 嵌入整个目录供 create 命令使用
│   │   ├── handler/         # HTTP 处理层
│   │   ├── service/         # 业务逻辑层
│   │   ├── dao/             # 数据访问层
│   │   ├── model/           # 数据模型
│   │   └── e/               # 错误码定义
│   ├── jinx/                # HTTP 服务 (jin 框架 + healthcheck)
│   ├── grpcGateway/         # gRPC 服务 + grpc-gateway REST 代理
│   ├── myDB/                # MySQL (GORM)
│   ├── pgsql/               # PostgreSQL (GORM)
│   ├── rds/                 # Redis
│   ├── b2x/                 # Backblaze B2 云存储
│   ├── uptrace/             # OpenTelemetry 分布式追踪
│   ├── pyroscope/           # 持续性能分析
│   └── jinPprof/            # pprof 调试端点
└── pkg/                     # 公共工具包
    ├── auth/                # JWT 认证
    ├── settings/            # 动态设置 (DB + 缓存)
    ├── stdao/               # 泛型 DAO 基类
    ├── cors/                # CORS 中间件
    ├── ctxKey/              # Context Key 常量
    └── utils/               # 加密 / ID生成 / 分页 等
```

## 快速开始

### 前置条件

- Go 1.23+
- MySQL / PostgreSQL (可选)
- Redis (可选)

### 启动

```bash
# 克隆项目
git clone https://github.com/juanjiTech/jframe.git
cd jframe

# 复制配置
cp config.example.yaml config.yaml
# 编辑 config.yaml 填入实际配置

# 启动开发环境依赖 (Redis + MySQL)
docker compose -f docker-compose-dev.yml up -d

# 运行
go run . server -c config.yaml
```

### CLI 命令

```bash
# 启动服务
jframe server -c ./config.yaml

# 生成配置模板
jframe config

# 创建新模块
jframe create -n users            # 在 mod/users/ 下生成模块骨架
jframe create -n users -p mymod   # 指定输出目录
jframe create -n users -f         # 强制覆盖已存在文件
```

## 核心概念

### Module 接口

所有功能模块实现 `kernel.Module` 接口，必须嵌入 `kernel.UnimplementedModule`：

```go
type Mod struct {
    kernel.UnimplementedModule
}

func (m *Mod) Name() string { return "myMod" }

// Config() — 返回配置结构体指针，内核自动反序列化
// PreInit() — 创建客户端，Map 依赖到容器
// Init()    — 校验依赖，健康检查
// PostInit()— 跨模块装配
// Load()    — 注册路由，启用插件
// Start()   — 长驻任务 (每个模块独立 goroutine)
// Stop()    — 优雅关闭
```

### jin HTTP 框架

jframe 使用 `jin`（gin 的 fork），去掉了 binding 包，用 DI 注入替代：

- **`HandlerFunc` 是 `interface{}`** — 任意函数签名，参数通过 `inject.Invoke` 自动注入
- **`jin/middleware/binding`** 提供 `binding.JSON(T{})` / `binding.Query(T{})` 中间件，将请求数据解析后 Map 到 DI，handler 直接作为参数接收
- **响应** 使用 `c.Render(code, render.JSON{Data: data})`

```go
import (
    "github.com/juanjiTech/jin/middleware/binding"
    "github.com/juanjiTech/jin/render"
)

type CreateReq struct {
    Name string `json:"name"`
}

// binding.JSON 解析请求体 → Map 到 DI → handler 参数 req 自动注入
j.POST("/api/users", binding.JSON(CreateReq{}), func(req CreateReq, c *jin.Context) {
    c.Render(http.StatusOK, render.JSON{Data: req.Name})
})
```

通过 `Hub`（包装了 `inject.Injector`）在模块间传递依赖：

```go
// 注册依赖
hub.Map(&myDB)

// 获取依赖
var db *gorm.DB
hub.Load(&db)

// 函数注入
hub.Invoke(func(db *gorm.DB) {
    // 自动注入
})
```

### 配置系统

每个模块通过 `Config()` 返回自己的配置结构体指针，内核使用 `reflect.StructOf` 动态构建带 `mapstructure` tag 的包装结构体，由 Viper 按模块 `Name()` 自动映射 YAML 节点和环境变量。

```yaml
# config.yaml
myMod:
    addr: "localhost"
    port: "3306"
```

等价环境变量: `MYMOD_ADDR=localhost MYMOD_PORT=3306`

## 创建新模块

```bash
jframe create -n myMod
```

生成以下结构：

```
mod/myMod/
├── mod.go          # 模块主体 (Name, 生命周期方法)
├── handler/        # HTTP Handler
├── service/        # 业务逻辑
├── dao/            # 数据访问
├── model/          # 数据模型
└── e/              # 错误码
```

然后在 `cmd/server/modList/list.go` 中注册：

```go
var ModList = []kernel.Module{
    // ... 其他模块
    &myMod.Mod{},
}
```

## Docker

```bash
# 构建镜像
docker build -t jframe .

# 使用 docker-compose 启动
docker compose up -d
```

## License

[MIT](LICENSE)
