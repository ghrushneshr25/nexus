package main

import (
	"fmt"
	"log"

	"github.com/ghrushneshr25/nexus"
)

type Database interface {
	Name() string
}

type Service interface {
	Run()
}

type database struct{}

func (database) Name() string {
	return "postgres"
}

type service struct {
	database Database
}

func (s service) Run() {
	fmt.Println("service using database:", s.database.Name())
}

func NewDatabase() Database {
	fmt.Println("constructing database")

	return database{}
}

func NewService(database Database) Service {
	fmt.Println("constructing service")

	return service{
		database: database,
	}
}

func serviceContract() nexus.Contract[Service] {
	return nexus.ContractOf[Service]()
}

func main() {
	nexus.MustDeclare(NewDatabase)
	nexus.MustDeclare(NewService)

	if err := nexus.Validate(); err != nil {
		log.Fatal(err)
	}

	if err := nexus.Initialize(); err != nil {
		log.Fatal(err)
	}

	service := nexus.MustGet(serviceContract)

	service.Run()
}
