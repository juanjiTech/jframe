# Advanced reference — DI, gRPC, settings, testing

Read this only when the task needs these topics. Core CRUD: see parent `SKILL.md` + `stdao.md` + `jin-handlers.md`.

## DI operations

```go
hub.Map(&value)   // always pointer
var v *T
hub.Load(&v)      // always check error
hub.Invoke(func(db *gorm.DB) { ... })
```

One value per concrete type; wrap duplicates (`type ReadDB struct{ *gorm.DB }`).  
Type table: `docs/di-reference.md`.

## gRPC gateway

In `PostInit`/`Load`:

```go
var gw *gateway.Gateway
if err := hub.Load(&gw); err != nil {
    return err
}
return gw.Register(func(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
    return pb.RegisterYourServiceHandler(ctx, mux, conn)
})
```

Routes under `/gapi/*`. HTTP and gRPC share one port via cmux.

## Settings

```go
import "github.com/juanjiTech/jframe/pkg/settings"

var maxRetries = settings.NewItem[int](settings.ItemConfig[int]{
    Key: "myMod.maxRetries", DefaultValue: 3, Store: store,
})
val := maxRetries.Get(ctx)
_ = maxRetries.Set(ctx, 5)
```

## Testing

- **Service:** mock dao; assert domain errors.
- **DAO:** real test DB; `dao.NewXxxDao(db)` then CRUD.
- **Handler:** `jin.New()`, register routes, `httptest` + `ServeHTTP`.
- **E2E:** `go build ./...` → run server → curl.
