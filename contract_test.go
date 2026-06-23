package nexus

import (
	"errors"
	"reflect"
	"testing"
)

type testService interface {
	Run() error
}

type testConcreteService struct{}

func (testConcreteService) Run() error {
	return nil
}

func TestContractOfStoresInterfaceType(t *testing.T) {
	contract := ContractOf[testService]()

	expectedType := reflect.TypeOf((*testService)(nil)).Elem()

	if contract.typ != expectedType {
		t.Fatalf(
			"contract type mismatch: got %v, want %v",
			contract.typ,
			expectedType,
		)
	}
}

func TestContractOfPanicsForStruct(t *testing.T) {
	assertPanics(t, func() {
		ContractOf[testConcreteService]()
	})
}

func TestContractOfPanicsForPointerToStruct(t *testing.T) {
	assertPanics(t, func() {
		ContractOf[*testConcreteService]()
	})
}

func TestContractOfPanicsForBuiltinType(t *testing.T) {
	assertPanics(t, func() {
		ContractOf[string]()
	})
}

func TestContractOfPanicWrapsExpectedError(t *testing.T) {
	recovered := assertPanics(t, func() {
		ContractOf[testConcreteService]()
	})

	err, ok := recovered.(error)
	if !ok {
		t.Fatalf("panic value should be an error, got %T", recovered)
	}

	if !errors.Is(err, ErrContractMustBeInterface) {
		t.Fatalf(
			"expected panic to wrap %v, got %v",
			ErrContractMustBeInterface,
			err,
		)
	}
}

func assertPanics(t *testing.T, fn func()) any {
	t.Helper()

	var recovered any

	func() {
		defer func() {
			recovered = recover()
		}()

		fn()
	}()

	if recovered == nil {
		t.Fatal("expected panic, got none")
	}

	return recovered
}
