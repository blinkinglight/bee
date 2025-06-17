package main

import (
	"log"
	"time"

	"github.com/blinkinglight/bee/gen"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func main() {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		panic(err)
	}
	defer nc.Close()
	s, _ := structpb.NewStruct(map[string]any{
		"id": "user-2408",
	})
	q := gen.QueryEnvelope{
		QueryType:     "one",
		ExtraMetadata: s,
	}
	b, _ := proto.Marshal(&q)
	r, err := nc.Request("query.users.get", b, 3*time.Second)
	if err != nil {
		log.Fatalf("Failed to send query: %v", err)
	}
	log.Printf("%s", r.Data)
}
