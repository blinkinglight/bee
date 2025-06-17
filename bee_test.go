package bee_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

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
		return nil
	}
	return nil
}

func client() (*nats.Conn, func(), error) {
	server, err := embeddednats.New(
		context.Background(),
		embeddednats.WithShouldClearData(true),
		embeddednats.WithDirectory("./tmp"),
		embeddednats.WithNATSServerOptions(&server.Options{
			JetStream:    true,
			NoLog:        false,
			Debug:        true,
			Trace:        true,
			TraceVerbose: true,
			Port:         4333,
		}),
	)
	if err != nil {
		return nil, nil, err
	}
	server.WaitForServer()

	nc, err := server.Client()

	return nc, func() {
		nc.Close()
		server.Close()
	}, err
}

func TestReplay(t *testing.T) {

	nc, cleanup, err := client()
	if err != nil {
		t.Fatalf("Failed to create NATS client: %v", err)
	}
	defer cleanup()

	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("Failed to get JetStream context: %v", err)
	}
	js.AddStream(&nats.StreamConfig{
		Name:     "events",
		Subjects: []string{"events.>"},
	})

	ctx := bee.WithNats(t.Context(), nc)
	ctx = bee.WithJetStream(ctx, js)

	evt1 := &gen.EventEnvelope{
		EventType:     "created",
		AggregateType: "users",
		AggregateId:   "123",
		Payload:       []byte(`{"name": "John Doe", "country": "USA"}`),
	}
	b1, _ := proto.Marshal(evt1)
	_, err = js.Publish("events.users.123.created", b1)
	if err != nil {
		t.Fatalf("Failed to publish event: %v", err)
	}

	evt2 := &gen.EventEnvelope{
		EventType:     "updated",
		AggregateType: "users",
		AggregateId:   "123",
		Payload:       []byte(`{"name": "John Smith", "country": "Canada"}`),
	}
	b2, _ := proto.Marshal(evt2)
	_, err = js.Publish("events.users.123.updated", b2)
	if err != nil {
		t.Fatalf("Failed to publish event: %v", err)
	}
	replayHandler := &MockReplayHandler{}
	bee.Replay(ctx, "users", "*", bee.DeliverAll, replayHandler)

	if replayHandler.Name != "John Smith" {
		t.Errorf("Expected name to be 'John Smith', got '%s'", replayHandler.Name)
	}
	if replayHandler.Country != "Canada" {
		t.Errorf("Expected country to be 'Canada', got '%s'", replayHandler.Country)
	}
}

func TestCommand(t *testing.T) {
	nc, cleanup, err := client()
	if err != nil {
		t.Fatalf("Failed to create NATS client: %v", err)
	}
	defer cleanup()

	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("Failed to get JetStream context: %v", err)
	}

	ctx := bee.WithNats(context.Background(), nc)
	ctx = bee.WithJetStream(ctx, js)
	go bee.NewCommandProcessor(ctx, "users", New(js))

	// service := New(js)
	// err = bee.Register(context.Background(), "users", service.Handle)
	cmd1 := &gen.CommandEnvelope{
		CommandType: "create",
		AggregateId: "123",
		Aggregate:   "users",
		Payload:     []byte(`{"name": "John Doe", "country": "USA"}`),
	}
	b1, _ := proto.Marshal(cmd1)
	_, err = js.Publish("cmds.users", b1)
	if err != nil {
		t.Fatalf("Failed to publish command: %v", err)
	}
	cmd2 := &gen.CommandEnvelope{
		CommandType: "update",
		AggregateId: "123",
		Aggregate:   "users",
		Payload:     []byte(`{"name": "John Doe", "country": "Canada"}`),
	}

	b2, _ := proto.Marshal(cmd2)
	_, err = js.Publish("cmds.users", b2)
	if err != nil {
		t.Fatalf("Failed to publish command: %v", err)
	}

	time.Sleep(500 * time.Millisecond) // Wait for command processing

	replayHandler := NewAggregate("123")
	bee.Replay(ctx, "users", "*", bee.DeliverAll, replayHandler)

	if replayHandler.Name != "John Doe" {
		t.Errorf("Expected name to be 'John Doe', got '%s'", replayHandler.Name)
	}
	if replayHandler.Country != "Canada" {
		t.Errorf("Expected country to be 'Canada', got '%s'", replayHandler.Country)
	}

}

func New(js nats.JetStreamContext) *UserServiceTest {
	return &UserServiceTest{
		js: js,
	}
}

type UserServiceTest struct {
	js nats.JetStreamContext
}

func (s UserServiceTest) Handle(ctx context.Context, m *gen.CommandEnvelope) ([]*gen.EventEnvelope, error) {
	agg := NewAggregate(m.AggregateId)
	bee.Replay(ctx, m.Aggregate, m.AggregateId, bee.DeliverAll, agg)
	return agg.ApplyCommand(ctx, m)
}

// --- UserAggregateTest implements ES aggregate logic ---
type UserAggregateTest struct {
	ID      string
	Name    string
	Country string
	Deleted bool
}

type User struct {
	Name    string `json:"name"`
	Country string `json:"country"`
}

func NewAggregate(id string) *UserAggregateTest {
	return &UserAggregateTest{ID: id}
}

func (u *UserAggregateTest) ApplyEvent(e *gen.EventEnvelope) error {
	// log.Printf("Applying event: %s for aggregate ID: %s", e.EventType, u.ID)
	switch e.EventType {
	case "created":
		data, _ := bee.Unmarshal[User](e.Payload)
		u.Name = data.Name
		u.Country = data.Country
		u.Deleted = false
	case "updated":
		data, _ := bee.Unmarshal[User](e.Payload)
		u.Country = data.Country
		u.Name = data.Name
	case "deleted":
		u.Deleted = true
	}
	return nil
}

func (u *UserAggregateTest) ApplyCommand(_ context.Context, c *gen.CommandEnvelope) ([]*gen.EventEnvelope, error) {
	if c.AggregateId != u.ID {
		return nil, fmt.Errorf("aggregate ID mismatch")
	}
	var event *gen.EventEnvelope = &gen.EventEnvelope{AggregateId: u.ID}
	event.AggregateType = "users"
	switch c.CommandType {
	case "create":
		event.EventType = "created"
		event.Payload = c.Payload
	case "update":
		if u.Deleted {
			return nil, fmt.Errorf("cannot update deleted user")
		}
		event.EventType = "updated"
		event.Payload = c.Payload
	case "delete":
		if u.Deleted {
			return nil, fmt.Errorf("user already deleted")
		}
		event.EventType = "deleted"
	default:
		return nil, fmt.Errorf("unknown command type: %s", c.CommandType)
	}
	return []*gen.EventEnvelope{event}, nil
}
