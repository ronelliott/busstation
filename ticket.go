package busstation

// newTicket creates a new ticket instance for the given bus, channel and subscribed event.
// It is used internally to create a ticket when a handler is registered to a bus.
// This is not intended to be used by the user.
func newTicket[T any](bus Bus[T], channel chan T, event string) *Ticket[T] {
	return &Ticket[T]{
		bus:     bus,
		channel: channel,
		event:   event,
	}
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
