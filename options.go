package cachez

import (
	"fmt"
	"time"
)

type Option interface {
	apply(target any)
}

type optionFunc func(target any)

func (f optionFunc) apply(target any) {
	f(target)
}

type hooksOption[V any] struct {
	hooks []Hook[V]
}

func (o hooksOption[V]) apply(target any) {
	c, ok := target.(*cache[V])
	if !ok {
		panic(fmt.Sprintf("cachez: hook option type mismatch: %T", target))
	}
	c.hooks = append(c.hooks, o.hooks...)
}

func WithDefaultTTL(ttl time.Duration) Option {
	return optionFunc(func(target any) {
		if ttl < 0 {
			return
		}
		c, ok := target.(interface{ setDefaultTTL(time.Duration) })
		if !ok {
			return
		}
		c.setDefaultTTL(ttl)
	})
}

func WithNowFunc(fn func() time.Time) Option {
	return optionFunc(func(target any) {
		if fn == nil {
			return
		}
		c, ok := target.(interface{ setNowFunc(func() time.Time) })
		if !ok {
			return
		}
		c.setNowFunc(fn)
	})
}

func WithHooks[V any](hooks ...Hook[V]) Option {
	return hooksOption[V]{hooks: hooks}
}
