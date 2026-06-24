package main

import (
	"context"

	"github.com/ghrushneshr25/nexus"
)

type UserRepository interface {
	Find(string) string
}

type UserService interface {
	Get(string) string
}

type MessageClient interface {
	Publish(context.Context, string, []byte) error
}

type Event struct {
	Type string
	ID   string
}

type EventHandler interface {
	Handle(context.Context, Event) error
}

func userServiceContract() nexus.Contract[UserService] {
	return nexus.ContractOf[UserService]()
}

func ordersClientContract() nexus.Contract[MessageClient] {
	return nexus.NamedContractOf[MessageClient]("orders")
}

func auditClientContract() nexus.Contract[MessageClient] {
	return nexus.NamedContractOf[MessageClient]("audit")
}
