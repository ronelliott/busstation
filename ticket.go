package busstation

// newTicket creates a new ticket instance for the given bus, channel and event.
func newTicket[T any](bus Bus[T], channel chan T, event string) *Ticket[T] {
	return &Ticket[T]{
		bus:     bus,
		channel: channel,
		event:   event,
	}
}

// Depart removes the ticket from the bus it was created from, preventing further
// calls to the handler. It returns true if the ticket was removed, false otherwise.
func (ticket *Ticket[T]) Depart() bool {
	return ticket.bus.Depart(ticket)
}

// IsValid returns true if the ticket is valid, false otherwise.
func (ticket *Ticket[T]) IsValid() bool {
	return ticket.bus != nil && ticket.channel != nil && ticket.event != ""
}

// invalidate invalidates the ticket.
func (ticket *Ticket[T]) invalidate() {
	ticket.bus = nil
	ticket.channel = nil
	ticket.event = ""
}
