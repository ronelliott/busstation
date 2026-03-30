package busstation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ronelliott/busstation"
)

func TestNewTicket_PanicOnNilBus(t *testing.T) {
	assert.Panics(t, func() {
		busstation.NewTicket[string](nil, make(chan string), "evt")
	})
}

func TestNewTicket_PanicOnNilChannel(t *testing.T) {
	bus := busstation.NewBus[string]()
	assert.Panics(t, func() {
		busstation.NewTicket[string](bus, nil, "evt")
	})
}

func TestNewTicket_HappyPath(t *testing.T) {
	bus := busstation.NewBus[string]()
	ch := make(chan string, 1)
	ticket := busstation.NewTicket[string](bus, ch, "evt")
	require.NotNil(t, ticket)
	assert.True(t, ticket.IsValid())
}

func TestRunHandler_WaitUnblocksAfterClose(t *testing.T) {
	bus := busstation.NewBus[string]()
	ch := make(chan string, 1)
	ticket := busstation.NewTicket[string](bus, ch, "evt")

	var received []string
	ticket.RunHandler(func(v string) {
		received = append(received, v)
	})

	ch <- "hello"
	close(ch)
	ticket.Wait()

	assert.Equal(t, []string{"hello"}, received)
}

func TestRunHandler_PanicOnNilHandler(t *testing.T) {
	bus := busstation.NewBus[string]()
	ch := make(chan string)
	ticket := busstation.NewTicket[string](bus, ch, "evt")
	assert.Panics(t, func() {
		ticket.RunHandler(nil)
	})
}

func TestRunHandler_PanicOnNilChannel(t *testing.T) {
	// Use a zero-value ticket which has a nil channel.
	ticket := &busstation.Ticket[string]{}
	assert.Panics(t, func() {
		ticket.RunHandler(func(string) {})
	})
}

func TestTicket_Depart(t *testing.T) {
	bus := busstation.NewBus[string]()

	callCount := 0
	ticket := bus.Embus("my-awesome-event", func(string) {
		callCount++
	})
	assert.True(t, bus.Announce("my-awesome-event", "Hello World!"),
		"Announce should return true when there are subscribers")
	assert.True(t, ticket.Depart(), "Depart should return true when the ticket is valid")
	ticket.Wait()
	assert.False(t, bus.Announce("my-awesome-event", "Hello World!"),
		"Announce should return false when there are no subscribers")
	assert.Equal(t, 1, callCount, "The handler should only be called once")
}

func TestTicket_IsValid(t *testing.T) {
	ticket := busstation.Ticket[string]{}
	assert.False(t, ticket.IsValid(), "The ticket should not be valid")
}
