# stdao — 数据访问

业务模块的 DAO 使用 **`github.com/juanjiTech/jframe/pkg/stdao`**。可运行金标准：[`mod/example/dao/item.go`](../mod/example/dao/item.go)。

## 最小用法

```go
package dao

import (
    "context"
    "github.com/juanjiTech/jframe/mod/<name>/model"
    "github.com/juanjiTech/jframe/pkg/stdao"
    "gorm.io/gorm"
)

type ItemDao struct {
    stdao.Std[*model.Item] // 类型参数必须是指针
}

func NewItemDao(db *gorm.DB) (*ItemDao, error) {
    d := &ItemDao{}
    if err := d.Init(db); err != nil { // AutoMigrate 本模型
        return nil, err
    }
    return d, nil
}

func (d *ItemDao) GetByID(ctx context.Context, id string) (*model.Item, error) {
    item := &model.Item{}
    err := d.GetTxFromCtx(ctx).WithContext(ctx).Where("id = ?", id).First(item).Error
    return item, err
}
```

在 `mod.go` 的 `Load()` 里：`hub.Load(&db)` → `dao.NewItemDao(db)` → 注入 service。

## 模型

推荐嵌入 `stdao.Model`（ULID 主键、时间戳、软删除）：

```go
type Item struct {
    stdao.Model
    Name string `json:"name" gorm:"size:128;not null"`
}
```

也可以不用 `stdao.Model`（自定义主键/复合键），仍可 `stdao.Std[*YourModel]` + `Init` 做 AutoMigrate。见 `pkg/settings` 的 `Model`（字符串主键、无软删除）。

## API 摘要

| 方法 | 作用 |
|------|------|
| `Init(db)` | 绑定 `*gorm.DB` 并 `AutoMigrate` |
| `Create/List/Update/Delete(ctx, …)` | 基础 CRUD（走当前 ctx 事务） |
| `GetTxFromCtx(ctx)` | 取事务或默认 DB；**自定义查询必须用它** |
| `Begin()` / `SetTxToCtx` | 开启事务并写入 context |

```go
tx := d.Begin()
ctx = stdao.SetTxToCtx(ctx, tx)
if err := d.Create(ctx, item); err != nil {
    tx.Rollback()
    return err
}
return tx.Commit().Error
```

## 约定

1. 一个模型一个 Dao 类型；`NewXxxDao` 内 `Init`。
2. 自定义查询一律 `GetTxFromCtx(ctx).WithContext(ctx)...`，以便事务传播。
3. `Delete` 返回 `*gorm.DB`，检查 `.Error`。
4. 不要在业务模块里再次 `gorm.Open`；从 Hub `Load` 已有 `*gorm.DB`。

## 延伸

- 分层与 `Load` 组装：[`module-implementation.md`](module-implementation.md)
- Agent 实现流程：`.claude/skills/jframe-module-dev/SKILL.md`
- 源码：`pkg/stdao/dao.go`、`pkg/stdao/model.go`
