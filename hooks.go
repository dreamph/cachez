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

type HookEvent[V any] struct {
	Type  EventType
	Key   string
	Value *V
	Err   error
}

type Hook[V any] interface {
	OnEvent(ctx context.Context, event HookEvent[V])
}

type HookFunc[V any] func(ctx context.Context, event HookEvent[V])

func (f HookFunc[V]) OnEvent(ctx context.Context, event HookEvent[V]) {
	f(ctx, event)
}
