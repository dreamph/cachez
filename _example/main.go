package main

import (
	"context"
	"fmt"
	"log"
	"time"

	cachez "github.com/dreamph/cachez"
	memory "github.com/dreamph/cachez/stores/memory"
)

type User struct {
	ID   int64
	Name string
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := memory.NewStore[User](
		memory.WithJanitor(ctx, 30*time.Second),
	)

	loggerHook := cachez.HookFunc[User](func(ctx context.Context, e cachez.HookEvent[User]) {
		log.Printf("cache event=%s key=%q err=%v", e.Type, e.Key, e.Err)
	})

	c := cachez.New[User](
		store,
		cachez.WithDefaultTTL(5*time.Minute),
		cachez.WithHooks(loggerHook),
	)

	err := c.Set(ctx, "user:1", User{
		ID:   1,
		Name: "Dream",
	})
	if err != nil {
		log.Fatal(err)
	}

	u, ok, err := c.Get(ctx, "user:1")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("get:", ok, u)

	u2, err := c.GetOrLoad(ctx, "user:2", 10*time.Minute, func(ctx context.Context) (User, error) {
		fmt.Println("loading user:2 from source...")
		return User{
			ID:   2,
			Name: "Alice",
		}, nil
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("get_or_load:", u2)

	u3, err := c.GetOrLoad(ctx, "user:2", 10*time.Minute, func(ctx context.Context) (User, error) {
		fmt.Println("this should not run because cache hit")
		return User{
			ID:   2,
			Name: "Bob",
		}, nil
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("get_or_load second:", u3)
}
