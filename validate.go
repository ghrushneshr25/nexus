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
// It validates default services, named services, and group members. It detects
// missing default service declarations, missing concrete values, and direct or
// indirect circular dependencies. Validate does not create singleton instances.
func Validate() error {
	globalRegistry.mu.RLock()

	// Copy all ordinary declarations first.
	declarations := make(map[serviceKey]declaration, len(globalRegistry.declarations))
	maps.Copy(declarations, globalRegistry.declarations)

	// Flatten group-member declarations into the same lookup map. Each group
	// member has a unique serviceKey through memberID, so members remain
	// independently traversable and independently cycle-safe.
	for _, members := range globalRegistry.groups {
		for _, member := range members {
			declarations[member.declaration.key] = member.declaration
		}
	}

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
func validateDeclaration(key serviceKey, declarations map[serviceKey]declaration, values map[reflect.Type]struct{}, states map[serviceKey]validationState) error {
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

	// Restore an incomplete traversal if a dependency fails validation.
	defer func() {
		if states[key] == validationVisiting {
			states[key] = validationUnvisited
		}
	}()

	for _, dependencyType := range declared.dependencies {
		// Interface dependencies always resolve through the default service
		// declaration. Named services and group members are never injected
		// automatically.
		if dependencyType.Kind() == reflect.Interface {
			dependencyKey := serviceKey{
				contract: dependencyType,
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
