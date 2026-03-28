package redis_test

import (
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ronelliott/busstation"
	busredis "github.com/ronelliott/busstation/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestBus[T any](t *testing.T, opts ...busredis.Option[T]) (busstation.Bus[T], func()) {
	t.Helper()
	mr := miniredis.RunT(t)
	bus := busredis.NewBus[T](mr.Addr(), opts...)
	return bus, func() { require.NoError(t, bus.Close()) }
}

func TestAnnounceAndReceive(t *testing.T) {
	bus, cleanup := newTestBus[string](t)
	defer cleanup()

	var received []string
	var mu sync.Mutex
	done := make(chan struct{})

	ticket := bus.Embus("greet", func(v string) {
		mu.Lock()
		received = append(received, v)
		mu.Unlock()
		close(done)
	})
	defer ticket.Depart()

	ok := bus.Announce("greet", "hello")
	require.True(t, ok)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message")
	}

	mu.Lock()
	assert.Equal(t, []string{"hello"}, received)
	mu.Unlock()
}

func TestMultipleSubscribers(t *testing.T) {
	bus, cleanup := newTestBus[int](t)
	defer cleanup()

	const n = 3
	received := make([]int, n)
	var wg sync.WaitGroup
	wg.Add(n)

	tickets := make([]interface{ Depart() bool }, n)
	for i := 0; i < n; i++ {
		i := i
		tickets[i] = bus.Embus("count", func(v int) {
			received[i] = v
			wg.Done()
		})
	}
	defer func() {
		for _, t := range tickets {
			t.Depart()
		}
	}()

	bus.Announce("count", 42)

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for all subscribers")
	}

	for i, v := range received {
		assert.Equal(t, 42, v, "subscriber %d did not receive value", i)
	}
}

func TestDepartStopsDelivery(t *testing.T) {
	bus, cleanup := newTestBus[string](t)
	defer cleanup()

	first := make(chan struct{})
	extra := make(chan struct{}, 1)

	ticket := bus.Embus("msg", func(v string) {
		select {
		case <-first:
			// first message already seen — signal unexpected second delivery
			extra <- struct{}{}
		default:
			close(first)
		}
	})

	bus.Announce("msg", "one")
	select {
	case <-first:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first message")
	}

	ticket.Depart()
	ticket.Wait()

	bus.Announce("msg", "two")

	select {
	case <-extra:
		t.Fatal("handler called after Depart")
	case <-time.After(100 * time.Millisecond):
		// no second delivery — pass
	}
}

func TestDistributedFanout(t *testing.T) {
	// Two bus instances on the same Redis server — simulates separate processes.
	mr := miniredis.RunT(t)
	busA := busredis.NewBus[string](mr.Addr())
	busB := busredis.NewBus[string](mr.Addr())
	defer busA.Close()
	defer busB.Close()

	done := make(chan string, 1)
	ticket := busB.Embus("ping", func(v string) { done <- v })
	defer ticket.Depart()

	busA.Announce("ping", "cross-process")

	select {
	case got := <-done:
		assert.Equal(t, "cross-process", got)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for cross-bus message")
	}
}

func TestErrorHandler(t *testing.T) {
	var errSeen error
	var mu sync.Mutex
	errDone := make(chan struct{})

	mr := miniredis.RunT(t)

	// Bus[int] — will receive a JSON object it cannot decode as int.
	intBus := busredis.NewBus[int](mr.Addr(),
		busredis.WithErrorHandler[int](func(err error) {
			mu.Lock()
			errSeen = err
			mu.Unlock()
			select {
			case <-errDone:
			default:
				close(errDone)
			}
		}),
	)
	defer intBus.Close()

	ticket := intBus.Embus("bad", func(int) {})
	defer ticket.Depart()

	// Publish a JSON object from a Bus[struct] on the same server — int bus
	// will fail to unmarshal it and route the error to the handler.
	type weird struct{ X string }
	weirdBus := busredis.NewBus[weird](mr.Addr())
	defer weirdBus.Close()
	weirdBus.Announce("bad", weird{X: "not-an-int"})

	select {
	case <-errDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for error handler")
	}

	mu.Lock()
	assert.Error(t, errSeen)
	mu.Unlock()
}
