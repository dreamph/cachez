package cachez

import (
	"context"
	"time"
)

type LoaderFunc[V any] func(ctx context.Context) (V, error)

type Cache[K comparable, V any] interface {
	Get(ctx context.Context, key K) (V, bool, error)
	Set(ctx context.Context, key K, value V, ttl ...time.Duration) error
	Delete(ctx context.Context, key K) error
	Has(ctx context.Context, key K) (bool, error)
	Clear(ctx context.Context) error
	GetOrLoad(ctx context.Context, key K, ttl time.Duration, loader LoaderFunc[V]) (V, error)
}

type Store[K comparable, V any] interface {
	Get(ctx context.Context, key K) (Entry[V], bool, error)
	Set(ctx context.Context, key K, entry Entry[V]) error
	Delete(ctx context.Context, key K) error
	Has(ctx context.Context, key K) (bool, error)
	Clear(ctx context.Context) error
}

type Entry[V any] struct {
	Value      V
	Expiration time.Time
}

func (e Entry[V]) Expired(now time.Time) bool {
	if e.Expiration.IsZero() {
		return false
	}
	return now.After(e.Expiration)
}
