package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ghrushneshr25/nexus"
)

func main() {
	registerServices()

	if err := nexus.Validate(); err != nil {
		log.Fatal(err)
	}

	if err := nexus.Initialize(); err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	users := nexus.MustGet(userServiceContract)
	orders := nexus.MustGetNamed(ordersClientContract)
	audit := nexus.MustGetNamed(auditClientContract)
	handlers := nexus.MustGetGroup[EventHandler]("event-handlers")

	fmt.Println(users.Get("42"))

	if err := orders.Publish(
		ctx,
		"orders.created",
		[]byte(`{"id":"42"}`),
	); err != nil {
		log.Fatal(err)
	}

	if err := audit.Publish(
		ctx,
		"audit.events",
		[]byte(`{"action":"order-created"}`),
	); err != nil {
		log.Fatal(err)
	}

	event := Event{
		Type: "order.created",
		ID:   "42",
	}

	for _, handler := range handlers {
		if err := handler.Handle(ctx, event); err != nil {
			log.Fatal(err)
		}
	}
}
