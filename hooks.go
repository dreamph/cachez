package cachez

import "context"

type EventType string

const (
	EventHit     EventType = "hit"
	EventMiss    EventType = "miss"
	EventSet     EventType = "set"
	EventDelete  EventType = "delete"
	EventClear   EventType = "clear"
	EventError   EventType = "error"
	EventLoad    EventType = "load"
	EventLoadHit EventType = "load_hit"
)

type HookEvent[K comparable, V any] struct {
	Type  EventType
	Key   K
	Value *V
	Err   error
}

type Hook[K comparable, V any] interface {
	OnEvent(ctx context.Context, event HookEvent[K, V])
}

type HookFunc[K comparable, V any] func(ctx context.Context, event HookEvent[K, V])

func (f HookFunc[K, V]) OnEvent(ctx context.Context, event HookEvent[K, V]) {
	f(ctx, event)
}
