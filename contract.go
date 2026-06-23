package nexus

import (
	"fmt"
	"reflect"
)

// You need fmt for wrapping the error with the actual type.
// You need reflect to obtain the runtime type of T.

// Contract is an opaque typed descriptor for a Nexus service contract.
//
// T must be an interface. Nexus uses this interface type later as the stable
// identity for declaration and resolution.
type Contract[T any] struct {
	typ  reflect.Type
	name string
}

// ContractOf creates a typed descriptor for interface contract T.
//
// It must panic when T is not an interface because that is a package-wiring
// error, not a recoverable runtime resolution error.
func ContractOf[T any]() Contract[T] {
	typ := reflect.TypeOf((*T)(nil)).Elem()

	if typ.Kind() != reflect.Interface {
		panic(fmt.Errorf("%w: %s", ErrContractMustBeInterface, typ))
	}

	return Contract[T]{
		typ: typ,
	}
}

// NamedContractOf returns a contract for a named service implementation.
//
// The type parameter must be an interface. The name distinguishes this
// registration from the default service and from other named implementations
// of the same interface.
func NamedContractOf[T any](name string) Contract[T] {
	contract := ContractOf[T]()
	contract.name = name

	return contract
}
