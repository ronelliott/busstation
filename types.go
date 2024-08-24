package busstation

import "sync"

// channelFanout is a fanout that sends values to a set of channels.
type channelFanout[T any] interface {
	// Add adds the given channels to the fanout.
	Add(...chan<- T)

	// Close is a convenience method that removes the given channels from the fanout
	// and then closes it.
	Close(...chan<- T)

	// Create creates a new channel and adds it to the fanout. The new channel is
	// returned.
	Create() chan T

	// Len returns the number of channels in the fanout.
	Len() int

	// Remove removes the given channels from the fanout.
	Remove(...chan<- T)

	// Send sends the each of the given values to all channels in the fanout.
	Send(...T)
}

// Bus is a message bus that allows handlers to be registered for events. When a
// value is sent to the bus for an event, the handlers registered for that event
// are called with the value.
type Bus[T any] interface {
	// Announce sends the given value to all handlers for the given event. The value
	// is sent to the subscribers in a separate goroutine via a fanout channel. It
	// returns true if there are subscribers for the given event, false otherwise.
	Announce(string, T) bool

	// Depart removes the ticket from the bus, preventing further calls to the handler.
	// It returns true if the ticket was removed, false otherwise.
	Depart(*Ticket[T]) bool

	// Embus adds the given handler to the bus for the given event. The handler will
	// be called for each value sent to the bus for the given event. The handler
	// will be called in a separate goroutine via a fanout channel. The ticket
	// returned can be used to remove the handler from the bus.
	Embus(string, Passenger[T]) *Ticket[T]
}

// Passenger is a handler for a bus event. It is called for each value sent to the
// bus for the given event the passenger is registered for.
type Passenger[T any] func(T)

// Ticket is a handle to a passenger registered with a bus. It can be used to
// remove the passenger from the bus it was created from.
type Ticket[T any] struct {
	bus     Bus[T]
	channel chan T
	event   string
	invalid bool
	wait    sync.WaitGroup
}
