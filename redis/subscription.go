package redis

import (
	"sync"

	goredis "github.com/redis/go-redis/v9"
)

// subscription bridges a Redis PubSub channel to a local chan T, feeding
// deserialized messages into it until the subscription is closed.
type subscription[T any] struct {
	pubsub  *goredis.PubSub
	localCh chan T
	codec   Codec[T]
	onErr   func(error)
	done    chan struct{}
	once    sync.Once
	wg      sync.WaitGroup
}

func newSubscription[T any](pubsub *goredis.PubSub, codec Codec[T], onErr func(error)) *subscription[T] {
	s := &subscription[T]{
		pubsub:  pubsub,
		localCh: make(chan T),
		codec:   codec,
		onErr:   onErr,
		done:    make(chan struct{}),
	}
	s.wg.Add(1)
	go s.run()
	return s
}

func (s *subscription[T]) run() {
	defer s.wg.Done()
	// Always close localCh when run exits so that ranging goroutines (e.g.
	// RunHandler) unblock even on a connection drop where close() is never
	// called. close() uses sync.Once and no longer closes localCh itself, so
	// there is no double-close risk.
	defer close(s.localCh)
	msgCh := s.pubsub.Channel()
	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				return
			}
			val, err := s.codec.Unmarshal([]byte(msg.Payload))
			if err != nil {
				if s.onErr != nil {
					s.onErr(err)
				}
				continue
			}
			select {
			case s.localCh <- val:
			case <-s.done:
				return
			}
		case <-s.done:
			return
		}
	}
}

// close stops the bridge goroutine and unsubscribes from Redis.
// It is safe to call more than once; subsequent calls are no-ops.
// localCh is closed by run() when it exits, not here, so there is no
// double-close risk regardless of whether the connection dropped first.
func (s *subscription[T]) close() {
	s.once.Do(func() {
		close(s.done)
		s.pubsub.Close()
		s.wg.Wait()
	})
}
