package cachez

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/singleflight"
)

type cache[K comparable, V any] struct {
	store       Store[K, V]
	defaultTTL  time.Duration
	now         func() time.Time
	hooks       []Hook[K, V]
	flightGroup singleflight.Group
}

func New[K comparable, V any](store Store[K, V], opts ...Option[K, V]) Cache[K, V] {
	c := &cache[K, V]{
		store:      store,
		defaultTTL: 5 * time.Minute,
		now:        time.Now,
		hooks:      nil,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (c *cache[K, V]) Get(ctx context.Context, key K) (V, bool, error) {
	entry, ok, err := c.store.Get(ctx, key)
	if err != nil {
		var zero V
		c.emit(ctx, HookEvent[K, V]{Type: EventError, Key: key, Err: err})
		return zero, false, err
	}

	if !ok {
		var zero V
		c.emit(ctx, HookEvent[K, V]{Type: EventMiss, Key: key})
		return zero, false, nil
	}

	if entry.Expired(c.now()) {
		if err := c.store.Delete(ctx, key); err != nil {
			var zero V
			c.emit(ctx, HookEvent[K, V]{Type: EventError, Key: key, Err: err})
			return zero, false, err
		}
		var zero V
		c.emit(ctx, HookEvent[K, V]{Type: EventMiss, Key: key})
		return zero, false, nil
	}

	v := entry.Value
	c.emit(ctx, HookEvent[K, V]{Type: EventHit, Key: key, Value: &v})
	return v, true, nil
}

func (c *cache[K, V]) Set(ctx context.Context, key K, value V, ttl ...time.Duration) error {
	effectiveTTL := c.defaultTTL
	if len(ttl) > 0 {
		effectiveTTL = ttl[0]
	}

	var expiration time.Time
	switch {
	case effectiveTTL > 0:
		expiration = c.now().Add(effectiveTTL)
	case effectiveTTL == 0:
		expiration = time.Time{}
	case effectiveTTL < 0:
		expiration = time.Time{}
	}

	entry := Entry[V]{
		Value:      value,
		Expiration: expiration,
	}

	if err := c.store.Set(ctx, key, entry); err != nil {
		c.emit(ctx, HookEvent[K, V]{Type: EventError, Key: key, Err: err})
		return err
	}

	c.emit(ctx, HookEvent[K, V]{Type: EventSet, Key: key, Value: &value})
	return nil
}

func (c *cache[K, V]) Delete(ctx context.Context, key K) error {
	if err := c.store.Delete(ctx, key); err != nil {
		c.emit(ctx, HookEvent[K, V]{Type: EventError, Key: key, Err: err})
		return err
	}

	c.emit(ctx, HookEvent[K, V]{Type: EventDelete, Key: key})
	return nil
}

func (c *cache[K, V]) Has(ctx context.Context, key K) (bool, error) {
	entry, ok, err := c.store.Get(ctx, key)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	if entry.Expired(c.now()) {
		return false, nil
	}
	return true, nil
}

func (c *cache[K, V]) Clear(ctx context.Context) error {
	if err := c.store.Clear(ctx); err != nil {
		var zeroKey K
		c.emit(ctx, HookEvent[K, V]{Type: EventError, Key: zeroKey, Err: err})
		return err
	}

	var zeroKey K
	c.emit(ctx, HookEvent[K, V]{Type: EventClear, Key: zeroKey})
	return nil
}

func (c *cache[K, V]) GetOrLoad(ctx context.Context, key K, ttl time.Duration, loader LoaderFunc[V]) (V, error) {
	if loader == nil {
		var zero V
		return zero, fmt.Errorf("loader is nil")
	}

	if v, ok, err := c.Get(ctx, key); err != nil {
		var zero V
		return zero, err
	} else if ok {
		return v, nil
	}

	flightKey := fmt.Sprintf("%v", key)

	result, err, _ := c.flightGroup.Do(flightKey, func() (any, error) {
		if v, ok, err := c.Get(ctx, key); err != nil {
			var zero V
			return zero, err
		} else if ok {
			c.emit(ctx, HookEvent[K, V]{Type: EventLoadHit, Key: key})
			return v, nil
		}

		v, err := loader(ctx)
		if err != nil {
			c.emit(ctx, HookEvent[K, V]{Type: EventError, Key: key, Err: err})
			var zero V
			return zero, err
		}

		if err := c.Set(ctx, key, v, ttl); err != nil {
			var zero V
			return zero, err
		}

		c.emit(ctx, HookEvent[K, V]{Type: EventLoad, Key: key, Value: &v})
		return v, nil
	})
	if err != nil {
		var zero V
		return zero, err
	}

	v, ok := result.(V)
	if !ok {
		var zero V
		return zero, fmt.Errorf("invalid singleflight result type")
	}
	return v, nil
}

func (c *cache[K, V]) emit(ctx context.Context, event HookEvent[K, V]) {
	for _, hook := range c.hooks {
		if hook == nil {
			continue
		}
		hook.OnEvent(ctx, event)
	}
}
