package nexus

import (
	"reflect"
	"sync"
)

// serviceKey uniquely identifies one resolvable Nexus singleton.
//
// Default services use an empty name and memberID zero.
// Named services use a non-empty name and memberID zero.
// Group members use a non-empty group and a unique non-zero memberID.
type serviceKey struct {
	contract reflect.Type
	name     string
	group    string
	memberID uint64
}

// declaration stores normalized constructor metadata.
//
// Constructors are parsed during declaration and invoked only during
// resolution.
type declaration struct {
	key serviceKey

	constructor reflect.Value

	dependencies []reflect.Type

	returnsError bool
}

// groupKey identifies all members registered for one interface contract and
// group name.
type groupKey struct {
	contract reflect.Type
	group    string
}

// groupMember represents one ordered constructor registration in a group.
//
// Its declaration key has a unique memberID, allowing multiple constructors
// for the same interface to have independent singleton instances.
type groupMember struct {
	declaration declaration
}

// buildState represents one in-progress singleton construction.
//
// Callers arriving while construction is in progress wait for done to close,
// then retry resolution.
type buildState struct {
	done chan struct{}
}

// registry owns the package-wide Nexus declaration and resolution state.
type registry struct {
	mu sync.RWMutex

	// declarations contains default and named service declarations.
	declarations map[serviceKey]declaration

	// groups contains ordered multi-binding registrations.
	//
	// The slice order is declaration order and is preserved by GetGroup.
	groups map[groupKey][]groupMember

	// nextGroupMemberID creates unique service keys for group members.
	nextGroupMemberID uint64

	// values stores concrete constructor dependencies by exact type.
	values map[reflect.Type]reflect.Value

	// instances stores successfully constructed singleton services, including
	// default, named, and group-member services.
	instances map[serviceKey]reflect.Value

	// building prevents duplicate concurrent construction of one service key.
	building map[serviceKey]*buildState
}

// globalRegistry is the package-wide registry used by the public Nexus API.
var globalRegistry = registry{
	declarations: make(map[serviceKey]declaration),
	groups:       make(map[groupKey][]groupMember),
	values:       make(map[reflect.Type]reflect.Value),
	instances:    make(map[serviceKey]reflect.Value),
	building:     make(map[serviceKey]*buildState),
}

// resolveContext tracks the active dependency chain for one resolution call.
//
// It is not shared between goroutines.
type resolveContext struct {
	path map[serviceKey]struct{}
}
