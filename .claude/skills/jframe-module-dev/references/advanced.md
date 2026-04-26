# jframe Module Development — Advanced Reference

## Table of Contents
- DI Container Operations
- Working with gRPC Gateway
- Using the Settings System
- Testing Strategies

---

## DI Container Operations

The `Hub` (which embeds `inject.Injector`) provides these key operations:

### Map — register a dependency

```go
hub.Map(&value)  // Always pass a pointer
```

The type of the pointer is the key. Only one value per concrete type. If you need multiple values of the same type, wrap them in distinct struct types.

### Load — retrieve a dependency

```go
var value *SomeType
if err := hub.Load(&value); err != nil {
    return errors.New("can't load SomeType from kernel")
}
```

Always check the error. A missing dependency means a module that should Map it either isn't registered or runs in a later phase.

### Invoke — function injection

```go
hub.Invoke(func(db *gorm.DB, rdb *redis.Client) {
    // Both args auto-resolved from DI container
})
```

Useful for one-off operations. For repeated access, prefer Load into a local variable.

### Value — direct type lookup

```go
str := hub.Value(reflect.TypeOf("")).String()
```

Lower-level. Prefer Load for type safety.

---

## Working with gRPC Gateway

If your module needs gRPC endpoints:

```go
func (m *Mod) PostInit(hub *kernel.Hub) error {
    var gw *gateway.Gateway
    if err := hub.Load(&gw); err != nil {
        return errors.New("can't load gateway from kernel")
    }

    // Register your gRPC service with the gateway
    err := gw.Register(func(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
        return pb.RegisterYourServiceHandler(ctx, mux, conn)
    })
    return err
}
```

gRPC gateway routes are mounted at `/gapi/*` on the HTTP server. The gRPC server and HTTP server share the same TCP port via cmux.

---

## Using the Settings System

For dynamic, database-backed settings that can change at runtime:

```go
import "github.com/juanjiTech/jframe/pkg/settings"

// Define a setting
var maxRetries = settings.NewItem[int](settings.ItemConfig[int]{
    Key:          "myMod.maxRetries",
    DefaultValue: 3,
    Store:        store,  // *settings.GormStore from DB
    AfterUpdate: func(old, new int) {
        // React to setting changes
    },
})

// Read the current value (uses cache, falls back to DB, then default)
val := maxRetries.Get(ctx)

// Update
maxRetries.Set(ctx, 5)
```

Settings are cached with a TTL and can be overridden by environment variables.

---

## Testing Strategies

### Unit testing service logic

Mock the dao layer to test business rules in isolation:

```go
func TestCreateUser_DuplicateUsername(t *testing.T) {
    // Set up mock dao that returns an existing user
    // Call service.CreateUser
    // Assert ErrDuplicateUser is returned
}
```

### Integration testing with real DB

For tests that need a real database, set up a test database and use the same config loading:

```go
func TestUserDao_CRUD(t *testing.T) {
    db := setupTestDB(t)  // Connect to test MySQL/SQLite
    userDao, _ := dao.NewUserDao(db)

    ctx := context.Background()
    err := userDao.Create(ctx, &model.User{Username: "test"})
    require.NoError(t, err)

    users, err := userDao.List(ctx)
    require.NoError(t, err)
    require.Len(t, users, 1)
}
```

### HTTP handler testing

Test handlers by creating a jin.Engine, registering routes, and using `httptest`:

```go
func TestCreateUserHandler(t *testing.T) {
    engine := jin.New()
    // set up handler with mock service...
    engine.POST("/api/users", binding.JSON(model.CreateUserReq{}), handler.createUser)

    w := httptest.NewRecorder()
    req := httptest.NewRequest("POST", "/api/users",
        strings.NewReader(`{"username":"test","email":"test@example.com"}`))
    req.Header.Set("Content-Type", "application/json")

    engine.ServeHTTP(w, req)
    assert.Equal(t, http.StatusOK, w.Code)
}
```

### End-to-end verification

After implementing a module:

1. `go build ./...` — verify compilation
2. `go vet ./...` — static analysis
3. Start the server: `go run . server -c config.yaml`
4. Test endpoints: `curl http://localhost:8080/api/<name>/...`
5. Check logs for errors
6. Verify health check: `curl http://localhost:8080/health`
