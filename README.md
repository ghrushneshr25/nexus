# nexus

Lightweight, interface-first dependency injection and service registry for Go. Nexus uses constructor injection and typed contracts to resolve services lazily as thread-safe singletons. Register default dependencies or named runtime services, keep implementations private, and avoid hidden field injection or concrete-type coupling.

## Usage

### Default service

Nexus registers constructors in `init()` and resolves services lazily.

```go
package main

import (
	"fmt"

	"github.com/ghrushneshr25/nexus"
)

type Config struct {
	DatabaseURL string
}

type UserRepository interface {
	Find(id string) string
}

type UserService interface {
	GetUser(id string) string
}

type userRepository struct {
	config Config
}

func (r *userRepository) Find(id string) string {
	return fmt.Sprintf("user:%s from %s", id, r.config.DatabaseURL)
}

type userService struct {
	repository UserRepository
}

func (s *userService) GetUser(id string) string {
	return s.repository.Find(id)
}

func NewUserRepository(config Config) UserRepository {
	return &userRepository{
		config: config,
	}
}

func NewUserService(repository UserRepository) UserService {
	return &userService{
		repository: repository,
	}
}

func UserServiceContract() nexus.Contract[UserService] {
	return nexus.ContractOf[UserService]()
}

func init() {
	nexus.MustDeclareValue(Config{
		DatabaseURL: "postgres://localhost:5432/app",
	})

	nexus.MustDeclare(NewUserRepository)
	nexus.MustDeclare(NewUserService)
}

func main() {
	service := nexus.MustGet(UserServiceContract)

	fmt.Println(service.GetUser("42"))
}
```

Nexus resolves the dependency graph lazily:

```text
UserService
└── UserRepository
    └── Config
```

- `UserRepository` is resolved from its default interface declaration.
- `Config` is resolved by exact concrete type from `MustDeclareValue`.
- Each successful service is constructed once and cached as a singleton.

### Named services

Use named services when one interface has multiple implementations that must be selected explicitly at runtime.

```go
package main

import (
	"context"
	"fmt"

	"github.com/ghrushneshr25/nexus"
)

type MessagingClient interface {
	Publish(ctx context.Context, topic string, body []byte) error
}

type messagingClient struct {
	name string
}

func (c *messagingClient) Publish(
	ctx context.Context,
	topic string,
	body []byte,
) error {
	fmt.Printf(
		"%s client published %q to %s\n",
		c.name,
		string(body),
		topic,
	)

	return nil
}

func NewOrdersClient() MessagingClient {
	return &messagingClient{
		name: "orders",
	}
}

func NewAuditClient() MessagingClient {
	return &messagingClient{
		name: "audit",
	}
}

func OrdersClientContract() nexus.Contract[MessagingClient] {
	return nexus.NamedContractOf[MessagingClient]("orders")
}

func AuditClientContract() nexus.Contract[MessagingClient] {
	return nexus.NamedContractOf[MessagingClient]("audit")
}

func init() {
	nexus.MustDeclareNamed("orders", NewOrdersClient)
	nexus.MustDeclareNamed("audit", NewAuditClient)
}

func main() {
	ordersClient := nexus.MustGetNamed(OrdersClientContract)
	auditClient := nexus.MustGetNamed(AuditClientContract)

	ctx := context.Background()

	_ = ordersClient.Publish(ctx, "orders.created", []byte(`{"id":"42"}`))
	_ = auditClient.Publish(ctx, "audit.events", []byte(`{"action":"created"}`))
}
```

`Get` and `GetNamed` have distinct responsibilities:

| Registration   | Resolution                  | Purpose                                  |
| -------------- | --------------------------- | ---------------------------------------- |
| `Declare`      | `Get` / `MustGet`           | Default implementation for an interface  |
| `DeclareNamed` | `GetNamed` / `MustGetNamed` | Explicitly selected named implementation |

Named services are not automatically injected into constructors. Interface constructor parameters always resolve the default service declaration.

### Error handling

Use `Get` and `GetNamed` when resolution failures must be handled:

```go
service, err := nexus.Get(UserServiceContract)
if err != nil {
	return fmt.Errorf("resolve user service: %w", err)
}
```

Use `MustGet` and `MustGetNamed` when resolution failure is an application wiring error and should stop execution:

```go
service := nexus.MustGet(UserServiceContract)
orders := nexus.MustGetNamed(OrdersClientContract)
```

