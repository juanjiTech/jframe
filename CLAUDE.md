# CLAUDE.md — jframe 项目 Agent 开发指南

## 项目概述

jframe 是一个基于模块化内核的 Go 应用脚手架框架。所有功能以 `kernel.Module` 为单位组织，通过依赖注入容器 (`inject/v2`) 在模块间传递依赖，实现零耦合。

**仓库:** github.com/juanjiTech/jframe
**Go 版本:** 1.23+
**入口:** `main.go` → `cmd.Execute()` → Cobra 子命令

## 核心架构

### 模块系统

所有功能都封装为 `kernel.Module`（定义在 `core/kernel/module.go`）。

**Module 接口:**

```
Name() string                           — 唯一标识符，同时作为配置 YAML key
Config() any                            — 返回配置结构体指针，nil 表示无配置
PreInit(*Hub) / Init(*Hub) / PostInit(*Hub) / Load(*Hub) — 4 个顺序初始化阶段
Start(*Hub)                             — 在独立 goroutine 中运行（长驻任务）
Stop(*sync.WaitGroup, context.Context)  — 优雅关闭（并发执行，必须 defer wg.Done()）
```

**生命周期执行顺序:**
1. Config 反序列化（遍历所有模块）
2. PreInit（遍历所有模块）— 创建资源，Map 到 DI 容器
3. Init（遍历所有模块）— 校验依赖
4. PostInit（遍历所有模块）— 跨模块装配
5. Load（遍历所有模块）— 注册路由、启用插件
6. Start（每个模块一个 goroutine）— 长驻服务

所有模块**必须**嵌入 `kernel.UnimplementedModule`，这是 gRPC 风格的前向兼容模式（`mustEmbedUnimplementedModule()` 未导出方法强制嵌入）。

### 依赖注入 (DI)

基于 `github.com/juanjiTech/inject/v2`，一个反射驱动的类型映射容器。

`Hub` 结构体（`core/kernel/module.go:10`）包装了 `inject.Injector` + `*zap.SugaredLogger`，传入每个生命周期方法。

**核心 DI 操作:**
- `hub.Map(&value)` — 注册依赖（按指针类型作为 key），一个类型只能存一个值
- `hub.Load(&variable)` — 获取依赖（必须检查 error）
- `hub.Invoke(func(dep Type) { ... })` — 函数参数自动注入
- `hub.Value(reflect.Type)` — 底层按类型直接查找

**重要规则:**
- Map 总是传指针：`hub.Map(&db)` 不是 `hub.Map(db)`
- Load 只能获取已被其他模块 Map 过的类型
- 同类型只能存一个值，需要多个相同类型用包装结构体区分
- 内核在所有模块之前 Map 了 `*net.Listener` 和 `cmux.CMux`

### 配置系统

**全局配置:** `conf/vars.go` 的 `GlobalConfig` 结构体，通过 `conf.Get()` 获取。

**模块配置:** 每个模块的 `Config()` 返回自定义结构体指针。内核使用 `reflect.StructOf` 动态构建包含 `mapstructure` tag 的结构体，由 Viper 按 `Name()` 自动映射：

```go
// 内核内部操作 (kernel.go:69-85)
structType := reflect.StructOf([]reflect.StructField{{
    Name: "Config",
    Type: reflect.TypeOf(moduleConfig),
    Tag:  reflect.StructTag(`mapstructure:"moduleName"`),
}})
instance := reflect.New(structType)
instance.Elem().Field(0).Set(reflect.ValueOf(moduleConfig))
viper.Unmarshal(instance.Interface())
```

**配置来源优先级:** 环境变量 > config.yaml > 默认值
**环境变量映射:** `MODULE_FIELD` → `Module.Field`（Viper 的 `.` → `_` 替换器）
**热重载:** Viper + fsnotify 监听文件变更

Config 结构体字段**必须同时有** `yaml` 和 `mapstructure` tag。

