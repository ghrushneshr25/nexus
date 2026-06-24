package main

import (
	"context"
	"fmt"
)

type userRepository struct {
	config Config
}

func (r userRepository) Find(id string) string {
	return fmt.Sprintf(
		"user=%s database=%s",
		id,
		r.config.DatabaseURL,
	)
}

type userService struct {
	repository UserRepository
}

func (s userService) Get(id string) string {
	return s.repository.Find(id)
}

type messageClient struct {
	name string
}

func (c messageClient) Publish(
	_ context.Context,
	topic string,
	message []byte,
) error {
	fmt.Printf(
		"client=%s topic=%s message=%s\n",
		c.name,
		topic,
		string(message),
	)

	return nil
}
