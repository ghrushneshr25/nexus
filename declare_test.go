package nexus

import (
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
)

// declaredService is used only to test valid interface-returning constructors.
type declaredService interface {
	Run() error
}

// anotherDeclaredService lets tests verify distinct interface declarations.
type anotherDeclaredService interface {
	Stop() error
}

// concreteService is intentionally not an interface and is used to verify that
// Nexus rejects concrete constructor return types.
type concreteService struct{}

func (concreteService) Run() error {
	return nil
}

func TestDeclareStoresValidConstructor(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	err := Declare(func() declaredService {
		return nil
	})
	if err != nil {
		t.Fatalf("Declare() error = %v", err)
	}

	contractType := reflect.TypeOf((*declaredService)(nil)).Elem()
	key := serviceKey{
		contract: contractType,
	}

	globalRegistry.mu.RLock()
	registered, exists := globalRegistry.declarations[key]
	globalRegistry.mu.RUnlock()

	if !exists {
		t.Fatal("expected declaration to be stored")
	}

	if registered.returnsError {
		t.Fatal("returnsError = true, want false")
	}

	if len(registered.dependencies) != 0 {
		t.Fatalf(
			"dependency count = %d, want 0",
			len(registered.dependencies),
		)
	}
}

func TestDeclareStoresErrorReturningConstructor(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	err := Declare(func() (declaredService, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatalf("Declare() error = %v", err)
	}

	contractType := reflect.TypeOf((*declaredService)(nil)).Elem()
	key := serviceKey{
		contract: contractType,
	}

	globalRegistry.mu.RLock()
	registered, exists := globalRegistry.declarations[key]
	globalRegistry.mu.RUnlock()

	if !exists {
		t.Fatal("expected declaration to be stored")
	}

	if !registered.returnsError {
		t.Fatal("returnsError = false, want true")
	}
}

func TestDeclareStoresDependenciesInOrder(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	err := Declare(func(
		first declaredService,
		second anotherDeclaredService,
	) declaredService {
		return nil
	})
	if err != nil {
		t.Fatalf("Declare() error = %v", err)
	}

	contractType := reflect.TypeOf((*declaredService)(nil)).Elem()
	key := serviceKey{
		contract: contractType,
	}

	globalRegistry.mu.RLock()
	registered, exists := globalRegistry.declarations[key]
	globalRegistry.mu.RUnlock()

	if !exists {
		t.Fatal("expected declaration to be stored")
	}

	firstType := reflect.TypeOf((*declaredService)(nil)).Elem()
	secondType := reflect.TypeOf((*anotherDeclaredService)(nil)).Elem()

	if len(registered.dependencies) != 2 {
		t.Fatalf(
			"dependency count = %d, want 2",
			len(registered.dependencies),
		)
	}

	if registered.dependencies[0] != firstType {
		t.Fatalf(
			"first dependency = %v, want %v",
			registered.dependencies[0],
			firstType,
		)
	}

	if registered.dependencies[1] != secondType {
		t.Fatalf(
			"second dependency = %v, want %v",
			registered.dependencies[1],
			secondType,
		)
	}
}

func TestDeclareRejectsNonFunction(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	err := Declare("not a constructor")

	if !errors.Is(err, ErrConstructorMustBeFunction) {
		t.Fatalf(
			"expected ErrConstructorMustBeFunction, got %v",
			err,
		)
	}
}

func TestDeclareRejectsNoReturnValues(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	err := Declare(func() {})

	if !errors.Is(err, ErrInvalidConstructor) {
		t.Fatalf(
			"expected ErrInvalidConstructor, got %v",
			err,
		)
	}
}

func TestDeclareRejectsThreeReturnValues(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	err := Declare(func() (declaredService, error, string) {
		return nil, nil, ""
	})

	if !errors.Is(err, ErrInvalidConstructor) {
		t.Fatalf(
			"expected ErrInvalidConstructor, got %v",
			err,
		)
	}
}

func TestDeclareRejectsConcreteReturnType(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	err := Declare(func() concreteService {
		return concreteService{}
	})

	if !errors.Is(err, ErrServiceMustBeInterface) {
		t.Fatalf(
			"expected ErrServiceMustBeInterface, got %v",
			err,
		)
	}
}

func TestDeclareRejectsInvalidSecondReturnType(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	err := Declare(func() (declaredService, string) {
		return nil, ""
	})

	if !errors.Is(err, ErrInvalidConstructorError) {
		t.Fatalf(
			"expected ErrInvalidConstructorError, got %v",
			err,
		)
	}
}

func TestDeclareRejectsDuplicateDefaultService(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	firstConstructor := func() declaredService {
		return nil
	}

	secondConstructor := func() declaredService {
		return nil
	}

	if err := Declare(firstConstructor); err != nil {
		t.Fatalf("first Declare() error = %v", err)
	}

	err := Declare(secondConstructor)

	if !errors.Is(err, ErrDuplicateServiceDeclaration) {
		t.Fatalf(
			"expected ErrDuplicateServiceDeclaration, got %v",
			err,
		)
	}
}

func TestDeclareDoesNotInvokeConstructor(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	var calls atomic.Int32

	err := Declare(func() declaredService {
		calls.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("Declare() error = %v", err)
	}

	if got := calls.Load(); got != 0 {
		t.Fatalf(
			"constructor calls = %d, want 0",
			got,
		)
	}
}