## 项目目录结构

```
cmd/
  init.go                    — Cobra 根命令，注册 server/config/create 子命令
  server/server.go           — 启动流程：加载配置 → Sentry → TCP 监听 → cmux → 创建内核 → 注册模块 → 启动
  server/modList/list.go     — ★ 模块注册清单 — 新增模块必须在这里添加
  config/config.go           — 生成配置模板命令
  create/createMod.go        — 脚手架命令，基于 mod/example/ 模板生成新模块

conf/
  config.go                  — Viper 加载逻辑 + 热重载
  vars.go                    — GlobalConfig 定义

core/
  kernel/kernel.go           — Engine：DI 容器 + 模块管理 + 生命周期引擎
  kernel/module.go           — Module 接口 + Hub + UnimplementedModule
  logx/logger.go             — Zap 日志封装，支持文件轮转 + CLS 云日志

mod/                         — 所有模块（内置 + 业务）
  example/                   — ★ 模板模块（脚手架源模板）
    mod.go                   — 示例生命周期实现 + DI 用法演示
    embed.go                 — go:embed 嵌入目录
    handler/ service/ dao/ model/ e/  — 分层骨架

  jinx/mod.go                — HTTP 服务（jin 框架），PreInit 创建 jin.Engine 并 Map
  grpcGateway/mod.go         — gRPC 服务 + grpc-gateway REST 代理
  myDB/mod.go                — MySQL (GORM)，PreInit 连接并 Map *gorm.DB
  pgsql/mod.go               — PostgreSQL (GORM)
  rds/mod.go                 — Redis，PreInit 连接并 Map *redis.Client
  uptrace/mod.go             — OpenTelemetry 分布式追踪
  pyroscope/mod.go           — 持续性能分析
  jinPprof/mod.go            — pprof 调试端点
  b2x/mod.go                 — Backblaze B2 云存储

pkg/                         — 公共工具包（非模块，可被任何模块导入）
  stdao/dao.go               — 泛型 DAO 基类 Std[T]，提供 CRUD + 事务传播
  stdao/model.go             — 基础 Model（ULID 主键 + 时间戳 + 软删除）
  settings/setting.go        — 动态设置 Item[T]（DB + 缓存 + 环境变量覆盖）
  auth/module.go             — JWT 认证 (HS256)
  cors/cors.go               — CORS 中间件
  ctxKey/key.go              — Context Key 常量（UID, OrgID, DbTransaction）
  utils/                     — 加密、ID 生成、分页、校验等工具
```

## 创建新模块的流程

### 方法 1：CLI 脚手架

```bash
go run . create -n <moduleName>              # 在 mod/<moduleName>/ 下生成
go run . create -n <moduleName> -p <path>    # 指定输出目录
go run . create -n <moduleName> -f           # 强制覆盖
```

脚手架原理（`cmd/create/createMod.go`）：读取 `mod/example/` 的 `embed.FS`，递归复制目录结构，将所有文件内容中的 `example` 替换为新模块名，跳过 `embed.go`。

### 方法 2：手动创建

对于简单模块（如纯服务客户端），可能只需要 `mod.go` 一个文件。

### 注册模块

在 `cmd/server/modList/list.go` 添加：

```go
var ModList = []kernel.Module{
    // ...existing
    &newMod.Mod{},
}
```

**模块顺序影响同阶段内的执行顺序。** 如果模块 A 的 PreInit Map 了某依赖，模块 B 的 PreInit 需要 Load 它，则 A 必须排在 B 前面。跨阶段（A 的 PreInit → B 的 Init）则不受顺序影响。

### 添加配置

在 `config.yaml` 和 `config.example.yaml` 中添加模块配置段：

```yaml
moduleName:
    field1: "value"
    field2: 0
```

也可以通过 `go run . config` 自动生成所有模块配置模板。

## DI 容器中已有的共享类型

