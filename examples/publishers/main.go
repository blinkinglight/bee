package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/blinkinglight/bee/gen"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

func main() {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		panic(err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		panic(err)
	}

	base, _ := strconv.Atoi(os.Args[1])

	for i := 0; i < 10; i++ {
		cmd := &gen.CommandEnvelope{
			Aggregate:     "users",
			AggregateId:   fmt.Sprintf("user-%d", base+i),
			CommandType:   "create",
			CorrelationId: fmt.Sprintf("correlation-%d", base+i),
			Payload:       []byte(`{"name": "User ` + fmt.Sprintf("%d", i) + `", "country": "Country ` + fmt.Sprintf("%d", i) + `"}`),
		}
		b, _ := proto.Marshal(cmd)
		js.Publish("cmds.users", b)
	}
}
