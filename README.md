# cachez

`cachez` is a lightweight generic cache library for Go.

It gives you:

- type-safe cache API (`Cache[K, V]`)
- TTL support
- in-memory store
- optional Redis store (separate module)
- `GetOrLoad` with `singleflight`
- hooks for cache events

## Install

```bash
go get github.com/dreamph/cachez@latest
```

## Quick Start (Memory)

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dreamph/cachez"
	memory "github.com/dreamph/cachez/stores/memory"
)

type User struct {
	ID   int64
	Name string
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := memory.NewStore[string, User](
		memory.WithJanitor[string, User](ctx, 30*time.Second),
	)

	loggerHook := cachez.HookFunc[string, User](func(ctx context.Context, e cachez.HookEvent[string, User]) {
		log.Printf("cache event=%s key=%q err=%v", e.Type, e.Key, e.Err)
	})

	c := cachez.New[string, User](
		store,
		cachez.WithDefaultTTL[string, User](5*time.Minute),
		cachez.WithHooks[string, User](loggerHook),
	)

	_ = c.Set(ctx, "user:1", User{ID: 1, Name: "Dream"})

	u, ok, _ := c.Get(ctx, "user:1")
	fmt.Println("get:", ok, u)

	u2, _ := c.GetOrLoad(ctx, "user:2", 10*time.Minute, func(ctx context.Context) (User, error) {
		fmt.Println("loading user:2 from source...")
		return User{ID: 2, Name: "Alice"}, nil
	})
	fmt.Println("get_or_load:", u2)
}
```

## Memory Store Options

- `memory.WithJanitor(ctx, interval)` starts internal cleanup loop for expired entries
- `memory.WithNowFunc(fn)` overrides current time source (useful for tests)

## Redis Store (Optional Module)

Redis support lives in a separate module, so projects that use only memory store do not pull Redis dependencies.

```bash
go get github.com/dreamph/cachez/stores/redis@latest
```

`stores/redis` is implemented using `github.com/go-redis/cache/v9`.

Safety note: `Clear()` is prefix-scoped. If prefix is empty, clear is blocked by default and requires `redisstore.WithUnsafeAllowEmptyPrefixClear()` (alias: `WithAllowEmptyPrefixClear()`).

```go
import (
	"context"
	"time"

	"github.com/dreamph/cachez"
	redisstore "github.com/dreamph/cachez/stores/redis"
	goredis "github.com/redis/go-redis/v9"
)

client := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})
store := redisstore.NewStore[string, User](
	client,
	redisstore.WithPrefix("cachez:user:"),
)

c := cachez.New[string, User](store, cachez.WithDefaultTTL[string, User](5*time.Minute))
_ = c.Set(context.Background(), "user:1", User{ID: 1, Name: "Dream"})
```

## Testing

Run core + memory tests:

```bash
go test ./...
```

Run Redis submodule tests:

```bash
cd stores/redis
go test ./...
```
