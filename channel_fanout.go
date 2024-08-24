package busstation

import "sync"

// channelFanoutImpl is the default channelFanout implementation.
type channelFanoutImpl[T any] []chan<- T

// newChannelFanout creates a new channel fanout.
func newChannelFanout[T any]() channelFanout[T] {
	return &channelFanoutImpl[T]{}
}

// Add adds the given channels to the fanout. The channels are added to the
// fanout in the order they are given.
func (fanout *channelFanoutImpl[T]) Add(channels ...chan<- T) {
	*fanout = append(*fanout, channels...)
}

// Close is a convenience method that removes the given channels from the fanout
// and then closes it.
func (fanout *channelFanoutImpl[T]) Close(channels ...chan<- T) {
	for _, channel := range channels {
		fanout.Remove(channel)
		close(channel)
	}
}

// Create creates a new channel and adds it to the fanout. The new channel is
// returned.
func (fanout *channelFanoutImpl[T]) Create() chan T {
	channel := make(chan T)
	fanout.Add(channel)
	return channel
}

// Len returns the number of channels in the fanout.
func (fanout *channelFanoutImpl[T]) Len() int {
	return len(*fanout)
}

// remove removes the given channel from the fanout if it exists. After finding
// the channel it is removed by copying the elements after it one position to
// the left.
func (fanout *channelFanoutImpl[T]) remove(channel chan<- T) bool {
	for idx, other := range *fanout {
		if other == channel {
			*fanout = append((*fanout)[:idx], (*fanout)[idx+1:]...)
			return true
		}
	}

	return false
}

// Remove removes the given channels from the fanout if it exists. After finding
// a channel it is removed by copying the elements after it one position to the left.
func (fanout *channelFanoutImpl[T]) Remove(channels ...chan<- T) {
	for _, channel := range channels {
		fanout.remove(channel)
	}
}

// send sends the given value to all channels in the fanout. The value is
// sent to the channels in separate goroutines.
func (fanout *channelFanoutImpl[T]) send(value T) {
	wait := sync.WaitGroup{}
	for _, channel := range *fanout {
		wait.Add(1)
		go func(channel chan<- T) {
			defer wait.Done()
			channel <- value
		}(channel)
	}
	wait.Wait()
}

// Send sends the each of the given values to all channels in the fanout in
// sequence. The values are sent to the channels in separate goroutines.
func (fanout *channelFanoutImpl[T]) Send(values ...T) {
	for _, value := range values {
		fanout.send(value)
	}
}
