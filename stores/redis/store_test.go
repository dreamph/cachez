package redis_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	cachez "github.com/dreamph/cachez"
	redisstore "github.com/dreamph/cachez/stores/redis"
	goredis "github.com/redis/go-redis/v9"
)

func TestStoreSetGet(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, store := setupStringStore(t, "cachez:test:setget:")

	err := store.Set(ctx, "k1", cachez.Entry[string]{Value: "v1"})
	if err != nil {
		t.Fatalf("set failed: %v", err)
	}

	entry, ok, err := store.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !ok {
		t.Fatal("expected hit")
	}
	if entry.Value != "v1" {
		t.Fatalf("expected v1, got %s", entry.Value)
	}
}

func TestStoreGetMiss(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, store := setupStringStore(t, "cachez:test:miss:")

	_, ok, err := store.Get(ctx, "missing")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if ok {
		t.Fatal("expected miss")
	}
}

func TestStoreDeleteAndHas(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, store := setupStringStore(t, "cachez:test:delete:")

	if err := store.Set(ctx, "k1", cachez.Entry[string]{Value: "v1"}); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	has, err := store.Has(ctx, "k1")
	if err != nil {
		t.Fatalf("has failed: %v", err)
	}
	if !has {
		t.Fatal("expected has=true")
	}

	if err := store.Delete(ctx, "k1"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	has, err = store.Has(ctx, "k1")
	if err != nil {
		t.Fatalf("has failed: %v", err)
	}
	if has {
		t.Fatal("expected has=false after delete")
	}
}

func TestStoreClearPrefixScoped(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := setupClient(t)

	a := redisstore.NewStore[string](client, redisstore.WithPrefix("cachez:a:"))
	b := redisstore.NewStore[string](client, redisstore.WithPrefix("cachez:b:"))

	if err := a.Set(ctx, "1", cachez.Entry[string]{Value: "a1"}); err != nil {
		t.Fatalf("set a failed: %v", err)
	}
	if err := b.Set(ctx, "1", cachez.Entry[string]{Value: "b1"}); err != nil {
		t.Fatalf("set b failed: %v", err)
	}

	if err := a.Clear(ctx); err != nil {
		t.Fatalf("clear failed: %v", err)
	}

	_, ok, err := a.Get(ctx, "1")
	if err != nil {
		t.Fatalf("get a failed: %v", err)
	}
	if ok {
		t.Fatal("expected cleared key in prefix a")
	}

	entry, ok, err := b.Get(ctx, "1")
	if err != nil {
		t.Fatalf("get b failed: %v", err)
	}
	if !ok || entry.Value != "b1" {
		t.Fatalf("expected prefix b key to remain, got ok=%v value=%s", ok, entry.Value)
	}
}

func TestStoreSetPastExpirationDeletesKey(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, store := setupStringStore(t, "cachez:test:expiry:")

	if err := store.Set(ctx, "k1", cachez.Entry[string]{Value: "v1"}); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	if err := store.Set(ctx, "k1", cachez.Entry[string]{
		Value:      "v2",
		Expiration: time.Now().Add(-time.Second),
	}); err != nil {
		t.Fatalf("set with past expiration failed: %v", err)
	}

	_, ok, err := store.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if ok {
		t.Fatal("expected key to be deleted for past expiration")
	}
}

func TestStoreGetInvalidPayload(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := setupClient(t)
	store := redisstore.NewStore[string](client, redisstore.WithPrefix("cachez:test:invalid:"))

	if err := client.Set(ctx, "cachez:test:invalid:k1", "not-json", 0).Err(); err != nil {
		t.Fatalf("seed invalid payload failed: %v", err)
	}

	_, _, err := store.Get(ctx, "k1")
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
}

func TestStoreNilClientErrors(t *testing.T) {
	ctx := context.Background()
	store := redisstore.NewStore[string](nil)

	if _, _, err := store.Get(ctx, "k"); err == nil {
		t.Fatal("expected get error when client is nil")
	}
	if err := store.Set(ctx, "k", cachez.Entry[string]{Value: "v"}); err == nil {
		t.Fatal("expected set error when client is nil")
	}
	if err := store.Delete(ctx, "k"); err == nil {
		t.Fatal("expected delete error when client is nil")
	}
	if _, err := store.Has(ctx, "k"); err == nil {
		t.Fatal("expected has error when client is nil")
	}
	if err := store.Clear(ctx); err == nil {
		t.Fatal("expected clear error when client is nil")
	}
}

func TestStoreWithNowFuncAndScanCount(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, client := setupClientWithServer(t)
	base := time.Unix(1_700_000_000, 0)

	store := redisstore.NewStore[string](
		client,
		redisstore.WithPrefix("cachez:test:opts:"),
		redisstore.WithNowFunc(func() time.Time { return base }),
		redisstore.WithScanCount(1),
	)

	if err := store.Set(ctx, "k1", cachez.Entry[string]{
		Value:      "v1",
		Expiration: base.Add(5 * time.Second),
	}); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	ttl, err := client.TTL(ctx, "cachez:test:opts:k1").Result()
	if err != nil {
		t.Fatalf("ttl failed: %v", err)
	}
	if ttl <= 0 || ttl > 5*time.Second {
		t.Fatalf("expected ttl in (0,5s], got %v", ttl)
	}

	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("k%d", i+2)
		if err := store.Set(ctx, key, cachez.Entry[string]{Value: "x"}); err != nil {
			t.Fatalf("set failed: %v", err)
		}
	}

	if err := store.Clear(ctx); err != nil {
		t.Fatalf("clear failed: %v", err)
	}

	for i := 0; i < 6; i++ {
		key := fmt.Sprintf("k%d", i+1)
		_, ok, err := store.Get(ctx, key)
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}
		if ok {
			t.Fatalf("expected key %s to be cleared", key)
		}
	}
}

