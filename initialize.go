package nexus

import (
	"fmt"
	"sort"
)

// Initialize constructs and caches every registered singleton service. It initializes default services, named services, and every group member. Dependencies are resolved through the normal resolver, so each service is constructed only once and dependencies are created before their dependents.
// Initialize does not call Starter.Start or Stopper.Stop. It only invokes constructors and populates the singleton cache.
func Initialize() error {
	keys := initializationKeys()

	for _, key := range keys {
		resolution := &resolveContext{
			path: make(map[serviceKey]struct{}),
		}

		if _, err := resolveDefault(key, resolution); err != nil {
			return fmt.Errorf("nexus: initialize %s: %w", serviceKeyString(key), err)
		}
	}

	return nil
}

// MustInitialize constructs and caches every registered singleton service.
// It panics if any constructor fails.
func MustInitialize() {
	if err := Initialize(); err != nil {
		panic(err)
	}
}

// initializationKeys returns every declared service key in deterministic order.
// The registry lock is held only while copying declaration metadata. Group
// members are included separately because they are stored outside the default
// and named declaration map.
func initializationKeys() []serviceKey {
	globalRegistry.mu.RLock()

	keys := make([]serviceKey, 0, len(globalRegistry.declarations))

	for key := range globalRegistry.declarations {
		keys = append(keys, key)
	}

	for _, members := range globalRegistry.groups {
		for _, member := range members {
			keys = append(keys, member.declaration.key)
		}
	}

	globalRegistry.mu.RUnlock()

	sort.Slice(keys, func(i, j int) bool {
		return initializationKeyLess(keys[i], keys[j])
	})

	return keys
}

// initializationKeyLess defines deterministic initialization order. Default services are initialized before named services, which are initialized before group members when contracts are otherwise equal. Group members use declaration order through memberID.
func initializationKeyLess(left, right serviceKey) bool {
	leftContract := left.contract.String()
	rightContract := right.contract.String()

	if leftContract != rightContract {
		return leftContract < rightContract
	}

	leftKind := initializationKind(left)
	rightKind := initializationKind(right)

	if leftKind != rightKind {
		return leftKind < rightKind
	}

	if left.name != right.name {
		return left.name < right.name
	}

	if left.group != right.group {
		return left.group < right.group
	}

	return left.memberID < right.memberID
}

// initializationKind returns a stable sort category for a service key.
// Default services sort first, named services second, and group members last.
func initializationKind(key serviceKey) int {
	if key.group != "" {
		return 2
	}

	if key.name != "" {
		return 1
	}

	return 0
}
