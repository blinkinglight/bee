package bee

import (
	"encoding/json"

	"github.com/blinkinglight/bee/gen"
)

var (
	eventsRegistry   = make(map[string]map[string]func() any, 0)
	commandsRegistry = make(map[string]map[string]func() any, 0)
)

func RegisterEvent[T any](aggreate, event string) {
	if _, ok := eventsRegistry[aggreate]; !ok {
		eventsRegistry[aggreate] = make(map[string]func() any, 0)
	}
	eventsRegistry[aggreate][event] = func() any {
		return new(T)
	}
}

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

func UnmarshalEvent(e *gen.EventEnvelope) (any, error) {
	if e == nil || e.EventType == "" || e.AggregateType == "" {
		return nil, nil
	}

	event := GetEvent(e.AggregateType, e.EventType)
	if event == nil {
		return nil, nil
	}

	if err := json.Unmarshal(e.Payload, event); err != nil {
		return nil, err
	}

	return event, nil
}

func RegisterCommand[T any](aggreate, command string) {
	if _, ok := commandsRegistry[aggreate]; !ok {
		commandsRegistry[aggreate] = make(map[string]func() any, 0)
	}
	commandsRegistry[aggreate][command] = func() any {
		return new(T)
	}
}

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
