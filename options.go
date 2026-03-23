package cachez

import "time"

type Option[K comparable, V any] func(*cache[K, V])

func WithDefaultTTL[K comparable, V any](ttl time.Duration) Option[K, V] {
	return func(c *cache[K, V]) {
		if ttl >= 0 {
			c.defaultTTL = ttl
		}
	}
}

func WithNowFunc[K comparable, V any](fn func() time.Time) Option[K, V] {
	return func(c *cache[K, V]) {
		if fn != nil {
			c.now = fn
		}
	}
}

func WithHooks[K comparable, V any](hooks ...Hook[K, V]) Option[K, V] {
	return func(c *cache[K, V]) {
		c.hooks = append(c.hooks, hooks...)
	}
}
