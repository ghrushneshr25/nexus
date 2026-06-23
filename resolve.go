package nexus

import (
	"fmt"
	"reflect"
)

// Get resolves the default singleton service represented by contract.
//
// Services are created lazily. The first successful resolution constructs and
// caches the service; later resolutions return the cached singleton.
//
// Get accepts only contracts created through ContractOf. Use GetNamed for
// contracts created through NamedContractOf.
func Get[T any](contract func() Contract[T]) (T, error) {
	var zero T

	if contract == nil {
		return zero, ErrNilContractFunction
	}

	descriptor := contract()

	if descriptor.name != "" {
		return zero, ErrNamedContractRequiresGetNamed
	}

	key := serviceKey{
		contract: descriptor.typ,
		name:     "",
	}

	resolution := &resolveContext{
		path: make(map[serviceKey]struct{}),
	}

	value, err := resolveDefault(key, resolution)
	if err != nil {
		return zero, err
	}

	service, ok := value.Interface().(T)
	if !ok {
		return zero, fmt.Errorf(
			"nexus: resolved service %s cannot be assigned to requested contract",
			serviceKeyString(key),
		)
	}

	return service, nil
}

// MustGet resolves the default singleton service represented by contract.
//
// It panics if resolution fails. Use Get when the caller needs to handle
// resolution failures explicitly.
func MustGet[T any](contract func() Contract[T]) T {
	service, err := Get(contract)
	if err != nil {
		panic(err)
	}

	return service
}

// GetNamed resolves the named singleton service represented by contract.
//
// The supplied contract must be created with NamedContractOf. Named services
// are intended for explicit runtime selection and are not injected
// automatically into constructors.
func GetNamed[T any](contract func() Contract[T]) (T, error) {
	var zero T

	if contract == nil {
		return zero, ErrNilContractFunction
	}

	descriptor := contract()

	if descriptor.name == "" {
		return zero, ErrInvalidServiceName
	}

	key := serviceKey{
		contract: descriptor.typ,
		name:     descriptor.name,
	}

	resolution := &resolveContext{
		path: make(map[serviceKey]struct{}),
	}

	value, err := resolveDefault(key, resolution)
	if err != nil {
		return zero, err
	}

	service, ok := value.Interface().(T)
	if !ok {
		return zero, fmt.Errorf(
			"nexus: resolved service %s cannot be assigned to requested contract",
			serviceKeyString(key),
		)
	}

	return service, nil
}

// MustGetNamed resolves the named singleton service represented by contract.
//
// It panics if named resolution fails. It is appropriate for application
// wiring where a missing named declaration is a startup configuration error.
func MustGetNamed[T any](contract func() Contract[T]) T {
	service, err := GetNamed(contract)
	if err != nil {
		panic(err)
	}

	return service
}

// resolveDefault resolves a singleton service by its full service key.
//
// The key can represent either a default service or a named service. It
// guarantees that at most one goroutine constructs a given service at a time.
// Other goroutines wait for the active build to finish, then retry resolution
// so they receive either the cached singleton or a fresh error.
func resolveDefault(
	key serviceKey,
	resolution *resolveContext,
) (reflect.Value, error) {
	if _, exists := resolution.path[key]; exists {
		return reflect.Value{}, fmt.Errorf(
			"%w: %s",
			ErrCircularDependency,
			serviceKeyString(key),
		)
	}

	resolution.path[key] = struct{}{}
	defer delete(resolution.path, key)

	for {
		// Read the cache first. This is the normal fast path after a service
		// has been constructed successfully.
		globalRegistry.mu.RLock()
		instance, exists := globalRegistry.instances[key]
		globalRegistry.mu.RUnlock()

		if exists {
			return instance, nil
		}

		// Coordinate ownership of construction for this service key.
		globalRegistry.mu.Lock()

		// Another goroutine may have completed construction after the first
		// cache read but before this goroutine acquired the write lock.
		if instance, exists := globalRegistry.instances[key]; exists {
			globalRegistry.mu.Unlock()
			return instance, nil
		}

		// If another goroutine is already building this service, wait for it
		// to finish and then retry from the cache lookup.
		if state, building := globalRegistry.building[key]; building {
			done := state.done
			globalRegistry.mu.Unlock()

			<-done
			continue
		}

		// This goroutine owns construction for key.
		state := &buildState{
			done: make(chan struct{}),
		}
		globalRegistry.building[key] = state

		declaredConstructor, declared := globalRegistry.declarations[key]
		globalRegistry.mu.Unlock()

		// finish releases waiting goroutines and removes the in-progress
		// marker. It must run for both successful and failed builds.
		finish := func() {
			globalRegistry.mu.Lock()
			delete(globalRegistry.building, key)
			close(state.done)
			globalRegistry.mu.Unlock()
		}

		if !declared {
			finish()

			return reflect.Value{}, fmt.Errorf(
				"%w: %s",
				ErrServiceNotDeclared,
				serviceKeyString(key),
			)
		}

		arguments, err := resolveDependencies(
			declaredConstructor.dependencies,
			resolution,
		)
		if err != nil {
			finish()
			return reflect.Value{}, err
		}

		// Never hold the registry lock while calling user constructor code.
		results := declaredConstructor.constructor.Call(arguments)
		service := results[0]

		if declaredConstructor.returnsError {
			errValue := results[1]

			if !errValue.IsNil() {
				finish()

				return reflect.Value{}, fmt.Errorf(
					"nexus: construct %s: %w",
					serviceKeyString(key),
					errValue.Interface().(error),
				)
			}
		}

		// Constructor return contracts are always interfaces, so IsNil is
		// valid here.
		if service.IsNil() {
			finish()

			return reflect.Value{}, fmt.Errorf(
				"%w: %s",
				ErrConstructorReturnedNil,
				serviceKeyString(key),
			)
		}

		// Cache the successful singleton before waking waiters.
		globalRegistry.mu.Lock()
		globalRegistry.instances[key] = service
		globalRegistry.mu.Unlock()

		finish()

		return service, nil
	}
}

// resolveDependencies resolves constructor parameters in declaration order.
//
// Interface parameters are resolved as default Nexus services. Non-interface
// parameters are resolved from values registered through DeclareValue.
func resolveDependencies(
	dependencies []reflect.Type,
	resolution *resolveContext,
) ([]reflect.Value, error) {
	// Constructor.Call requires one argument for each constructor parameter,
	// in exactly the same order as the original function signature.
	arguments := make([]reflect.Value, 0, len(dependencies))

	for _, dependencyType := range dependencies {
		// Interface parameters represent service contracts. Resolve their
		// default implementation recursively.
		if dependencyType.Kind() == reflect.Interface {
			value, err := resolveDefault(
				serviceKey{
					contract: dependencyType,
					name:     "",
				},
				resolution,
			)
			if err != nil {
				return nil, fmt.Errorf(
					"%w: %s: %w",
					ErrDependencyNotDeclared,
					dependencyType,
					err,
				)
			}

			arguments = append(arguments, value)
			continue
		}

		// Concrete parameter types are injected only from explicitly
		// registered values, using exact reflect.Type equality.
		globalRegistry.mu.RLock()
		value, exists := globalRegistry.values[dependencyType]
		globalRegistry.mu.RUnlock()

		if !exists {
			return nil, fmt.Errorf(
				"%w: %s",
				ErrDependencyNotDeclared,
				dependencyType,
			)
		}

		arguments = append(arguments, value)
	}

	return arguments, nil
}
