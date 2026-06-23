package nexus

import "reflect"

// Reset clears Nexus registry state.
//
// It is intended for tests. Production applications should not reset
// declarations after package init() registration has completed.
func Reset() {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	globalRegistry.declarations = make(map[serviceKey]declaration)
	globalRegistry.values = make(map[reflect.Type]reflect.Value)
	globalRegistry.instances = make(map[serviceKey]reflect.Value)
	globalRegistry.building = make(map[serviceKey]*buildState)
	globalRegistry.groups = make(map[groupKey][]groupMember)
	globalRegistry.nextGroupMemberID = 0
}