func TestStoreClearWithEmptyPrefixDisabledByDefault(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, client := setupClientWithServer(t)
	store := redisstore.NewStore[string](client, redisstore.WithPrefix(""))

	if err := client.Set(ctx, "external:key", "1", 0).Err(); err != nil {
		t.Fatalf("seed failed: %v", err)
	}
	if err := store.Set(ctx, "internal", cachez.Entry[string]{Value: "v"}); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	if err := store.Clear(ctx); err == nil {
		t.Fatal("expected clear error with empty prefix by default")
	}

	n, err := client.Exists(ctx, "external:key", "internal").Result()
	if err != nil {
		t.Fatalf("exists failed: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected keys to remain untouched, remaining=%d", n)
	}
}

func TestStoreClearWithEmptyPrefixAllowed(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, client := setupClientWithServer(t)
	store := redisstore.NewStore[string](
		client,
		redisstore.WithPrefix(""),
		redisstore.WithAllowEmptyPrefixClear(),
	)

	if err := client.Set(ctx, "external:key", "1", 0).Err(); err != nil {
		t.Fatalf("seed failed: %v", err)
	}
	if err := store.Set(ctx, "internal", cachez.Entry[string]{Value: "v"}); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	if err := store.Clear(ctx); err != nil {
		t.Fatalf("clear failed: %v", err)
	}

	n, err := client.Exists(ctx, "external:key", "internal").Result()
	if err != nil {
		t.Fatalf("exists failed: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected all keys cleared with explicit allow, remaining=%d", n)
	}
}

func setupStringStore(t *testing.T, prefix string) (*goredis.Client, *redisstore.Store[string]) {
	t.Helper()

	client := setupClient(t)
	store := redisstore.NewStore[string](client, redisstore.WithPrefix(prefix))
	return client, store
}

func setupClient(t *testing.T) *goredis.Client {
	t.Helper()
	_, client := setupClientWithServer(t)
	return client
}

func setupClientWithServer(t *testing.T) (*miniredis.Miniredis, *goredis.Client) {
	t.Helper()

	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
		mr.Close()
	})
	return mr, client
}
