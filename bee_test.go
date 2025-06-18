package bee_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/blinkinglight/bee"
	"github.com/blinkinglight/bee/co"
	"github.com/blinkinglight/bee/gen"
	"github.com/blinkinglight/bee/ro"
	"github.com/delaneyj/toolbelt/embeddednats"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

func init() {
	bee.RegisterEvent[UserCreatedEvent]("users", "created")
	bee.RegisterEvent[UserUpdatedEvent]("users", "updated")
	bee.RegisterEvent[UserDeletedEvent]("users", "deleted")

	bee.RegisterEvent[TicketCreatedEvent]("tickets", "created")

	bee.RegisterCommand[CreateUserCommand]("users", "create")
	bee.RegisterCommand[UpdateUserCommand]("users", "update")
	bee.RegisterCommand[DeleteUserCommand]("users", "delete")
}

type UserCreatedEvent struct {
	Name    string `json:"name"`
	Country string `json:"country"`
}

type UserUpdatedEvent struct {
	Name    string `json:"name"`
	Country string `json:"country"`
}
type UserDeletedEvent struct {
}

type CreateUserCommand struct {
	Name    string `json:"name"`
	Country string `json:"country"`
}

type UpdateUserCommand struct {
	Name    string `json:"name"`
	Country string `json:"country"`
}
type DeleteUserCommand struct {
}

type MockReplayHandler struct {
	Name    string
	Country string
}

func (m *MockReplayHandler) ApplyEvent(e *gen.EventEnvelope) error {
	event, err := bee.UnmarshalEvent(e)
	if err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}
	switch event := event.(type) {
	case *UserCreatedEvent:
		m.Name = event.Name
		m.Country = event.Country
	case *UserUpdatedEvent:
		m.Name = event.Name
		m.Country = event.Country
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
	bee.Replay(ctx, replayHandler, ro.WithAggreate("users"), ro.WithAggregateID("*"))

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
	go bee.Command(ctx, New(js), co.WithAggreate("users"))

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
	bee.Replay(ctx, replayHandler, ro.WithAggreate("users"), ro.WithAggregateID("*"))

	if replayHandler.Name != "John Doe" {
		t.Errorf("Expected name to be 'John Doe', got '%s'", replayHandler.Name)
	}
	if replayHandler.Country != "Canada" {
		t.Errorf("Expected country to be 'Canada', got '%s'", replayHandler.Country)
	}

	evt := &gen.EventEnvelope{
		EventType:     "created",
		AggregateType: "tickets",
		AggregateId:   "1",
		Payload:       []byte(`{"name": "Project 1", "description": "This is a test project"}`),
		Parents:       []*gen.ParentRef{{AggregateType: "projects", AggregateId: "1"}},
	}
	b, _ := proto.Marshal(evt)
	js.Publish("events.projects.1.tickets.1.created", b)

	var projectAgg = &TicketsAggregate{}
	bee.Replay(ctx, projectAgg, ro.WithAggreate("tickets"), ro.WithAggregateID("1"), ro.WithParent("projects", "1"))
	if len(projectAgg.List) == 0 {
		t.Errorf("Expected tickets list to contain at least one item, got %v", projectAgg.List)
	}
	if projectAgg.List[0] != "Project 1" {
		t.Errorf("Expected tickets list to contain 'Project 1', got %v", projectAgg.List)
	}
}

type TicketCreatedEvent struct {
	Name string `json:"name"`
}

type TicketsAggregate struct {
	List []string
}

func (t *TicketsAggregate) ApplyEvent(e *gen.EventEnvelope) error {
	event, err := bee.UnmarshalEvent(e)
	if err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}
	switch event := event.(type) {
	case *TicketCreatedEvent:
		t.List = append(t.List, event.Name)
	}
	return nil
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
	bee.Replay(ctx, agg, ro.WithAggreate(m.Aggregate), ro.WithAggregateID(m.AggregateId))
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
	event, err := bee.UnmarshalEvent(e)
	if err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}
	switch event := event.(type) {
	case *UserCreatedEvent:
		u.Name = event.Name
		u.Country = event.Country
		u.Deleted = false
	case *UserUpdatedEvent:
		u.Country = event.Country
		u.Name = event.Name
	case *UserDeletedEvent:
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

	command, err := bee.UnmarshalCommand(c)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal command: %w", err)
	}

	switch command.(type) {
	case *CreateUserCommand:
		event.EventType = "created"
		event.Payload = c.Payload
	case *UpdateUserCommand:
		if u.Deleted {
			return nil, fmt.Errorf("cannot update deleted user")
		}
		event.EventType = "updated"
		event.Payload = c.Payload
	case *DeleteUserCommand:
		if u.Deleted {
			return nil, fmt.Errorf("user already deleted")
		}
		event.EventType = "deleted"
	default:
		return nil, fmt.Errorf("unknown command type: %s", c.CommandType)
	}
	return []*gen.EventEnvelope{event}, nil
}