**维护位置：** [`docs/di-reference.md`](docs/di-reference.md) — 共享类型表、Load 示例与 Map/Load 规则以该文档为准；新增基础设施 Map 时请同步更新。

业务模块优先查该表，在约定阶段 `hub.Load` 取用；若对 Map 时机或配置仍有疑问，可再阅读对应 `mod/<name>/mod.go`。

## 模块分层约定

业务模块的标准内部结构：

```
handler/ — HTTP 请求处理，解析输入、调用 service、格式化响应
service/ — 业务逻辑，编排 dao 调用、执行业务规则
dao/     — 数据访问层，GORM 查询，基于 stdao.Std[T]
model/   — 数据模型（GORM 结构体）和 DTO
e/       — 领域错误码定义
```

数据流向: `handler → service → dao → model`

在 `mod.go` 的 `Load()` 方法中自底向上组装：dao → service → handler → 注册路由。

## 常用工具包 (pkg/)

### stdao — 泛型 DAO

```go
type Std[T any] struct { db *gorm.DB }
// 提供: Init(db), Create(ctx, t), List(ctx), Update(ctx, t), Delete(ctx, t)
// 事务: Begin(), SetTxToCtx(ctx, tx), GetTxFromCtx(ctx) — 通过 context 传播事务
```

### settings — 动态设置

```go
item := settings.NewItem[int](settings.ItemConfig[int]{
    Key: "module.setting", DefaultValue: 3, Store: gormStore,
})
val := item.Get(ctx)   // 缓存 → DB → 默认值
item.Set(ctx, 5)       // 更新 DB + 缓存
```

### auth — JWT

```go
token, _ := auth.GenerateToken(uid, orgID, secret, expiry)
claims, _ := auth.ParseToken(tokenString, secret)
```

## 构建与部署

```bash
# 开发
go run . server -c config.yaml

# 构建（注入版本号）
go build -ldflags "-X github.com/juanjiTech/jframe/conf.SysVersion=v1.0.0" -o jframe .

# Docker
docker build -t jframe .
docker compose up -d

# 开发依赖 (Redis + MySQL)
docker compose -f docker-compose-dev.yml up -d
```

## Agent 开发注意事项

1. **新功能 = 新模块。** jframe 的一切功能都是模块。不要在 main.go 或 cmd/ 中直接写业务逻辑。
2. **通过 DI 通信。** 模块间不直接 import，通过 Hub 的 Map/Load 传递共享资源。
3. **生命周期选择很重要。** 参考已有模块选择正确的阶段。最常见的模式：基础设施在 PreInit，业务路由在 Load。
4. **Config 需要双 tag。** `yaml:"field" mapstructure:"field"` 缺一不可。
5. **Map 传指针。** `hub.Map(&x)` 不是 `hub.Map(x)`。Load 也是 `hub.Load(&x)`。
6. **Stop 必须 defer wg.Done()。** 否则关闭流程永远等待。
7. **jin 不是 gin。** HTTP 框架是 `github.com/juanjiTech/jin`（gin 的 fork），**去掉了 binding 包，改用 DI 注入**。Handler 的 `HandlerFunc` 是 `interface{}`，任意函数签名都行，参数通过 `inject.Invoke` 自动注入。`jin/middleware/binding` 提供 `binding.JSON(T{})` 和 `binding.Query(T{})` 中间件，将请求数据解析后 Map 到请求级 DI 容器，handler 直接作为函数参数接收。Query binding 使用 `query:"key"` struct tag。响应用 `c.Render(code, render.JSON{Data: data})`，没有 `c.JSON` 快捷方法。
8. **协议复用。** HTTP 和 gRPC 共享同一 TCP 端口，通过 cmux 区分。
9. **使用 hub.Log。** 不要用 `fmt.Println` 或裸 `zap.S()`，`hub.Log` 自带模块名命名空间。
10. **使用已有 skills。** 设计新模块用 `jframe-module-design`，实现模块代码用 `jframe-module-dev`。
