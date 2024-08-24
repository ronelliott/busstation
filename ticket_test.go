package busstation_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ronelliott/busstation"
)

func TestTicket_Depart(t *testing.T) {
	bus := busstation.NewBus[string]()

	callCount := 0
	ticket := bus.Embus("my-awesome-event", func(string) {
		callCount++
	})
	assert.True(t, bus.Announce("my-awesome-event", "Hello World!"),
		"Announce should return true when there are subscribers")

	time.Sleep(time.Millisecond * 100)

	assert.True(t, ticket.Depart(), "Depart should return true when the ticket is valid")
	assert.False(t, bus.Announce("my-awesome-event", "Hello World!"),
		"Announce should return false when there are no subscribers")

	time.Sleep(time.Millisecond * 100)

	assert.Equal(t, 1, callCount, "The handler should only be called once")
}

func TestTicket_IsValid(t *testing.T) {
	ticket := busstation.Ticket[string]{}
	assert.False(t, ticket.IsValid(), "The ticket should not be valid")
}
