package bee

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/blinkinglight/bee/gen"
	"google.golang.org/protobuf/proto"
)

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
	_, err := js.Publish("cmds."+cmd.Aggregate, b)
	return err
}
