package redis

import (
	"context"
	"sync"

	goredis "github.com/redis/go-redis/v9"
	"github.com/ronelliott/busstation"
)

// Option is a functional option for configuring a Redis-backed bus.
type Option[T any] func(*redisBusImpl[T])

// WithCodec overrides the default JSON codec. A nil value is ignored and the
// default JSONCodec is kept to avoid a later panic in Marshal/Unmarshal.
func WithCodec[T any](c Codec[T]) Option[T] {
	return func(b *redisBusImpl[T]) {
		if c == nil {
			return
		}
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
	old := b.subscriptions
	b.subscriptions = map[*busstation.Ticket[T]]*subscription[T]{}
	b.mu.Unlock()

	for ticket, s := range old {
		ticket.MarkDeparted()
		s.close()
	}

	return b.client.Close()
}

func (b *redisBusImpl[T]) Depart(ticket *busstation.Ticket[T]) bool {
	if ticket == nil {
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

	// Always close the subscription — it was removed from the map and must
	// not be orphaned regardless of whether this call wins the departed CAS.
	// sub.close is idempotent so a concurrent Close call is safe.
	departed := ticket.MarkDeparted()
	sub.close()
	return departed
}

// Embus subscribes to the given event and calls handler for each message.
// It blocks briefly to confirm the Redis subscription before returning.
// Returns nil if the subscription cannot be established (e.g. Redis is
// unreachable or the bus has been closed); the error is forwarded to the
// WithErrorHandler callback if one is set.
func (b *redisBusImpl[T]) Embus(event string, handler busstation.Passenger[T]) *busstation.Ticket[T] {
	pubsub := b.client.Subscribe(b.ctx, event)

	// Block until Redis acknowledges the subscription so that an Announce
	// immediately after Embus is guaranteed to be received.
	if _, err := pubsub.Receive(b.ctx); err != nil {
		if b.onErr != nil {
			b.onErr(err)
		}
		pubsub.Close()
		return nil
	}

	sub := newSubscription[T](pubsub, b.codec, b.onErr)
	ticket := busstation.NewTicket[T](b, sub.localCh, event)
	ticket.RunHandler(handler)

	b.mu.Lock()
	b.subscriptions[ticket] = sub
	b.mu.Unlock()

	return ticket
}
