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
// calls to the handler. It returns true if the ticket was removed successfully,
// or false otherwise.
func (ticket *Ticket[T]) Depart() bool {
	return ticket.bus.Depart(ticket)
}

// IsValid returns true if the ticket is valid, false otherwise. Tickets are considered
// valid if they are not nil and the bus, channel, and event they are subscribed to
// are not nil or empty.
func (ticket *Ticket[T]) IsValid() bool {
	return ticket.bus != nil && ticket.channel != nil && ticket.event != ""
}

// invalidate invalidates the ticket by setting its bus and channel to nil and it's
// subscribed event to an empty string. This is used internally to mark the ticket
// as invalid after it has been removed from the bus.
// This is not intended to be used by the user.
func (ticket *Ticket[T]) invalidate() {
	ticket.bus = nil
	ticket.channel = nil
	ticket.event = ""
}
