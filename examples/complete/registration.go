package main

import "github.com/ghrushneshr25/nexus"

func registerServices() {
	nexus.MustDeclareValue(Config{
		ApplicationName: "nexus-complete-example",
		DatabaseURL:     "postgres://localhost:5432/nexus",
	})

	nexus.MustDeclare(NewUserRepository)
	nexus.MustDeclare(NewUserService)

	nexus.MustDeclareNamed("orders", NewOrdersClient)
	nexus.MustDeclareNamed("audit", NewAuditClient)

	nexus.MustDeclareGroup("event-handlers", NewAuditHandler)
	nexus.MustDeclareGroup("event-handlers", NewMetricsHandler)
}
