package bee

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/blinkinglight/bee/gen"
	"google.golang.org/protobuf/proto"
)

// PublishCommand publishes a command to the JetStream server.
// It takes a context, a CommandEnvelope, and an optional payload.
// If the payload is not nil, it marshals the payload into JSON and sets it in the CommandEnvelope.
// It retrieves the JetStream context from the context and publishes the command to the subject
// "cmds.<AggregateType>" with the serialized CommandEnvelope as the message body.
// If the JetStream context is not initialized, it returns an error.
// The function returns an error if the publish operation fails.
func PublishCommand(ctx context.Context, cmd *gen.CommandEnvelope, payload any) error {
	if payload != nil {
		b, _ := json.Marshal(payload)
		cmd.Payload = b
	}

	js, _ := JetStream(ctx)
	if js == nil {
		return errors.New("JetStream is not initialized")
	}

	b, _ := proto.Marshal(cmd)
	_, err := js.Publish(CommandsPrefix+"."+cmd.Aggregate, b)
	return err
}
