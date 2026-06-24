package main

import (
	"fmt"

	"github.com/ghrushneshr25/nexus"
)

type MessageClient interface {
	Name() string
}

type messageClient struct {
	name string
}

func (c messageClient) Name() string {
	return c.name
}

func NewOrdersClient() MessageClient {
	return messageClient{
		name: "orders",
	}
}

func NewAuditClient() MessageClient {
	return messageClient{
		name: "audit",
	}
}

func ordersClientContract() nexus.Contract[MessageClient] {
	return nexus.NamedContractOf[MessageClient]("orders")
}

func auditClientContract() nexus.Contract[MessageClient] {
	return nexus.NamedContractOf[MessageClient]("audit")
}

func main() {
	nexus.MustDeclareNamed("orders", NewOrdersClient)
	nexus.MustDeclareNamed("audit", NewAuditClient)

	orders := nexus.MustGetNamed(ordersClientContract)
	audit := nexus.MustGetNamed(auditClientContract)

	fmt.Println("orders client:", orders.Name())
	fmt.Println("audit client:", audit.Name())
}
