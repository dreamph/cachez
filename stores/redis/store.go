package redis

import (
	"context"
	"fmt"
	"time"

	cachez "github.com/dreamph/cachez"
	rediscache "github.com/go-redis/cache/v9"
	goredis "github.com/redis/go-redis/v9"
)

type Option func(*config)

type config struct {
	prefix                string
	now                   func() time.Time
	scanCount             int64
	allowEmptyPrefixClear bool
}

type Store[K comparable, V any] struct {
	client                goredis.Cmdable
	redisCache            *rediscache.Cache
	prefix                string
	now                   func() time.Time
	scanCount             int64
	allowEmptyPrefixClear bool
}

func WithPrefix(prefix string) Option {
	return func(cfg *config) {
		cfg.prefix = prefix
	}
}

func WithNowFunc(now func() time.Time) Option {
	return func(cfg *config) {
		if now != nil {
			cfg.now = now
		}
	}
}

func WithScanCount(count int64) Option {
	return func(cfg *config) {
		if count > 0 {
			cfg.scanCount = count
		}
	}
}

func WithAllowEmptyPrefixClear() Option {
	return func(cfg *config) {
		cfg.allowEmptyPrefixClear = true
	}
}

func WithUnsafeAllowEmptyPrefixClear() Option {
	return WithAllowEmptyPrefixClear()
}

func NewStore[K comparable, V any](client goredis.Cmdable, opts ...Option) *Store[K, V] {
	cfg := config{
		prefix:                "cachez:",
		now:                   time.Now,
		scanCount:             200,
		allowEmptyPrefixClear: false,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	var rc *rediscache.Cache
	if client != nil {
		rc = rediscache.New(&rediscache.Options{Redis: client})
	}

	return &Store[K, V]{
		client:                client,
		redisCache:            rc,
		prefix:                cfg.prefix,
		now:                   cfg.now,
		scanCount:             cfg.scanCount,
		allowEmptyPrefixClear: cfg.allowEmptyPrefixClear,
	}
}

func (s *Store[K, V]) Get(ctx context.Context, key K) (cachez.Entry[V], bool, error) {
	if s.client == nil {
		var zero cachez.Entry[V]
		return zero, false, fmt.Errorf("redis client is nil")
	}

	var entry cachez.Entry[V]
	err := s.redisCache.Get(ctx, s.redisKey(key), &entry)
	if err == rediscache.ErrCacheMiss {
		var zero cachez.Entry[V]
		return zero, false, nil
	}
	if err != nil {
		var zero cachez.Entry[V]
		return zero, false, err
	}
	return entry, true, nil
}

func (s *Store[K, V]) Set(ctx context.Context, key K, entry cachez.Entry[V]) error {
	if s.client == nil {
		return fmt.Errorf("redis client is nil")
	}

	ttl := time.Duration(0)
	if !entry.Expiration.IsZero() {
		ttl = entry.Expiration.Sub(s.now())
		if ttl <= 0 {
			return s.client.Del(ctx, s.redisKey(key)).Err()
		}
	}

	return s.redisCache.Set(&rediscache.Item{
		Ctx:   ctx,
		Key:   s.redisKey(key),
		Value: entry,
		TTL:   ttl,
	})
}

func (s *Store[K, V]) Delete(ctx context.Context, key K) error {
	if s.client == nil {
		return fmt.Errorf("redis client is nil")
	}
	return s.client.Del(ctx, s.redisKey(key)).Err()
}

func (s *Store[K, V]) Has(ctx context.Context, key K) (bool, error) {
	if s.client == nil {
		return false, fmt.Errorf("redis client is nil")
	}

	n, err := s.client.Exists(ctx, s.redisKey(key)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *Store[K, V]) Clear(ctx context.Context) error {
	if s.client == nil {
		return fmt.Errorf("redis client is nil")
	}
	if s.prefix == "" && !s.allowEmptyPrefixClear {
		return fmt.Errorf("empty prefix clear is disabled; use WithUnsafeAllowEmptyPrefixClear (or WithAllowEmptyPrefixClear) to enable")
	}

	pattern := s.prefix + "*"
	if s.prefix == "" {
		pattern = "*"
	}

	var cursor uint64
	for {
		keys, next, err := s.client.Scan(ctx, cursor, pattern, s.scanCount).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := s.client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}

		cursor = next
		if cursor == 0 {
			break
		}
	}

	return nil
}

func (s *Store[K, V]) redisKey(key K) string {
	return s.prefix + fmt.Sprintf("%v", key)
}
