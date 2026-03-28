package busstation

// NewTicket creates a new ticket for the given bus, channel, and event. This is
// intended for use by custom Bus implementations in external packages that need
// to construct tickets on behalf of their subscribers.
// Panics if bus or channel is nil.
func NewTicket[T any](bus Bus[T], channel chan T, event string) *Ticket[T] {
	if bus == nil {
		panic("busstation: NewTicket called with nil bus")
	}
	if channel == nil {
		panic("busstation: NewTicket called with nil channel")
	}
	return &Ticket[T]{
		bus:     bus,
		channel: channel,
		event:   event,
	}
}

// newTicket is the internal alias used within this package.
func newTicket[T any](bus Bus[T], channel chan T, event string) *Ticket[T] {
	return NewTicket(bus, channel, event)
}

// MarkDeparted atomically marks the ticket as departed. It returns true if this
// call was the one to mark it (i.e. it was not already departed), false otherwise.
// This is intended for use by custom Bus implementations that need to claim
// departure ownership before tearing down their subscription state.
func (ticket *Ticket[T]) MarkDeparted() bool {
	if ticket == nil {
		return false
	}
	return ticket.departed.CompareAndSwap(false, true)
}

// RunHandler starts a goroutine that reads from the ticket's channel and calls
// handler for each value. The goroutine is tracked by the ticket's WaitGroup so
// that Wait() returns only after it exits. This is intended for use by custom
// Bus implementations in external packages.
func (ticket *Ticket[T]) RunHandler(handler Passenger[T]) {
	if ticket == nil {
		return
	}
	if ticket.channel == nil {
		panic("busstation: RunHandler called on ticket with nil channel")
	}
	if handler == nil {
		panic("busstation: RunHandler called with nil handler")
	}
	ticket.wait.Add(1)
	go func() {
		defer ticket.wait.Done()
		for value := range ticket.channel {
			handler(value)
		}
	}()
}

// Depart removes the ticket from the bus it was created from, preventing further
// messages from being delivered to the handler. It returns true if the ticket
// was removed successfully, or false otherwise. Call Wait after Depart if you
// need a synchronous guarantee that the handler goroutine has fully exited.
func (ticket *Ticket[T]) Depart() bool {
	if ticket == nil || ticket.departed.Load() {
		return false
	}
	return ticket.bus.Depart(ticket)
}

// Wait blocks until the handler goroutine for this ticket has fully exited.
// It is safe to call from any goroutine, including concurrently with Depart,
// but must not be called from within the handler itself.
func (ticket *Ticket[T]) Wait() {
	if ticket == nil {
		return
	}
	ticket.wait.Wait()
}

// IsValid returns true if the ticket is valid, false otherwise. Tickets are
// considered valid if they have not yet departed and were created by a bus
// (i.e. bus, channel, and event are non-zero).
func (ticket *Ticket[T]) IsValid() bool {
	return !ticket.departed.Load() && ticket.bus != nil && ticket.channel != nil && ticket.event != ""
}
