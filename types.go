package cachez

import (
	"context"
	"time"
)

type LoaderFunc[V any] func(ctx context.Context) (V, error)

type Cache[V any] interface {
	Get(ctx context.Context, key string) (V, bool, error)
	Set(ctx context.Context, key string, value V, ttl ...time.Duration) error
	Delete(ctx context.Context, key string) error
	Has(ctx context.Context, key string) (bool, error)
	Clear(ctx context.Context) error
	GetOrLoad(ctx context.Context, key string, ttl time.Duration, loader LoaderFunc[V]) (V, error)
}

type Store[V any] interface {
	Get(ctx context.Context, key string) (Entry[V], bool, error)
	Set(ctx context.Context, key string, entry Entry[V]) error
	Delete(ctx context.Context, key string) error
	Has(ctx context.Context, key string) (bool, error)
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
