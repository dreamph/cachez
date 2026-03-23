package cachez_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	cachez "github.com/dreamph/cachez"
	memory "github.com/dreamph/cachez/stores/memory"
)

func TestCacheBasic(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore[string]()
	c := cachez.New[string](store, cachez.WithDefaultTTL(time.Minute))

	if err := c.Set(ctx, "hello", "world"); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	v, ok, err := c.Get(ctx, "hello")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !ok {
		t.Fatal("expected hit")
	}
	if v != "world" {
		t.Fatalf("expected world, got %s", v)
	}
}

func TestCacheExpiration(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	store := memory.NewStore[string]()
	c := cachez.New[string](
		store,
		cachez.WithNowFunc(func() time.Time { return now }),
	)

	if err := c.Set(ctx, "k1", "v1", time.Second); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	now = now.Add(2 * time.Second)

	_, ok, err := c.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if ok {
		t.Fatal("expected miss after expiration")
	}
}

func TestGetOrLoadSingleflight(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore[string]()
	c := cachez.New[string](store)

	var called int32
	loader := func(ctx context.Context) (string, error) {
		atomic.AddInt32(&called, 1)
		time.Sleep(50 * time.Millisecond)
		return "value", nil
	}

	const workers = 10
	var wg sync.WaitGroup
	errCh := make(chan error, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err := c.GetOrLoad(ctx, "same-key", time.Minute, loader)
			if err != nil {
				errCh <- err
				return
			}
			if v != "value" {
				errCh <- fmt.Errorf("unexpected value: %s", v)
				return
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}

	if atomic.LoadInt32(&called) != 1 {
		t.Fatalf("expected loader called once, got %d", called)
	}
}

func TestGetOrLoadNilLoader(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore[string]()
	c := cachez.New[string](store)

	_, err := c.GetOrLoad(ctx, "k", time.Minute, nil)
	if err == nil {
		t.Fatal("expected error when loader is nil")
	}
}

func TestGetOrLoadLoaderError(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore[string]()
	c := cachez.New[string](store)

	loaderErr := fmt.Errorf("load failed")
	loader := func(context.Context) (string, error) {
		return "", loaderErr
	}

	_, err := c.GetOrLoad(ctx, "k", time.Minute, loader)
	if err == nil {
		t.Fatal("expected loader error")
	}

	_, ok, getErr := c.Get(ctx, "k")
	if getErr != nil {
		t.Fatalf("get failed: %v", getErr)
	}
	if ok {
		t.Fatal("expected key to stay missing when loader fails")
	}
}

func TestCacheDeleteHasClear(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore[string]()
	c := cachez.New[string](store)

	if err := c.Set(ctx, "k1", "v1"); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	has, err := c.Has(ctx, "k1")
	if err != nil {
		t.Fatalf("has failed: %v", err)
	}
	if !has {
		t.Fatal("expected has=true")
	}

	if err := c.Delete(ctx, "k1"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	has, err = c.Has(ctx, "k1")
	if err != nil {
		t.Fatalf("has failed: %v", err)
	}
	if has {
		t.Fatal("expected has=false after delete")
	}

	if err := c.Set(ctx, "k2", "v2"); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	if err := c.Set(ctx, "k3", "v3"); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	if err := c.Clear(ctx); err != nil {
		t.Fatalf("clear failed: %v", err)
	}

	hasK2, err := c.Has(ctx, "k2")
	if err != nil {
		t.Fatalf("has failed: %v", err)
	}
	hasK3, err := c.Has(ctx, "k3")
	if err != nil {
		t.Fatalf("has failed: %v", err)
	}
	if hasK2 || hasK3 {
		t.Fatalf("expected all keys cleared, got k2=%v k3=%v", hasK2, hasK3)
	}
}

func TestCacheHasExpiredNoSideEffects(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	store := memory.NewStore[string]()

	var hookCalls int32
	h := cachez.HookFunc[string](func(context.Context, cachez.HookEvent[string]) {
		atomic.AddInt32(&hookCalls, 1)
	})

	c := cachez.New[string](
		store,
		cachez.WithNowFunc(func() time.Time { return now }),
		cachez.WithHooks(h),
	)

	if err := store.Set(ctx, "expired", cachez.Entry[string]{
		Value:      "v",
		Expiration: now.Add(-time.Second),
	}); err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	has, err := c.Has(ctx, "expired")
	if err != nil {
		t.Fatalf("has failed: %v", err)
	}
	if has {
		t.Fatal("expected has=false for expired entry")
	}
	if got := atomic.LoadInt32(&hookCalls); got != 0 {
		t.Fatalf("expected has to not emit hooks, got %d calls", got)
	}
	if got := store.Len(); got != 1 {
		t.Fatalf("expected has to avoid deleting key, got len=%d", got)
	}
}

func TestCacheGetExpiredDeleteError(t *testing.T) {
	ctx := context.Background()
	deleteErr := errors.New("delete failed")
	store := &expiredDeleteErrorStore{deleteErr: deleteErr}

	var errorHooks int32
	h := cachez.HookFunc[string](func(_ context.Context, e cachez.HookEvent[string]) {
		if e.Type == cachez.EventError && errors.Is(e.Err, deleteErr) {
			atomic.AddInt32(&errorHooks, 1)
		}
	})

	c := cachez.New[string](store, cachez.WithHooks(h))

	_, ok, err := c.Get(ctx, "k1")
	if err == nil {
		t.Fatal("expected get error when deleting expired entry fails")
	}
	if ok {
		t.Fatal("expected miss when deleting expired entry fails")
	}
	if !errors.Is(err, deleteErr) {
		t.Fatalf("expected delete error, got %v", err)
	}
	if got := atomic.LoadInt32(&errorHooks); got != 1 {
		t.Fatalf("expected 1 error hook, got %d", got)
	}
}

func TestHooks(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore[string]()

	var hitCount int32
	var missCount int32

	h := cachez.HookFunc[string](func(ctx context.Context, e cachez.HookEvent[string]) {
		switch e.Type {
		case cachez.EventHit:
			atomic.AddInt32(&hitCount, 1)
		case cachez.EventMiss:
			atomic.AddInt32(&missCount, 1)
		}
	})

	c := cachez.New[string](store, cachez.WithHooks(h))

	if _, ok, err := c.Get(ctx, "missing"); err != nil {
		t.Fatal(err)
	} else if ok {
		t.Fatal("expected miss")
	}

	if err := c.Set(ctx, "a", "b"); err != nil {
		t.Fatal(err)
	}

	if _, ok, err := c.Get(ctx, "a"); err != nil {
		t.Fatal(err)
	} else if !ok {
		t.Fatal("expected hit")
	}

	if atomic.LoadInt32(&missCount) != 1 {
		t.Fatalf("expected 1 miss, got %d", missCount)
	}
	if atomic.LoadInt32(&hitCount) != 1 {
		t.Fatalf("expected 1 hit, got %d", hitCount)
	}
}

func TestJanitorDeletesExpired(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	now := time.Now()
	store := memory.NewStore[string](
		memory.WithNowFunc(func() time.Time { return now }),
		memory.WithJanitor(ctx, 5*time.Millisecond),
	)
	_ = store.Set(ctx, "k", cachez.Entry[string]{
		Value:      "v",
		Expiration: now.Add(10 * time.Millisecond),
	})

	time.Sleep(12 * time.Millisecond)
	now = now.Add(20 * time.Millisecond)
	time.Sleep(15 * time.Millisecond)

	if got := store.Len(); got != 0 {
		t.Fatalf("expected store to be empty, got len=%d", got)
	}
}

type expiredDeleteErrorStore struct {
	deleteErr error
}

func (s *expiredDeleteErrorStore) Get(context.Context, string) (cachez.Entry[string], bool, error) {
	return cachez.Entry[string]{
		Value:      "v",
		Expiration: time.Unix(0, 0),
	}, true, nil
}

func (s *expiredDeleteErrorStore) Set(context.Context, string, cachez.Entry[string]) error {
	return nil
}

func (s *expiredDeleteErrorStore) Delete(context.Context, string) error {
	return s.deleteErr
}

func (s *expiredDeleteErrorStore) Has(context.Context, string) (bool, error) {
	return true, nil
}

func (s *expiredDeleteErrorStore) Clear(context.Context) error {
	return nil
}
