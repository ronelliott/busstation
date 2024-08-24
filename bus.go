package busstation

import "sync"

// busImpl is the default Bus implementation.
type busImpl[T any] struct {
	fanouts map[string]channelFanout[T]
	mutex   sync.Mutex
}

// NewBus creates a new bus instance with no registered passengers.
func NewBus[T any]() Bus[T] {
	return &busImpl[T]{
		fanouts: map[string]channelFanout[T]{},
	}
}

// Announce sends the given value to all handlers for the given event. The value
// is sent to the subscribers in a separate goroutine via a fanout channel. It
// returns true if there are subscribers for the given event, false otherwise.
func (bus *busImpl[T]) Announce(event string, data T) bool {
	bus.mutex.Lock()
	defer bus.mutex.Unlock()

	fanout, ok := bus.fanouts[event]
	if !ok {
		return false
	}

	fanout.Send(data)
	return true
}

// Depart removes the ticket from the bus, preventing further calls to the handler.
// It returns true if the ticket was removed, false otherwise.
func (bus *busImpl[T]) Depart(ticket *Ticket[T]) bool {
	bus.mutex.Lock()
	defer bus.mutex.Unlock()

	if ticket == nil || !ticket.IsValid() {
		return false
	}

	fanout, ok := bus.fanouts[ticket.event]
	if !ok {
		return false
	}

	fanout.Close(ticket.channel)
	ticket.wait.Wait()

	if fanout.Len() == 0 {
		delete(bus.fanouts, ticket.event)
	}

	ticket.invalidate()
	return true
}

// Embus adds the given handler to the bus for the given event. The handler will
// be called for each value sent to the bus for the given event. The handler
// will be called in a separate goroutine via a fanout channel. The ticket
// returned can be used to remove the handler from the bus.
func (bus *busImpl[T]) Embus(event string, handler Passenger[T]) *Ticket[T] {
	bus.mutex.Lock()
	defer bus.mutex.Unlock()

	fanout, ok := bus.fanouts[event]
	if !ok {
		fanout = newChannelFanout[T]()
		bus.fanouts[event] = fanout
	}

	channel := fanout.Create()
	ticket := newTicket[T](bus, channel, event)
	ticket.wait.Add(1)
	go bus.handler(channel, handler, &ticket.wait)
	return ticket
}

// handler is a helper method that calls the given handler for each value sent
// to the given channel.
func (bus *busImpl[T]) handler(channel <-chan T, handler Passenger[T], wait *sync.WaitGroup) {
	defer wait.Done()
	for value := range channel {
		handler(value)
	}
}
