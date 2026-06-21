// Package nexus provides lazy, interface-first dependency resolution for Go.
//
// Services are declared through constructors, resolved by interface contracts,
// and cached as singletons after successful construction. Nexus supports
// default injectable services, named runtime service lookup, and registered
// values for configuration.
//
// Declarations are typically made from package init functions. Constructors
// are not executed during registration; services are created lazily when
// resolved.
//
// Nexus uses constructor injection only. Field injection, struct-tag injection,
// concrete-type resolution, and named constructor injection are not supported.
package nexus
