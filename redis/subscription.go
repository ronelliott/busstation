package redis

import (
	"context"
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
func (s *subscription[T]) close(ctx context.Context) {
	close(s.done)
	s.pubsub.Close()
	s.wg.Wait()
	close(s.localCh)
}
