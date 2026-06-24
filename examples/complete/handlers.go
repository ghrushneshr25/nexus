package main

import (
	"context"
	"fmt"
)

type auditHandler struct{}

func (auditHandler) Handle(
	_ context.Context,
	event Event,
) error {
	fmt.Printf(
		"audit event=%s id=%s\n",
		event.Type,
		event.ID,
	)

	return nil
}

type metricsHandler struct{}

func (metricsHandler) Handle(
	_ context.Context,
	event Event,
) error {
	fmt.Printf(
		"metrics event=%s\n",
		event.Type,
	)

	return nil
}
