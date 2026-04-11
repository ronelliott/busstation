package busstation

// Passenger is a handler for a bus event. It is called for each value sent to the
// bus for the given event the passenger is registered to receive.
type Passenger[T any] func(T)
