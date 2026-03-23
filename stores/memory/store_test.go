package memory_test

import (
	"context"
	"testing"
	"time"

	cachez "github.com/dreamph/cachez"
	memory "github.com/dreamph/cachez/stores/memory"
)

func TestStoreCRUDAndLen(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore[string, string]()

	if got := store.Len(); got != 0 {
		t.Fatalf("expected len=0, got %d", got)
	}

	if err := store.Set(ctx, "k1", cachez.Entry[string]{Value: "v1"}); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	entry, ok, err := store.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !ok || entry.Value != "v1" {
		t.Fatalf("expected hit value=v1, got ok=%v value=%s", ok, entry.Value)
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

	_, ok, err = store.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if ok {
		t.Fatal("expected miss after delete")
	}

	if err := store.Set(ctx, "k2", cachez.Entry[string]{Value: "v2"}); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	if err := store.Set(ctx, "k3", cachez.Entry[string]{Value: "v3"}); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	if got := store.Len(); got != 2 {
		t.Fatalf("expected len=2, got %d", got)
	}

	if err := store.Clear(ctx); err != nil {
		t.Fatalf("clear failed: %v", err)
	}

	if got := store.Len(); got != 0 {
		t.Fatalf("expected len=0 after clear, got %d", got)
	}
}

func TestStoreDeleteExpired(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore[string, string]()
	now := time.Now()

	if err := store.Set(ctx, "expired", cachez.Entry[string]{
		Value:      "v1",
		Expiration: now.Add(-time.Second),
	}); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	if err := store.Set(ctx, "valid", cachez.Entry[string]{
		Value:      "v2",
		Expiration: now.Add(time.Second),
	}); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	if err := store.Set(ctx, "forever", cachez.Entry[string]{
		Value:      "v3",
		Expiration: time.Time{},
	}); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	deleted := store.DeleteExpired(func() time.Time { return now })
	if deleted != 1 {
		t.Fatalf("expected deleted=1, got %d", deleted)
	}

	_, expiredOK, _ := store.Get(ctx, "expired")
	_, validOK, _ := store.Get(ctx, "valid")
	_, foreverOK, _ := store.Get(ctx, "forever")

	if expiredOK {
		t.Fatal("expected expired key to be removed")
	}
	if !validOK || !foreverOK {
		t.Fatalf("expected valid keys to remain, got valid=%v forever=%v", validOK, foreverOK)
	}
}

func TestStoreDeleteExpiredNilNowUsesStoreClock(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	store := memory.NewStore[string, string](
		memory.WithNowFunc[string, string](func() time.Time { return now }),
	)

	if err := store.Set(ctx, "expired", cachez.Entry[string]{
		Value:      "v1",
		Expiration: now.Add(-time.Second),
	}); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	if err := store.Set(ctx, "valid", cachez.Entry[string]{
		Value:      "v2",
		Expiration: now.Add(time.Second),
	}); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	deleted := store.DeleteExpired(nil)
	if deleted != 1 {
		t.Fatalf("expected deleted=1, got %d", deleted)
	}

	if got := store.Len(); got != 1 {
		t.Fatalf("expected len=1 after delete expired, got %d", got)
	}
}

func TestStoreWithNowFuncAndJanitor(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	now := time.Now()
	store := memory.NewStore[string, string](
		memory.WithNowFunc[string, string](func() time.Time { return now }),
		memory.WithJanitor[string, string](ctx, 5*time.Millisecond),
	)

	if err := store.Set(ctx, "k1", cachez.Entry[string]{
		Value:      "v1",
		Expiration: now.Add(10 * time.Millisecond),
	}); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	time.Sleep(12 * time.Millisecond)
	now = now.Add(20 * time.Millisecond)
	time.Sleep(15 * time.Millisecond)

	_, ok, err := store.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if ok {
		t.Fatal("expected janitor to delete expired key")
	}
}

func TestStoreWithJanitorInvalidConfigIgnored(t *testing.T) {
	store := memory.NewStore[string, string](
		memory.WithJanitor[string, string](nil, time.Second),
		memory.WithJanitor[string, string](context.Background(), 0),
		memory.WithNowFunc[string, string](nil),
	)
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}
