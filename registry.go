package nexus

import (
	"reflect"
	"sync"
)

// serviceKey uniquely identifies a Nexus service declaration.
//
// A default service uses an empty name. Named services will use a non-empty
// name later, so the field is included now to keep the internal key stable.
//
// Example default key:
//
//	MessagingClient + ""
type serviceKey struct {
	contract reflect.Type
	name     string
}

// declaration stores normalized constructor metadata.
//
// Nexus parses constructors during Declare rather than during resolution.
// This keeps resolution focused on dependency lookup and constructor execution.
//
// Constructors are never invoked while creating a declaration.
type declaration struct {
	// key identifies the service returned by this constructor.
	key serviceKey

	// constructor is the reflected user-provided constructor function.
	constructor reflect.Value

	// dependencies preserves constructor parameter types in declaration order.
	// They will be resolved and passed to constructor in the same order later.
	dependencies []reflect.Type

	// returnsError reports whether the constructor has this form:
	//
	//	func(...) (Service, error)
	returnsError bool
}

// buildState represents one in-progress singleton construction.
//
// Callers that arrive while a service is being constructed wait for done to be
// closed, then retry resolution. The constructing goroutine closes done after
// it either caches a successful instance or finishes with an error.
type buildState struct {
	done chan struct{}
}

// registry owns the process-wide Nexus declaration state.
//
// Nexus uses one global registry because services are intended to be declared
// from package init() functions. All access to declarations must hold mu.
//
// At this stage, declarations contain metadata only. No singleton instances
// are stored and no constructors are executed.
type registry struct {
	mu sync.RWMutex

	// declarations maps each service key to exactly one constructor declaration.
	//
	// A duplicate default declaration for the same interface must be rejected.
	declarations map[serviceKey]declaration
	values       map[reflect.Type]reflect.Value
	instances    map[serviceKey]reflect.Value

	// building tracks services currently being constructed.
	//
	// It prevents multiple goroutines from invoking the same constructor
	// concurrently. Failed builds are not retained after completion.
	building map[serviceKey]*buildState
}

// globalRegistry is the package-wide registry used by the public Nexus API.
// It must be initialized before any importing package init() function runs.
var globalRegistry = registry{
	declarations: make(map[serviceKey]declaration),
	values:       make(map[reflect.Type]reflect.Value),
	instances:    make(map[serviceKey]reflect.Value),
	building:     make(map[serviceKey]*buildState),
}

// resolveContext tracks the active dependency chain for one resolution call.
//
// It is not shared between goroutines. It exists only to detect cycles such
// as ServiceA -> ServiceB -> ServiceA before the resolver waits on itself.
type resolveContext struct {
	path map[serviceKey]struct{}
}
