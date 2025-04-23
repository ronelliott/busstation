package busstation

// Package busstation provides a simple message bus implementation for Go.
// It allows for the registration of handlers for events, and the sending of
// values to those handlers. The bus is designed to be used in a concurrent
// environment, and is safe for use from multiple goroutines. The bus uses
// a fanout strategy to send values to all handlers for a given event. The
// handlers are called in a separate goroutine, allowing for concurrent
// processing of events. The bus also provides a ticket system for tracking
// subscriptions to the bus. A ticket is created when a handler is registered
// with the bus, and can be used to remove the handler from the bus. The ticket
// is invalidated when the handler is removed, preventing further calls to the
// handler. The bus is designed to be simple and easy to use, with a focus on
// performance and concurrency.
