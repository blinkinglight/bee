package bee_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/blinkinglight/bee"
	"github.com/blinkinglight/bee/gen"
	"github.com/delaneyj/toolbelt/embeddednats"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

type UserEvent struct {
	Name    string `json:"name"`
	Country string `json:"country"`
}

type MockReplayHandler struct {
	Name    string
	Country string
}

func (m *MockReplayHandler) ApplyEvent(event *gen.EventEnvelope) error {
	var tmp UserEvent
	json.Unmarshal(event.GetPayload(), &tmp)
	switch event.EventType {
	case "user.created":
		m.Name = tmp.Name
		m.Country = tmp.Country
	case "user.updated":
		m.Name = tmp.Name
		m.Country = tmp.Country
	default:
		return nil // Ignore other event types
	}
	return nil
}

func TestCore(t *testing.T) {
	server, err := embeddednats.New(
		context.Background(),
		embeddednats.WithShouldClearData(true),
		embeddednats.WithDirectory("./tmp"),
		embeddednats.WithNATSServerOptions(&server.Options{
			JetStream: true,
		}),
	)
	if err != nil {
		t.Fatalf("Failed to start embedded NATS server: %v", err)
	}
	server.WaitForServer()

	defer server.Close()

	nc, err := server.Client()

	if err != nil {
		t.Fatalf("Failed to connect to NATS server: %v", err)
	}

	defer nc.Close()
	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("Failed to get JetStream context: %v", err)
	}
	js.AddStream(&nats.StreamConfig{
		Name:     "events",
		Subjects: []string{"events.>"},
	})

	replayHandler := &MockReplayHandler{}

	evt1 := &gen.EventEnvelope{
		EventType: "user.created",
		Payload:   []byte(`{"name": "John Doe", "country": "USA"}`),
	}
	b1, _ := proto.Marshal(evt1)
	_, err = js.Publish("events.user.123.created", b1)
	if err != nil {
		t.Fatalf("Failed to publish event: %v", err)
	}

	evt2 := &gen.EventEnvelope{
		EventType: "user.updated",
		Payload:   []byte(`{"name": "John Smith", "country": "Canada"}`),
	}
	b2, _ := proto.Marshal(evt2)
	_, err = js.Publish("events.user.123.updated", b2)
	if err != nil {
		t.Fatalf("Failed to publish event: %v", err)
	}

	bee.Replay(t.Context(), js, "user", "*", bee.DeliverAll, replayHandler)

	if replayHandler.Name != "John Smith" {
		t.Errorf("Expected name to be 'John Smith', got '%s'", replayHandler.Name)
	}
	if replayHandler.Country != "Canada" {
		t.Errorf("Expected country to be 'Canada', got '%s'", replayHandler.Country)
	}
}
