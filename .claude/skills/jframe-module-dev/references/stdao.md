# stdao reference (Agent)

Import: `github.com/juanjiTech/jframe/pkg/stdao`  
Gold standard: `mod/example/dao/item.go`  
Human doc: `docs/stdao.md`

## Pattern

```go
type ItemDao struct {
    stdao.Std[*model.Item] // pointer type parameter required
}

func NewItemDao(db *gorm.DB) (*ItemDao, error) {
    d := &ItemDao{}
    if err := d.Init(db); err != nil { // AutoMigrate
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

## Rules

- Custom queries **must** use `GetTxFromCtx(ctx)` so they join ambient transactions.
- `Delete` returns `*gorm.DB` — check `.Error`.
- Prefer embedding `stdao.Model` (ULID + soft delete). Custom PK models are OK (see `pkg/settings`).
- Never `gorm.Open` inside a business module — `hub.Load(&db)`.

## Transactions

```go
tx := d.Begin()
ctx = stdao.SetTxToCtx(ctx, tx)
if err := d.Create(ctx, item); err != nil {
    tx.Rollback()
    return err
}
return tx.Commit().Error
```
