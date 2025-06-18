package bee

import (
	"encoding/json"

	"github.com/blinkinglight/bee/gen"
)

var (
	registry = make(map[string]map[string]func() any, 0)
)

func Register[T any](aggreate, event string) {
	if _, ok := registry[aggreate]; !ok {
		registry[aggreate] = make(map[string]func() any, 0)
	}
	registry[aggreate][event] = func() any {
		return new(T)
	}
}

func GetEvent(aggreate, event string) any {
	if aggreate == "" || event == "" {
		return nil
	}

	if aggreateEvents, ok := registry[aggreate]; ok {
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
