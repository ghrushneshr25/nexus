package main

import (
	"fmt"

	"github.com/ghrushneshr25/nexus"
)

type Event struct {
	Type string
}

type EventHandler interface {
	Handle(Event)
}

type auditHandler struct{}

func (auditHandler) Handle(event Event) {
	fmt.Println("audit handler:", event.Type)
}

type metricsHandler struct{}

func (metricsHandler) Handle(event Event) {
	fmt.Println("metrics handler:", event.Type)
}

func NewAuditHandler() EventHandler {
	return auditHandler{}
}

func NewMetricsHandler() EventHandler {
	return metricsHandler{}
}

func main() {
	nexus.MustDeclareGroup("event-handlers", NewAuditHandler)
	nexus.MustDeclareGroup("event-handlers", NewMetricsHandler)

	handlers := nexus.MustGetGroup[EventHandler]("event-handlers")

	for _, handler := range handlers {
		handler.Handle(Event{
			Type: "order.created",
		})
	}
}
