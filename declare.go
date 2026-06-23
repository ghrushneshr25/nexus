package nexus

import (
	"fmt"
	"reflect"
)

// Declare registers a default service constructor.
//
// A default service is eligible for constructor injection and can be resolved
// through Get or MustGet. Declare validates and stores constructor metadata
// only; it does not invoke the constructor.
//
// Valid signatures:
//
//	func() Service
//	func() (Service, error)
//	func(DependencyA, DependencyB) Service
//	func(DependencyA, DependencyB) (Service, error)
func Declare(constructor any) error {
	return declare("", constructor)
}

// MustDeclare registers a default service constructor and panics on failure.
//
// It is intended for init() functions, where invalid registration is an
// application wiring error and should fail fast.
func MustDeclare(constructor any) {
	if err := Declare(constructor); err != nil {
		panic(err)
	}
}

// DeclareNamed registers a named service constructor.
//
// Named services are intended for explicit runtime lookup through Lookup or
// MustLookup. They are not selected automatically during constructor
// injection; interface constructor parameters always resolve the default
// declaration for that interface.
//
// Valid signatures:
//
//	func() Service
//	func() (Service, error)
//	func(DependencyA, DependencyB) Service
//	func(DependencyA, DependencyB) (Service, error)
func DeclareNamed(name string, constructor any) error {
	if name == "" {
		return ErrInvalidServiceName
	}

	return declare(name, constructor)
}

// MustDeclareNamed registers a named service constructor and panics on
// failure.
//
// It is intended for init() functions, where invalid registration is an
// application wiring error and should fail fast.
func MustDeclareNamed(name string, constructor any) {
	if err := DeclareNamed(name, constructor); err != nil {
		panic(err)
	}
}

// declare validates and stores a service constructor under the supplied name.
//
// An empty name represents the default service declaration. A non-empty name
// represents a named service declaration. This helper centralizes duplicate
// detection and registry mutation for both public declaration APIs.
func declare(name string, constructor any) error {
	parsed, err := parseConstructor(constructor)
	if err != nil {
		return err
	}

	// parseConstructor always creates a default key. Override its name here
	// so the same constructor metadata can be used for default and named
	// registrations.
	parsed.key.name = name

	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	if _, exists := globalRegistry.declarations[parsed.key]; exists {
		return fmt.Errorf(
			"%w: %s",
			ErrDuplicateServiceDeclaration,
			serviceKeyString(parsed.key),
		)
	}

	globalRegistry.declarations[parsed.key] = parsed

	return nil
}

// parseConstructor validates a constructor and converts it into declaration
// metadata.
//
// This function must not mutate globalRegistry. Keeping parsing separate makes
// it easier to test and ensures registry locking is held only for map access.
func parseConstructor(constructor any) (declaration, error) {
	value := reflect.ValueOf(constructor)

	// A declaration must be backed by a function. reflect.ValueOf(nil) is
	// invalid, while other values may be valid but still not functions.
	if !value.IsValid() || value.Kind() != reflect.Func {
		return declaration{}, ErrConstructorMustBeFunction
	}

	typ := value.Type()

	// Nexus supports constructors returning either:
	//
	//	func(...) Service
	//	func(...) (Service, error)
	//
	// Any other number of return values is unsupported.
	if typ.NumOut() != 1 && typ.NumOut() != 2 {
		return declaration{}, fmt.Errorf(
			"%w: constructor must return Service or (Service, error)",
			ErrInvalidConstructor,
		)
	}

	// The first return value defines the service contract. Nexus uses
	// interface types as service identities and intentionally rejects
	// concrete implementation return types.
	serviceType := typ.Out(0)
	if serviceType.Kind() != reflect.Interface {
		return declaration{}, fmt.Errorf(
			"%w: %s",
			ErrServiceMustBeInterface,
			serviceType,
		)
	}

	returnsError := false

	// When a constructor has two return values, the second one must be the
	// standard error interface exactly.
	if typ.NumOut() == 2 {
		errorType := reflect.TypeOf((*error)(nil)).Elem()

		if typ.Out(1) != errorType {
			return declaration{}, fmt.Errorf(
				"%w: got %s",
				ErrInvalidConstructorError,
				typ.Out(1),
			)
		}

		returnsError = true
	}

	// Dependencies must preserve constructor argument order because the
	// resolver will later pass resolved values to constructor.Call in this
	// same order.
	dependencies := make([]reflect.Type, 0, typ.NumIn())

	for i := 0; i < typ.NumIn(); i++ {
		dependencies = append(dependencies, typ.In(i))
	}

	return declaration{
		key: serviceKey{
			contract: serviceType,
			name:     "",
		},
		constructor:  value,
		dependencies: dependencies,
		returnsError: returnsError,
	}, nil
}

// serviceKeyString returns a stable service-key representation for errors.
//
// Default service output:
//
//	package.MessagingClient
//
// Named service output:
//
//	package.MessagingClient["orders"]
func serviceKeyString(key serviceKey) string {
	base := key.contract.String()

	if key.name != "" {
		return fmt.Sprintf("%s[%q]", base, key.name)
	}

	if key.group != "" {
		return fmt.Sprintf("%s group %q", base, key.group)
	}

	return base
}

// DeclareGroup registers constructor as an ordered member of group.
//
// Multiple constructors may be registered for the same interface and group.
// Group members are resolved explicitly through GetGroup or MustGetGroup;
// they are never selected automatically for ordinary constructor injection.
//
// A group name must be non-empty. Group members preserve declaration order.
func DeclareGroup(group string, constructor any) error {
	if group == "" {
		return ErrInvalidGroupName
	}

	parsed, err := parseConstructor(constructor)
	if err != nil {
		return err
	}

	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	key := groupKey{
		contract: parsed.key.contract,
		group:    group,
	}

	globalRegistry.nextGroupMemberID++

	parsed.key = serviceKey{
		contract: parsed.key.contract,
		group:    group,
		memberID: globalRegistry.nextGroupMemberID,
	}

	globalRegistry.groups[key] = append(
		globalRegistry.groups[key],
		groupMember{
			declaration: parsed,
		},
	)

	return nil
}

// MustDeclareGroup registers constructor as a member of group and panics if
// registration fails.
//
// It is intended for application wiring where invalid registration should
// fail fast.
func MustDeclareGroup(group string, constructor any) {
	if err := DeclareGroup(group, constructor); err != nil {
		panic(err)
	}
}
