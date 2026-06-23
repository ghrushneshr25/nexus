package nexus

import (
	"errors"
	"reflect"
	"testing"
)

type testConfig struct {
	Address string
}

func TestDeclareValueStoresStructValue(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	value := testConfig{
		Address: "localhost",
	}

	if err := DeclareValue(value); err != nil {
		t.Fatalf("DeclareValue() error = %v", err)
	}

	typ := reflect.TypeOf(value)

	globalRegistry.mu.RLock()
	registered, exists := globalRegistry.values[typ]
	globalRegistry.mu.RUnlock()

	if !exists {
		t.Fatal("expected value to be stored")
	}

	if !reflect.DeepEqual(registered.Interface(), value) {
		t.Fatalf(
			"stored value = %#v, want %#v",
			registered.Interface(),
			value,
		)
	}
}

func TestDeclareValueStoresPointerValue(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	value := &testConfig{
		Address: "localhost",
	}

	if err := DeclareValue(value); err != nil {
		t.Fatalf("DeclareValue() error = %v", err)
	}

	typ := reflect.TypeOf(value)

	globalRegistry.mu.RLock()
	registered, exists := globalRegistry.values[typ]
	globalRegistry.mu.RUnlock()

	if !exists {
		t.Fatal("expected pointer value to be stored")
	}

	if registered.Interface() != value {
		t.Fatal("stored pointer does not match original pointer")
	}
}

func TestDeclareValueAllowsStructAndPointerTypes(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	structValue := testConfig{
		Address: "struct",
	}

	pointerValue := &testConfig{
		Address: "pointer",
	}

	if err := DeclareValue(structValue); err != nil {
		t.Fatalf("DeclareValue(struct) error = %v", err)
	}

	if err := DeclareValue(pointerValue); err != nil {
		t.Fatalf("DeclareValue(pointer) error = %v", err)
	}

	structType := reflect.TypeOf(structValue)
	pointerType := reflect.TypeOf(pointerValue)

	globalRegistry.mu.RLock()
	_, structExists := globalRegistry.values[structType]
	_, pointerExists := globalRegistry.values[pointerType]
	globalRegistry.mu.RUnlock()

	if !structExists {
		t.Fatal("expected struct value to be stored")
	}

	if !pointerExists {
		t.Fatal("expected pointer value to be stored")
	}
}

func TestDeclareValueRejectsDuplicateExactType(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := DeclareValue(testConfig{Address: "first"}); err != nil {
		t.Fatalf("first DeclareValue() error = %v", err)
	}

	err := DeclareValue(testConfig{Address: "second"})

	if !errors.Is(err, ErrDuplicateValueDeclaration) {
		t.Fatalf(
			"expected ErrDuplicateValueDeclaration, got %v",
			err,
		)
	}
}

func TestDeclareValueRejectsNil(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	err := DeclareValue(nil)

	if !errors.Is(err, ErrNilValue) {
		t.Fatalf("expected ErrNilValue, got %v", err)
	}
}

func TestMustDeclareValuePanicsForDuplicateType(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	MustDeclareValue(testConfig{Address: "first"})

	defer func() {
		if recover() == nil {
			t.Fatal("expected MustDeclareValue to panic")
		}
	}()

	MustDeclareValue(testConfig{Address: "second"})
}
