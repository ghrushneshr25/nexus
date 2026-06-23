// Package nexus provides a lightweight, reflection-based dependency injection
// container for Go applications.
//
// Nexus registers constructors during application initialization and resolves
// services lazily on first use. Each successfully resolved service is cached as
// a singleton and reused for later resolutions.
//
// Services are identified by interface contracts. A constructor must return an
// interface, optionally followed by error:
//
//	func() Service
//	func() (Service, error)
//	func(DependencyA, DependencyB) Service
//	func(DependencyA, DependencyB) (Service, error)
//
// Interface constructor parameters are resolved as default Nexus services.
// Concrete constructor parameters are resolved from values registered with
// DeclareValue.
//
// Default services are registered with Declare and resolved with Get:
//
//	func init() {
//		nexus.MustDeclare(NewService)
//	}
//
//	service := nexus.MustGet(ServiceContract)
//
// Named services allow multiple implementations of the same interface. They
// are registered with DeclareNamed and resolved explicitly with GetNamed:
//
//	func init() {
//		nexus.MustDeclareNamed("orders", NewOrdersClient)
//	}
//
//	orders := nexus.MustGetNamed(OrdersClientContract)
//
// Named services are not selected automatically during constructor injection.
// Interface constructor parameters always resolve the default declaration for
// that interface.
//
// Nexus invokes constructors lazily and caches only successful, non-nil
// results. Constructor errors and failed resolutions are returned by Get and
// GetNamed, or cause MustGet and MustGetNamed to panic.
//
// Concurrent resolution of the same service constructs it at most once.
// Nexus also detects direct and indirect circular dependencies during
// resolution.

// Validate checks the registered dependency graph without invoking
// constructors. It detects missing default service declarations, missing
// concrete values, and direct or indirect circular dependencies.
package nexus
