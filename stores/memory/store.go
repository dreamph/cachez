package memory

import (
	"context"
	"sync"
	"time"

	cachez "github.com/dreamph/cachez"
)

type Option[K comparable, V any] func(*Store[K, V])

type Store[K comparable, V any] struct {
	mu             sync.RWMutex
	items          map[K]cachez.Entry[V]
	now            func() time.Time
	janitorCtx     context.Context
	janitorEnabled bool
	janitorEvery   time.Duration
}

func WithNowFunc[K comparable, V any](now func() time.Time) Option[K, V] {
	return func(s *Store[K, V]) {
		if now != nil {
			s.now = now
		}
	}
}

func WithJanitor[K comparable, V any](ctx context.Context, interval time.Duration) Option[K, V] {
	return func(s *Store[K, V]) {
		if ctx == nil || interval <= 0 {
			return
		}
		s.janitorCtx = ctx
		s.janitorEvery = interval
		s.janitorEnabled = true
	}
}

func NewStore[K comparable, V any](opts ...Option[K, V]) *Store[K, V] {
	s := &Store[K, V]{
		items: make(map[K]cachez.Entry[V]),
		now:   time.Now,
	}

	for _, opt := range opts {
		opt(s)
	}

	if s.janitorEnabled {
		go s.runJanitor()
	}

	return s
}

func (m *Store[K, V]) Get(_ context.Context, key K) (cachez.Entry[V], bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.items[key]
	return entry, ok, nil
}

func (m *Store[K, V]) Set(_ context.Context, key K, entry cachez.Entry[V]) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.items[key] = entry
	return nil
}

func (m *Store[K, V]) Delete(_ context.Context, key K) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.items, key)
	return nil
}

func (m *Store[K, V]) Has(_ context.Context, key K) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.items[key]
	return ok, nil
}

func (m *Store[K, V]) Clear(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.items = make(map[K]cachez.Entry[V])
	return nil
}

func (m *Store[K, V]) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.items)
}

func (m *Store[K, V]) DeleteExpired(nowFn func() time.Time) int {
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

func (m *Store[K, V]) runJanitor() {
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
