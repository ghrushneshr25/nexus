package nexus

import "errors"

var (
	// ErrContractMustBeInterface indicates contract must be an interface
	ErrContractMustBeInterface = errors.New("nexus: contract must be an interface")

	// ErrConstructorMustBeFunction indicates that Declare received a value that
	// is not a constructor function.
	ErrConstructorMustBeFunction = errors.New(
		"nexus: constructor must be a function",
	)

	// ErrInvalidConstructor indicates that a constructor has an unsupported
	// signature, such as an invalid number of return values.
	ErrInvalidConstructor = errors.New(
		"nexus: invalid constructor",
	)

	// ErrServiceMustBeInterface indicates that a constructor returns a concrete
	// type. Nexus service contracts must always be interfaces.
	ErrServiceMustBeInterface = errors.New(
		"nexus: service return type must be an interface",
	)

	// ErrInvalidConstructorError indicates that a constructor with two return
	// values does not use error as its second return type.
	ErrInvalidConstructorError = errors.New(
		"nexus: constructor second return value must be error",
	)

	// ErrDuplicateServiceDeclaration indicates that a default implementation has
	// already been declared for the same interface contract.
	ErrDuplicateServiceDeclaration = errors.New(
		"nexus: service is already declared",
	)

	ErrNilValue = errors.New(
		"nexus: value cannot be nil",
	)

	ErrDuplicateValueDeclaration = errors.New(
		"nexus: value already declared",
	)

	// ErrServiceNotDeclared indicates that Nexus could not find a default service
	// declaration for an interface contract.
	ErrServiceNotDeclared = errors.New(
		"nexus: service is not declared",
	)

	// ErrDependencyNotDeclared indicates that a constructor dependency could not
	// be resolved as either a default service or a registered value.
	ErrDependencyNotDeclared = errors.New(
		"nexus: dependency is not declared",
	)

	// ErrConstructorReturnedNil indicates that a constructor completed without an
	// error but returned a nil service implementation.
	ErrConstructorReturnedNil = errors.New(
		"nexus: constructor returned nil service",
	)

	ErrNilContractFunction = errors.New("nexus: nil contract function not allowed")

	// ErrCircularDependency indicates that a service depends on itself directly
	// or indirectly through one or more declared services.
	ErrCircularDependency = errors.New(
		"nexus: circular dependency detected",
	)

	// ErrInvalidServiceName indicates that a named service declaration or lookup
	// received an empty name.
	ErrInvalidServiceName = errors.New(
		"nexus: service name cannot be empty",
	)

	// ErrNamedContractRequiresGetNamed indicates that Get or MustGet received a
	// named contract. Named services must be resolved through GetNamed or
	// MustGetNamed.
	ErrNamedContractRequiresGetNamed = errors.New(
		"nexus: named contract requires GetNamed",
	)

	ErrInvalidGroupName = errors.New("nexus: group name must not be empty")

	ErrDuplicateGroupDeclaration = errors.New(
		"nexus: duplicate group service declaration",
	)
)
