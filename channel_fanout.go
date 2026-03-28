package busstation

import "sync"

// fanoutEntry holds a subscriber channel alongside the coordination state
// needed to safely close it while sends may be in flight.
//
// Lifecycle of a close:
//  1. Close removes the entry from the fanout slice (under slice write lock).
//  2. Close signals done, waking any send goroutine blocked in the select.
//  3. Close calls inflight.Wait() — guaranteed to observe all Add calls made
//     before the slice write lock was acquired (see send).
//  4. Close calls close(ch) — safe because no goroutine is sending to ch.
type fanoutEntry[T any] struct {
	ch       chan<- T
	done     chan struct{}
	inflight sync.WaitGroup
}

// channelFanoutImpl is the default channelFanout implementation.
type channelFanoutImpl[T any] struct {
	mu      sync.RWMutex
	entries []*fanoutEntry[T]
}

// newChannelFanout creates a new channel fanout.
func newChannelFanout[T any]() channelFanout[T] {
	return &channelFanoutImpl[T]{}
}

// Add adds the given channels to the fanout.
func (fanout *channelFanoutImpl[T]) Add(channels ...chan<- T) {
	fanout.mu.Lock()
	defer fanout.mu.Unlock()
	for _, ch := range channels {
		fanout.entries = append(fanout.entries, &fanoutEntry[T]{
			ch:   ch,
			done: make(chan struct{}),
		})
	}
}

// Close removes channels found in the fanout, signals their done channel so
// any in-flight send goroutines can exit without blocking on the channel, waits
// for all in-flight sends to finish, then closes the channel. Channels not
// found in the fanout are ignored.
func (fanout *channelFanoutImpl[T]) Close(channels ...chan<- T) {
	var toClose []*fanoutEntry[T]

	fanout.mu.Lock()
	for _, ch := range channels {
		if e := fanout.findAndRemove(ch); e != nil {
			toClose = append(toClose, e)
		}
	}
	fanout.mu.Unlock()

	for _, e := range toClose {
		close(e.done)      // unblock any select goroutine waiting on this entry
		e.inflight.Wait() // wait for all in-flight sends to exit
		close(e.ch)       // safe: no goroutine is sending to ch
	}
}

// Create creates a new channel, adds it to the fanout, and returns it.
func (fanout *channelFanoutImpl[T]) Create() chan T {
	channel := make(chan T)
	fanout.Add(channel)
	return channel
}

// Len returns the number of channels in the fanout.
func (fanout *channelFanoutImpl[T]) Len() int {
	fanout.mu.RLock()
	defer fanout.mu.RUnlock()
	return len(fanout.entries)
}

// findAndRemove finds and removes the entry for the given channel from the
// entries slice. Returns nil if not found. Caller must hold mu.
func (fanout *channelFanoutImpl[T]) findAndRemove(ch chan<- T) *fanoutEntry[T] {
	for i, e := range fanout.entries {
		if e.ch == ch {
			fanout.entries = append(fanout.entries[:i], fanout.entries[i+1:]...)
			return e
		}
	}
	return nil
}

// Remove removes the given channels from the fanout without closing them.
func (fanout *channelFanoutImpl[T]) Remove(channels ...chan<- T) {
	fanout.mu.Lock()
	defer fanout.mu.Unlock()
	for _, ch := range channels {
		fanout.findAndRemove(ch)
	}
}

// send sends the given value to all current subscribers. The entries slice is
// snapshotted and each entry's inflight counter is incremented while holding
// the slice RLock. This guarantees that Close's inflight.Wait() — which can
// only start after the write lock is acquired (i.e., after all RLocks are
// released) — will observe every Add call made here.
//
// Each send goroutine uses a select so that a departing subscriber (done
// closed) causes the message to be dropped rather than blocking indefinitely.
func (fanout *channelFanoutImpl[T]) send(value T) {
	fanout.mu.RLock()
	entries := make([]*fanoutEntry[T], len(fanout.entries))
	copy(entries, fanout.entries)
	for _, e := range entries {
		e.inflight.Add(1)
	}
	fanout.mu.RUnlock()

	wait := sync.WaitGroup{}
	for _, entry := range entries {
		wait.Add(1)
		go func(e *fanoutEntry[T]) {
			defer wait.Done()
			defer e.inflight.Done()
			select {
			case e.ch <- value:
			case <-e.done:
			}
		}(entry)
	}
	wait.Wait()
}

// Send sends each of the given values to all channels in the fanout in sequence.
func (fanout *channelFanoutImpl[T]) Send(values ...T) {
	for _, value := range values {
		fanout.send(value)
	}
}
