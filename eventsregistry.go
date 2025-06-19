package bee

import (
	"encoding/json"
	"fmt"

	"github.com/blinkinglight/bee/gen"
)

var (
	eventsRegistry   = make(map[string]map[string]func() any, 0)
	commandsRegistry = make(map[string]map[string]func() any, 0)
)

// RegisterEvent registers an event type for a specific aggregate.
// The type T should be a struct that represents the event.
// It creates a function that returns a new instance of T when called.
// This allows for dynamic creation of event instances based on the aggregate and event type.
func RegisterEvent[T any](aggreate, event string) {
	if _, ok := eventsRegistry[aggreate]; !ok {
		eventsRegistry[aggreate] = make(map[string]func() any, 0)
	}
	eventsRegistry[aggreate][event] = func() any {
		return new(T)
	}
}

// GetEvent retrieves an event instance based on the aggregate type and event type.
// It checks if the aggregate and event are registered, and if so, it calls the corresponding
// function to create a new instance of the event type.
// If the aggregate or event is not registered, it returns nil.
// This function is useful for dynamically handling events in a type-safe manner.
func GetEvent(aggreate, event string) any {
	if aggreate == "" || event == "" {
		return nil
	}

	if aggreateEvents, ok := eventsRegistry[aggreate]; ok {
		if eventFn, ok := aggreateEvents[event]; ok {
			return eventFn()
		}
	}

	return nil
}

// UnmarshalEvent unmarshals a JSON payload into an event instance.
// It takes an EventEnvelope, checks if it has valid event type and aggregate type,
// and retrieves the corresponding event instance using GetEvent.
// If the event type or aggregate type is not registered, it returns nil.
// If the event instance is found, it unmarshals the JSON payload into that instance.
// If unmarshaling fails, it returns an error.
// This function is useful for converting raw event data into structured event types.
func UnmarshalEvent(e *gen.EventEnvelope) (any, error) {
	if e == nil || e.EventType == "" || e.AggregateType == "" {
		return nil, fmt.Errorf("invalid event envelope: %v %v", e.EventType, e.AggregateType)
	}

	event := GetEvent(e.AggregateType, e.EventType)
	if event == nil {
		return nil, fmt.Errorf("event type %s for aggregate %s not registered", e.EventType, e.AggregateType)
	}

	if err := json.Unmarshal(e.Payload, event); err != nil {
		return nil, err
	}

	return event, nil
}

// RegisterCommand registers a command type for a specific aggregate.
// The type T should be a struct that represents the command.
// It creates a function that returns a new instance of T when called.
// This allows for dynamic creation of command instances based on the aggregate and command type.
func RegisterCommand[T any](aggreate, command string) {
	if _, ok := commandsRegistry[aggreate]; !ok {
		commandsRegistry[aggreate] = make(map[string]func() any, 0)
	}
	commandsRegistry[aggreate][command] = func() any {
		return new(T)
	}
}

// GetCommand retrieves a command instance based on the aggregate type and command type.
// It checks if the aggregate and command are registered, and if so, it calls the corresponding
// function to create a new instance of the command type.
// If the aggregate or command is not registered, it returns nil.
// This function is useful for dynamically handling commands in a type-safe manner.
func GetCommand(aggreate, command string) any {
	if aggreate == "" || command == "" {
		return nil
	}

	if aggreateCommands, ok := commandsRegistry[aggreate]; ok {
		if commandFn, ok := aggreateCommands[command]; ok {
			return commandFn()
		}
	}

	return nil
}

// UnmarshalCommand unmarshals a JSON payload into a command instance.
// It takes a CommandEnvelope, checks if it has valid command type and aggregate type,
// and retrieves the corresponding command instance using GetCommand.
// If the command type or aggregate type is not registered, it returns nil.
// If the command instance is found, it unmarshals the JSON payload into that instance.
// If unmarshaling fails, it returns an error.
// This function is useful for converting raw command data into structured command types.
func UnmarshalCommand(c *gen.CommandEnvelope) (any, error) {
	if c == nil || c.CommandType == "" || c.Aggregate == "" {
		return nil, nil
	}

	command := GetCommand(c.Aggregate, c.CommandType)
	if command == nil {
		return nil, nil
	}

	if err := json.Unmarshal(c.Payload, command); err != nil {
		return nil, err
	}

	return command, nil
}
