package nexus

import (
	"fmt"
	"reflect"
)

// DeclareValue registers a static dependency by its exact Go type.
//
// Values are intended for configuration and immutable application-owned
// dependencies. They will later be injected into constructors when a
// constructor parameter is not an interface.
func DeclareValue(value any) error {
	if value == nil {
		return ErrNilValue
	}

	reflectedValue := reflect.ValueOf(value)

	typ := reflectedValue.Type()

	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	if _, exists := globalRegistry.values[typ]; exists {
		return fmt.Errorf(
			"%w: %s",
			ErrDuplicateValueDeclaration,
			typ,
		)
	}

	globalRegistry.values[typ] = reflectedValue
	return nil
}

// MustDeclareValue registers a static value and panics on failure.
//
// It is intended for init() functions, where invalid or duplicate application
// configuration is a wiring error.
func MustDeclareValue(value any) {
	if err := DeclareValue(value); err != nil {
		panic(err)
	}
}
