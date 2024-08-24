package busstation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ronelliott/busstation"
)

func TestNewBus(t *testing.T) {
	bus := busstation.NewBus[string]()
	assert.NotNil(t, bus)
}

func TestBus_Basic(t *testing.T) {
	bus := busstation.NewBus[string]()

	testValue1 := "test-value-1"
	testValue2 := "test-value-2"

	handler1CallCount := 0
	handler2CallCount := 0
	handler3CallCount := 0

	ticket1 := bus.Embus("test1", func(data string) {
		handler1CallCount++
		assert.Equal(t, testValue1, data, "handler1 should receive the emitted value")
	})

	ticket2 := bus.Embus("test1", func(data string) {
		handler2CallCount++
		assert.Equal(t, testValue1, data, "handler2 should receive the emitted value")
	})

	ticket3 := bus.Embus("test2", func(data string) {
		handler3CallCount++
		assert.Equal(t, testValue2, data, "handler3 should receive the emitted value")
	})

	assert.True(t, bus.Announce("test1", testValue1), "Announce should return true when there are subscribers")
	assert.True(t, bus.Announce("test1", testValue1), "Announce should return true when there are subscribers")
	assert.True(t, bus.Depart(ticket2), "Depart should return true when the ticket is valid")
	assert.True(t, bus.Announce("test1", testValue1), "Announce should return true when there are subscribers")
	assert.True(t, bus.Depart(ticket1), "Depart should return true when the ticket is valid")
	assert.False(t, bus.Announce("test1", testValue1), "Announce should return false when there are no subscribers")

	assert.True(t, bus.Announce("test2", testValue2), "Announce should return true when there are subscribers")
	assert.True(t, bus.Announce("test2", testValue2), "Announce should return true when there are subscribers")
	assert.True(t, bus.Announce("test2", testValue2), "Announce should return true when there are subscribers")
	assert.True(t, bus.Announce("test2", testValue2), "Announce should return true when there are subscribers")
	assert.True(t, bus.Depart(ticket3), "Depart should return true when the ticket is valid")
	assert.False(t, bus.Announce("test2", testValue2), "Announce should return false when there are no subscribers")

	assert.Equal(t, 3, handler1CallCount, "handler1 should be called 3 times")
	assert.Equal(t, 2, handler2CallCount, "handler2 should be called 2 times")
	assert.Equal(t, 4, handler3CallCount, "handler3 should be called 4 times")
}

func TestBus_Announce_Empty(t *testing.T) {
	bus := busstation.NewBus[string]()
	assert.False(t, bus.Announce("test", "test-value"), "Announce should return false when there are no subscribers")

	ticket := bus.Embus("test", func(data string) {})
	assert.True(t, bus.Announce("test", "test-value"), "Announce should return true when there are subscribers")

	assert.True(t, bus.Depart(ticket), "Depart should return true when the ticket is valid")
	assert.False(t, bus.Announce("test", "test-value"), "Announce should return false when there are no subscribers")
}

func TestBus_Depart_Empty(t *testing.T) {
	bus := busstation.NewBus[string]()
	assert.False(t, bus.Depart(&busstation.Ticket[string]{}), "Depart should return false when the ticket is invalid")
}

func TestBus_Depart_Invalid(t *testing.T) {
	bus1 := busstation.NewBus[string]()
	bus2 := busstation.NewBus[string]()
	assert.False(t, bus1.Depart(&busstation.Ticket[string]{}), "Depart should return false when the ticket is invalid")

	ticket := bus2.Embus("test", func(data string) {})
	assert.False(t, bus1.Depart(ticket), "Depart should return false when the ticket is from another bus")
}
