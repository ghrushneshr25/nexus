package main

import (
	"fmt"

	"github.com/ghrushneshr25/nexus"
)

type Greeter interface {
	Greet(string) string
}

type greeter struct{}

func (greeter) Greet(name string) string {
	return "hello, " + name
}

func NewGreeter() Greeter {
	return greeter{}
}

func greeterContract() nexus.Contract[Greeter] {
	return nexus.ContractOf[Greeter]()
}

func main() {
	nexus.MustDeclare(NewGreeter)

	service := nexus.MustGet(greeterContract)

	fmt.Println(service.Greet("Nexus"))
}
