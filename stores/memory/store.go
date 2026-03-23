package memory

import (
	"context"
	"sync"
	"time"

	cachez "github.com/dreamph/cachez"
)

type Option func(*config)

type config struct {
	now            func() time.Time
	janitorCtx     context.Context
	janitorEnabled bool
	janitorEvery   time.Duration
}

type Store[V any] struct {
	mu             sync.RWMutex
	items          map[string]cachez.Entry[V]
	now            func() time.Time
	janitorCtx     context.Context
	janitorEnabled bool
	janitorEvery   time.Duration
}

func WithNowFunc(now func() time.Time) Option {
	return func(cfg *config) {
		if now != nil {
			cfg.now = now
		}
	}
}

func WithJanitor(ctx context.Context, interval time.Duration) Option {
	return func(cfg *config) {
		if ctx == nil || interval <= 0 {
			return
		}
		cfg.janitorCtx = ctx
		cfg.janitorEvery = interval
		cfg.janitorEnabled = true
	}
}

func NewStore[V any](opts ...Option) *Store[V] {
	cfg := config{
		now: time.Now,
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(&cfg)
	}

	s := &Store[V]{
		items:          make(map[string]cachez.Entry[V]),
		now:            cfg.now,
		janitorCtx:     cfg.janitorCtx,
		janitorEnabled: cfg.janitorEnabled,
		janitorEvery:   cfg.janitorEvery,
	}

	if s.janitorEnabled {
		go s.runJanitor()
	}

	return s
}

func (m *Store[V]) Get(_ context.Context, key string) (cachez.Entry[V], bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.items[key]
	return entry, ok, nil
}

func (m *Store[V]) Set(_ context.Context, key string, entry cachez.Entry[V]) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.items[key] = entry
	return nil
}

func (m *Store[V]) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.items, key)
	return nil
}

func (m *Store[V]) Has(_ context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.items[key]
	return ok, nil
}

func (m *Store[V]) Clear(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.items = make(map[string]cachez.Entry[V])
	return nil
}

func (m *Store[V]) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.items)
}

func (m *Store[V]) DeleteExpired(nowFn func() time.Time) int {
	if nowFn == nil {
		nowFn = m.now
	}
	if nowFn == nil {
		nowFn = time.Now
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	currentTime := nowFn()
	deleted := 0

	for k, entry := range m.items {
		if entry.Expired(currentTime) {
			delete(m.items, k)
			deleted++
		}
	}
	return deleted
}

func (m *Store[V]) runJanitor() {
	ticker := time.NewTicker(m.janitorEvery)
	defer ticker.Stop()

	for {
		select {
		case <-m.janitorCtx.Done():
			return
		case <-ticker.C:
			m.DeleteExpired(m.now)
		}
	}
}