### Lifecycle behavior

- Constructors are not called during `Declare` or `DeclareNamed`.
- A service is constructed only when first resolved.
- Successful services are cached as singletons.
- Constructor failures are not cached; a later resolution retries construction.
- Concurrent requests for the same service construct it at most once.
- Direct and indirect circular dependencies return an error.
- `Reset` clears registered declarations, values, cached instances, and in-progress build state. It is intended for tests.

### Validate dependency wiring

Call `Validate` after all registrations are complete and before starting the application. Validation checks the dependency graph without invoking constructors or creating singleton instances.

```go
func main() {
	if err := nexus.Validate(); err != nil {
		log.Fatal(err)
	}

	server := nexus.MustGet(ServerContract)
	server.Run()
}
```

Validate detects:
- missing default service declarations for interface dependencies;
- missing values for concrete constructor dependencies;
- direct and indirect circular dependencies.

Validation is structural only. It does not invoke constructors, connect to external systems, or guarantee that a constructor cannot return an error.

## Groups and multi-binding

Groups allow multiple implementations of the same interface to be registered and resolved together. They are useful for ordered middleware chains, plugins, event handlers, validators, exporters, and hooks.

Group members are resolved explicitly. They are not selected automatically when Nexus injects an interface dependency into a constructor.

```go
package main

import (
	"context"
	"fmt"

	"github.com/ghrushneshr25/nexus"
)

type Event struct {
	Type string
	ID   string
}

type EventHandler interface {
	Handle(ctx context.Context, event Event) error
}

type auditHandler struct{}

func (auditHandler) Handle(ctx context.Context, event Event) error {
	fmt.Printf("audit event: %s %s\n", event.Type, event.ID)

	return nil
}

type metricsHandler struct{}

func (metricsHandler) Handle(ctx context.Context, event Event) error {
	fmt.Printf("recording metric for event: %s\n", event.Type)

	return nil
}

type webhookHandler struct{}

func (webhookHandler) Handle(ctx context.Context, event Event) error {
	fmt.Printf("sending webhook for event: %s\n", event.ID)

	return nil
}

func NewAuditHandler() EventHandler {
	return auditHandler{}
}

func NewMetricsHandler() EventHandler {
	return metricsHandler{}
}

func NewWebhookHandler() EventHandler {
	return webhookHandler{}
}

func init() {
	nexus.MustDeclareGroup("event-handlers", NewAuditHandler)
	nexus.MustDeclareGroup("event-handlers", NewMetricsHandler)
	nexus.MustDeclareGroup("event-handlers", NewWebhookHandler)
}

func main() {
	handlers := nexus.MustGetGroup[EventHandler]("event-handlers")

	event := Event{
		Type: "order.created",
		ID:   "order-42",
	}

	for _, handler := range handlers {
		if err := handler.Handle(context.Background(), event); err != nil {
			panic(err)
		}
	}
}
```

Group members are returned in the same order they were declared:

```text
AuditHandler
MetricsHandler
WebhookHandler
```

Each group member is created lazily and cached independently as a singleton. Repeated calls to `GetGroup` or `MustGetGroup` return the same member instances.

```go
first := nexus.MustGetGroup[EventHandler]("event-handlers")
second := nexus.MustGetGroup[EventHandler]("event-handlers")

// first[0] and second[0] refer to the same AuditHandler singleton.
```

### Group rules

| Rule                     | Behavior                                                                                                                              |
| ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------- |
| Group name               | Must be non-empty.                                                                                                                    |
| Ordering                 | Members are returned in declaration order.                                                                                            |
| Multiple implementations | Multiple constructors returning the same interface may be registered in one group.                                                    |
| Duplicate constructor    | The same constructor cannot be registered twice for the same interface and group.                                                     |
| Default injection        | Group members are never injected automatically. Constructor interface dependencies always resolve the default `Declare` registration. |
| Named services           | Groups are separate from named services. Use `GetNamed` when selecting one implementation by name.                                    |
| Dependencies             | Group constructors support normal constructor injection and declared concrete values.                                                 |
| Lifecycle                | Each group member is a lazy singleton and is constructed at most once concurrently.                                                   |

Use named services when selecting one implementation explicitly:

```go
client := nexus.MustGetNamed(OrdersClientContract)
```

Use groups when executing every registered implementation:

```go
handlers := nexus.MustGetGroup[EventHandler]("event-handlers")
```
