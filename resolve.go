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
		return zero, fmt.Errorf("nexus: resolved service %s cannot be assigned to requested contract", serviceKeyString(key))
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
		return zero, fmt.Errorf("nexus: resolved service %s cannot be assigned to requested contract", serviceKeyString(key))
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
// The key can represent a default service, a named service, or a group member.
// It guarantees that at most one goroutine constructs a given service at a
// time. Other goroutines wait for the active build to finish, then retry
// resolution so they receive either the cached singleton or a fresh error.
func resolveDefault(
	key serviceKey,
	resolution *resolveContext,
) (reflect.Value, error) {
	if _, exists := resolution.path[key]; exists {
		return reflect.Value{}, fmt.Errorf("%w: %s", ErrCircularDependency, serviceKeyString(key))
	}

	resolution.path[key] = struct{}{}
	defer delete(resolution.path, key)

	for {
		// Fast path: return an already-created singleton.
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

		// If another goroutine is building this service, wait for completion
		// and retry resolution from the cache lookup.
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

		// Default and named declarations are stored directly in declarations.
		// Group members are stored in an ordered group slice and are identified
		// by their unique memberID.
		var (
			declaredConstructor declaration
			declared            bool
		)

		if key.group == "" {
			declaredConstructor, declared = globalRegistry.declarations[key]
		} else {
			members := globalRegistry.groups[groupKey{
				contract: key.contract,
				group:    key.group,
			}]

			for _, member := range members {
				if member.declaration.key.memberID == key.memberID {
					declaredConstructor = member.declaration
					declared = true
					break
				}
			}
		}

		globalRegistry.mu.Unlock()

		// finish removes the in-progress marker and wakes all waiting
		// goroutines. It must run after both successful and failed builds.
		finish := func() {
			globalRegistry.mu.Lock()
			delete(globalRegistry.building, key)
			close(state.done)
			globalRegistry.mu.Unlock()
		}

		if !declared {
			finish()
			return reflect.Value{}, fmt.Errorf("%w: %s", ErrServiceNotDeclared, serviceKeyString(key))
		}

		arguments, err := resolveDependencies(declaredConstructor.dependencies, resolution)
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
				return reflect.Value{}, fmt.Errorf("nexus: construct %s: %w", serviceKeyString(key), errValue.Interface().(error))
			}
		}

		// All service constructors return interfaces, so IsNil is valid.
		if service.IsNil() {
			finish()

			return reflect.Value{}, fmt.Errorf("%w: %s", ErrConstructorReturnedNil, serviceKeyString(key))
		}

		// Cache the singleton before waking concurrent waiters.
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
//
// Group members are never injected automatically. If a constructor depends on
// an interface, Nexus always resolves that interface's default declaration.
func resolveDependencies(dependencies []reflect.Type, resolution *resolveContext) ([]reflect.Value, error) {
	arguments := make([]reflect.Value, 0, len(dependencies))

	for _, dependencyType := range dependencies {
		if dependencyType.Kind() == reflect.Interface {
			value, err := resolveDefault(
				serviceKey{
					contract: dependencyType,
				},
				resolution,
			)
			if err != nil {
				return nil, fmt.Errorf("%w: %s: %w", ErrDependencyNotDeclared, dependencyType, err)
			}

			arguments = append(arguments, value)
			continue
		}

		globalRegistry.mu.RLock()
		value, exists := globalRegistry.values[dependencyType]
		globalRegistry.mu.RUnlock()

		if !exists {
			return nil, fmt.Errorf("%w: %s", ErrDependencyNotDeclared, dependencyType)
		}

		arguments = append(arguments, value)
	}

	return arguments, nil
}

// GetGroup resolves all singleton services registered for T in group.
//
// Group members are returned in declaration order. Each member is constructed
// lazily and cached independently as a singleton.
//
// Group members are resolved explicitly and are never injected automatically
// into ordinary constructor parameters.
func GetGroup[T any](group string) ([]T, error) {
	if group == "" {
		return nil, ErrInvalidGroupName
	}

	contractType := ContractOf[T]().typ

	registryGroupKey := groupKey{
		contract: contractType,
		group:    group,
	}

	// Copy member keys while holding the lock, then resolve outside the lock
	// because constructors may execute arbitrary user code.
	globalRegistry.mu.RLock()

	members := globalRegistry.groups[registryGroupKey]
	memberKeys := make([]serviceKey, 0, len(members))

	for _, member := range members {
		memberKeys = append(memberKeys, member.declaration.key)
	}

	globalRegistry.mu.RUnlock()

	if len(memberKeys) == 0 {
		return nil, fmt.Errorf("%w: %s group %q", ErrServiceNotDeclared, contractType, group)
	}

	resolved := make([]T, 0, len(memberKeys))

	for _, memberKey := range memberKeys {
		resolution := &resolveContext{
			path: make(map[serviceKey]struct{}),
		}

		value, err := resolveDefault(memberKey, resolution)
		if err != nil {
			return nil, err
		}

		service, ok := value.Interface().(T)
		if !ok {
			return nil, fmt.Errorf("nexus: resolved group service %s cannot be assigned to requested contract", serviceKeyString(memberKey))
		}

		resolved = append(resolved, service)
	}

	return resolved, nil
}

// MustGetGroup resolves all singleton services registered for T in group and
// panics if resolution fails.
func MustGetGroup[T any](group string) []T {
	services, err := GetGroup[T](group)
	if err != nil {
		panic(err)
	}

	return services
}
