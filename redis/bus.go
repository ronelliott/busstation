package redis

import (
	"context"
	"sync"

	goredis "github.com/redis/go-redis/v9"
	"github.com/ronelliott/busstation"
)

// Option is a functional option for configuring a Redis-backed bus.
type Option[T any] func(*redisBusImpl[T])

// WithCodec overrides the default JSON codec.
func WithCodec[T any](c Codec[T]) Option[T] {
	return func(b *redisBusImpl[T]) {
		b.codec = c
	}
}

// WithErrorHandler sets a callback for errors that occur during message
// deserialization. By default, malformed messages are silently dropped.
func WithErrorHandler[T any](fn func(error)) Option[T] {
	return func(b *redisBusImpl[T]) {
		b.onErr = fn
	}
}

type redisBusImpl[T any] struct {
	client        *goredis.Client
	codec         Codec[T]
	onErr         func(error)
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.Mutex
	subscriptions map[*busstation.Ticket[T]]*subscription[T]
}

// NewBus creates a Redis-backed Bus[T].
//
// Announce publishes to a Redis channel; all processes subscribed to that
// channel receive the message. The return value of Announce reflects publish
// success, not local subscriber count.
//
// Close must be called to release the Redis connection when the bus is no
// longer needed.
func NewBus[T any](addr string, opts ...Option[T]) busstation.Bus[T] {
	ctx, cancel := context.WithCancel(context.Background())
	b := &redisBusImpl[T]{
		client: goredis.NewClient(&goredis.Options{Addr: addr}),
		codec:  JSONCodec[T]{},
		ctx:    ctx,
		cancel: cancel,
		subscriptions: map[*busstation.Ticket[T]]*subscription[T]{},
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

func (b *redisBusImpl[T]) Announce(event string, data T) bool {
	payload, err := b.codec.Marshal(data)
	if err != nil {
		return false
	}
	return b.client.Publish(b.ctx, event, payload).Err() == nil
}

func (b *redisBusImpl[T]) Close() error {
	b.cancel()

	b.mu.Lock()
	subs := make([]*subscription[T], 0, len(b.subscriptions))
	for _, s := range b.subscriptions {
		subs = append(subs, s)
	}
	b.mu.Unlock()

	for _, s := range subs {
		s.close(b.ctx)
	}

	return b.client.Close()
}

func (b *redisBusImpl[T]) Depart(ticket *busstation.Ticket[T]) bool {
	if ticket == nil || !ticket.MarkDeparted() {
		return false
	}

	b.mu.Lock()
	sub, ok := b.subscriptions[ticket]
	if ok {
		delete(b.subscriptions, ticket)
	}
	b.mu.Unlock()

	if !ok {
		return false
	}

	sub.close(b.ctx)
	return true
}

func (b *redisBusImpl[T]) Embus(event string, handler busstation.Passenger[T]) *busstation.Ticket[T] {
	pubsub := b.client.Subscribe(b.ctx, event)
	sub := newSubscription[T](pubsub, b.codec, b.onErr)
	ticket := busstation.NewTicket[T](b, sub.localCh, event)
	ticket.RunHandler(handler)

	b.mu.Lock()
	b.subscriptions[ticket] = sub
	b.mu.Unlock()

	return ticket
}
