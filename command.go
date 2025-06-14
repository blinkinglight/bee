package bee

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/blinkinglight/bee/gen"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

const commandsSubject = "cmds.>"
const commandsDurableName = "commands"
const commandsStream = "COMMANDS"

var (
	registry = make(map[string]CommandHandlerFunc)
	mu       = &sync.RWMutex{}
)

type CommandHandlerFunc func(ctx context.Context, m *gen.CommandEnvelope) ([]*gen.EventEnvelope, error)

func Register(ctx context.Context, aggregate string, handler CommandHandlerFunc) error {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := registry[aggregate]; exists {
		return fmt.Errorf("aggregate %s already registered", aggregate)
	}
	registry[aggregate] = handler
	return nil
}

func getCommandHandler(aggregate string) (CommandHandlerFunc, error) {
	if _, ok := registry[aggregate]; !ok {
		return nil, fmt.Errorf("aggregate %s not found", aggregate)
	}
	return registry[aggregate], nil
}

type CommandProcessor struct {
	js nats.JetStreamContext
	nc *nats.Conn
}

func NewCommandProcessor(ctx context.Context, nc *nats.Conn, js nats.JetStreamContext) {
	cp := &CommandProcessor{js: js, nc: nc}
	c, cancel := context.WithCancel(ctx)
	go func() {
		for {
			err := cp.init(c, cancel)
			if err != nil {
				log.Printf("Error initializing command processor: %v", err)
				continue
			}
		}
	}()
	<-c.Done()
}

func (cp *CommandProcessor) init(ctx context.Context, cancel context.CancelFunc) error {

	_, _ = cp.js.AddStream(&nats.StreamConfig{
		Name:       commandsStream,
		Subjects:   []string{commandsSubject},
		Retention:  nats.WorkQueuePolicy,
		Storage:    nats.FileStorage,
		Replicas:   1,
		Duplicates: 5 * time.Minute,
	})

	var msgs = make(chan *nats.Msg, 256)

	sub, err := cp.js.Subscribe(commandsSubject, func(msg *nats.Msg) {
		log.Printf("Received command on subject %s", msg.Subject)
		msgs <- msg
	}, nats.BindStream(commandsStream), nats.ManualAck(), nats.DeliverAll(), nats.Durable(commandsDurableName), nats.MaxAckPending(200))

	if err != nil {
		log.Printf("Error subscribing to commands: %v", err)
		cancel()
	}
	defer sub.Unsubscribe()

	for msg := range msgs {
		log.Printf("Received command on subject %s", msg.Subject)
		var cmd gen.CommandEnvelope
		err := proto.Unmarshal(msg.Data, &cmd)
		if err != nil {
			log.Printf("Error unmarshalling message: %v", err)
			msg.Ack()
			continue
		}

		errorNotificationSubject := fmt.Sprintf("notifications.%s.error", cmd.CorrelationId)
		successNotificationSubject := fmt.Sprintf("notifications.%s.success", cmd.CorrelationId)

		handler, err := getCommandHandler(cmd.Aggregate)
		if err != nil {
			log.Printf("Error getting command handler: %v", err)
			if cmd.CorrelationId != "" {
				cp.nc.Publish(errorNotificationSubject, []byte(`{"message":"`+err.Error()+`"}`))
			}
			msg.Ack()
			continue
		}

		events, err := handler(context.Background(), &cmd)
		if err != nil {
			log.Printf("Error handling command: %v", err)
			if cmd.CorrelationId != "" {
				cp.nc.Publish(errorNotificationSubject, []byte(`{"message":"`+err.Error()+`"}`))
			}
			msg.Ack()
			continue
		}

		for _, event := range events {
			eventSubject := fmt.Sprintf("events.%s.%s.%s", event.AggregateType, event.AggregateId, event.EventType)
			b, _ := proto.Marshal(event)
			if _, err := cp.js.Publish(eventSubject, b); err != nil {
				log.Printf("Error publishing event %v", err)
			}
		}

		if cmd.CorrelationId != "" {
			cp.nc.Publish(successNotificationSubject, []byte(`{"message":"`+cmd.AggregateId+`"}`))
		}
		_ = msg.Ack()
	}
	<-ctx.Done()
	return nil
}
