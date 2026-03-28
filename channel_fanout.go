package busstation

import "sync"

// channelFanoutImpl is the default channelFanout implementation.
type channelFanoutImpl[T any] struct {
	mu       sync.RWMutex
	channels []chan<- T
}

// newChannelFanout creates a new channel fanout.
func newChannelFanout[T any]() channelFanout[T] {
	return &channelFanoutImpl[T]{}
}

// Add adds the given channels to the fanout.
func (fanout *channelFanoutImpl[T]) Add(channels ...chan<- T) {
	fanout.mu.Lock()
	defer fanout.mu.Unlock()
	fanout.channels = append(fanout.channels, channels...)
}

// Close removes channels that exist in the fanout and closes them. Channels not
// found in the fanout are ignored, preventing accidental closure of channels
// owned by other fanouts.
func (fanout *channelFanoutImpl[T]) Close(channels ...chan<- T) {
	fanout.mu.Lock()
	defer fanout.mu.Unlock()
	for _, channel := range channels {
		if fanout.remove(channel) {
			close(channel)
		}
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
	return len(fanout.channels)
}

// remove removes a single channel from the fanout slice. Caller must hold mu.
func (fanout *channelFanoutImpl[T]) remove(channel chan<- T) bool {
	for idx, other := range fanout.channels {
		if other == channel {
			fanout.channels = append(fanout.channels[:idx], fanout.channels[idx+1:]...)
			return true
		}
	}
	return false
}

// Remove removes the given channels from the fanout.
func (fanout *channelFanoutImpl[T]) Remove(channels ...chan<- T) {
	fanout.mu.Lock()
	defer fanout.mu.Unlock()
	for _, channel := range channels {
		fanout.remove(channel)
	}
}

// send sends the given value to all channels in the fanout. An RLock is held
// for the entire duration — including waiting for all sends to complete — so
// that Close cannot close a channel while a send to it is in flight.
func (fanout *channelFanoutImpl[T]) send(value T) {
	fanout.mu.RLock()
	defer fanout.mu.RUnlock()
	wait := sync.WaitGroup{}
	for _, channel := range fanout.channels {
		wait.Add(1)
		go func(channel chan<- T) {
			defer wait.Done()
			channel <- value
		}(channel)
	}
	wait.Wait()
}

// Send sends each of the given values to all channels in the fanout in sequence.
func (fanout *channelFanoutImpl[T]) Send(values ...T) {
	for _, value := range values {
		fanout.send(value)
	}
}
