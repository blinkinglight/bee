package main

import (
	"context"
	"log"

	"github.com/delaneyj/toolbelt/embeddednats"
)

func main() {

	ns, err := embeddednats.New(context.Background(), embeddednats.WithShouldClearData(true))
	if err != nil {
		panic(err)
	}
	defer ns.Close()

	ns.WaitForServer()
	log.Printf("NATS server is running at %s", ns.NatsServer.ClientURL())
	select {}
}
