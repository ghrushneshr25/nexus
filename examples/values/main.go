package main

import (
	"fmt"

	"github.com/ghrushneshr25/nexus"
)

type Config struct {
	ApplicationName string
}

type Greeter interface {
	Greet(string) string
}

type greeter struct {
	config Config
}

func (g greeter) Greet(name string) string {
	return g.config.ApplicationName + ": hello, " + name
}

func NewGreeter(config Config) Greeter {
	return greeter{
		config: config,
	}
}

func greeterContract() nexus.Contract[Greeter] {
	return nexus.ContractOf[Greeter]()
}

func main() {
	nexus.MustDeclareValue(Config{
		ApplicationName: "nexus-example",
	})

	nexus.MustDeclare(NewGreeter)

	service := nexus.MustGet(greeterContract)

	fmt.Println(service.Greet("Nexus"))
}
