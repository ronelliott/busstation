# busstation

![Build Status](https://github.com/ronelliott/busstation/actions/workflows/master.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/ronelliott/busstation)](https://goreportcard.com/report/github.com/ronelliott/busstation)
[![Coverage Status](https://coveralls.io/repos/github/ronelliott/busstation/badge.svg?branch=master)](https://coveralls.io/github/ronelliott/busstation?branch=master)
[![Go Reference](https://pkg.go.dev/badge/github.com/ronelliott/busstation.svg)](https://pkg.go.dev/github.com/ronelliott/busstation)

busstation is a golang implementation of the event emitter pattern using channels and goroutines. To allow for concurrent designs, each subscriber is run in a separate goroutine, and messages are sent to the subscribers using channels.

## Usage

```go
bus := busstation.NewBus[string]()

ticket := bus.Embus("my-awesome-event", func(data string) {
	fmt.Println("got event!", data)
})

bus.Announce("my-awesome-event", "Hello World!")
ticket.Depart()
```
