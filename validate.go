package nexus

import (
	"fmt"
	"maps"
	"reflect"
)

// validationState represents a service's depth-first traversal state.
type validationState uint8

const (
	validationUnvisited validationState = iota
	validationVisiting
	validationVisited
)

// Validate checks whether all registered service declarations have resolvable
// dependencies without invoking constructors.
//
// It validates default and named declarations, detects missing default service
// declarations and missing concrete values, and detects direct or indirect
// circular dependencies. Validate does not create singleton instances.
func Validate() error {
	globalRegistry.mu.RLock()

	// Copy registry metadata so validation does not hold the registry lock
	// during graph traversal.
	declarations := make(map[serviceKey]declaration, len(globalRegistry.declarations))

	maps.Copy(declarations, globalRegistry.declarations)

	// Validation only needs to know whether a concrete value type exists.
	values := make(map[reflect.Type]struct{}, len(globalRegistry.values))
	for valueType := range globalRegistry.values {
		values[valueType] = struct{}{}
	}

	globalRegistry.mu.RUnlock()

	states := make(map[serviceKey]validationState, len(declarations))

	for key := range declarations {
		if err := validateDeclaration(key, declarations, values, states); err != nil {
			return err
		}
	}

	return nil
}

// validateDeclaration recursively validates one service and its dependencies.
func validateDeclaration(key serviceKey, declarations map[serviceKey]declaration, values map[reflect.Type]struct{}, states map[serviceKey]validationState,
) error {
	switch states[key] {
	case validationVisited:
		return nil

	case validationVisiting:
		return fmt.Errorf("%w: %s", ErrCircularDependency, serviceKeyString(key))
	}

	declared, exists := declarations[key]
	if !exists {
		return fmt.Errorf("%w: %s", ErrServiceNotDeclared, serviceKeyString(key))
	}

	states[key] = validationVisiting

	// If dependency validation fails, restore this node to unvisited so the
	// state map remains internally consistent for the current traversal.
	defer func() {
		if states[key] == validationVisiting {
			states[key] = validationUnvisited
		}
	}()

	for _, dependencyType := range declared.dependencies {
		// Interface dependencies always resolve through the default service
		// declaration, even when the current service itself is named.
		if dependencyType.Kind() == reflect.Interface {
			dependencyKey := serviceKey{
				contract: dependencyType,
				name:     "",
			}

			if err := validateDeclaration(dependencyKey, declarations, values, states); err != nil {
				return fmt.Errorf("%w: %s: %w", ErrDependencyNotDeclared, dependencyType, err)
			}

			continue
		}

		// Concrete dependencies must be explicitly registered as values.
		if _, exists := values[dependencyType]; !exists {
			return fmt.Errorf("%w: %s", ErrDependencyNotDeclared, dependencyType)
		}
	}

	states[key] = validationVisited

	return nil
}
