package nexus

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

type lookupService interface {
	Name() string
}

type lookupServiceImpl struct {
	name string
}

func (s *lookupServiceImpl) Name() string {
	return s.name
}

type anotherLookupService interface {
	ID() string
}

type anotherLookupServiceImpl struct {
	id string
}

func (s *anotherLookupServiceImpl) ID() string {
	return s.id
}

func OrdersLookupContract() Contract[lookupService] {
	return NamedContractOf[lookupService]("orders")
}

func AuditLookupContract() Contract[lookupService] {
	return NamedContractOf[lookupService]("audit")
}

func DefaultLookupContract() Contract[lookupService] {
	return ContractOf[lookupService]()
}

func AnotherOrdersLookupContract() Contract[anotherLookupService] {
	return NamedContractOf[anotherLookupService]("orders")
}

func TestGetNamedResolvesNamedService(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	err := DeclareNamed("orders", func() lookupService {
		return &lookupServiceImpl{
			name: "orders",
		}
	})
	if err != nil {
		t.Fatalf("DeclareNamed() error = %v", err)
	}

	service, err := GetNamed(OrdersLookupContract)
	if err != nil {
		t.Fatalf("GetNamed() error = %v", err)
	}

	if got := service.Name(); got != "orders" {
		t.Fatalf("service.Name() = %q, want %q", got, "orders")
	}
}

func TestGetNamedResolvesMultipleNamedServicesForSameInterface(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := DeclareNamed("orders", func() lookupService {
		return &lookupServiceImpl{name: "orders"}
	}); err != nil {
		t.Fatalf("DeclareNamed(orders) error = %v", err)
	}

	if err := DeclareNamed("audit", func() lookupService {
		return &lookupServiceImpl{name: "audit"}
	}); err != nil {
		t.Fatalf("DeclareNamed(audit) error = %v", err)
	}

	orders, err := GetNamed(OrdersLookupContract)
	if err != nil {
		t.Fatalf("GetNamed(orders) error = %v", err)
	}

	audit, err := GetNamed(AuditLookupContract)
	if err != nil {
		t.Fatalf("GetNamed(audit) error = %v", err)
	}

	if got := orders.Name(); got != "orders" {
		t.Fatalf("orders.Name() = %q, want %q", got, "orders")
	}

	if got := audit.Name(); got != "audit" {
		t.Fatalf("audit.Name() = %q, want %q", got, "audit")
	}

	if orders == audit {
		t.Fatal("named lookups returned the same instance")
	}
}

func TestGetNamedAllowsDefaultAndNamedServiceForSameInterface(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := Declare(func() lookupService {
		return &lookupServiceImpl{name: "default"}
	}); err != nil {
		t.Fatalf("Declare() error = %v", err)
	}

	if err := DeclareNamed("orders", func() lookupService {
		return &lookupServiceImpl{name: "orders"}
	}); err != nil {
		t.Fatalf("DeclareNamed() error = %v", err)
	}

	defaultService, err := Get(DefaultLookupContract)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	ordersService, err := GetNamed(OrdersLookupContract)
	if err != nil {
		t.Fatalf("GetNamed() error = %v", err)
	}

	if got := defaultService.Name(); got != "default" {
		t.Fatalf("default service name = %q, want %q", got, "default")
	}

	if got := ordersService.Name(); got != "orders" {
		t.Fatalf("orders service name = %q, want %q", got, "orders")
	}

	if defaultService == ordersService {
		t.Fatal("default and named services returned the same instance")
	}
}

func TestDeclareNamedRejectsDuplicateInterfaceAndName(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := DeclareNamed("orders", func() lookupService {
		return &lookupServiceImpl{name: "first"}
	}); err != nil {
		t.Fatalf("first DeclareNamed() error = %v", err)
	}

	err := DeclareNamed("orders", func() lookupService {
		return &lookupServiceImpl{name: "second"}
	})

	if !errors.Is(err, ErrDuplicateServiceDeclaration) {
		t.Fatalf(
			"expected ErrDuplicateServiceDeclaration, got %v",
			err,
		)
	}
}

func TestDeclareNamedAllowsSameNameForDifferentInterfaces(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := DeclareNamed("orders", func() lookupService {
		return &lookupServiceImpl{name: "lookup-orders"}
	}); err != nil {
		t.Fatalf("DeclareNamed(lookupService) error = %v", err)
	}

	if err := DeclareNamed("orders", func() anotherLookupService {
		return &anotherLookupServiceImpl{id: "another-orders"}
	}); err != nil {
		t.Fatalf("DeclareNamed(anotherLookupService) error = %v", err)
	}

	first, err := GetNamed(OrdersLookupContract)
	if err != nil {
		t.Fatalf("GetNamed(lookupService) error = %v", err)
	}

	second, err := GetNamed(AnotherOrdersLookupContract)
	if err != nil {
		t.Fatalf("GetNamed(anotherLookupService) error = %v", err)
	}

	if got := first.Name(); got != "lookup-orders" {
		t.Fatalf("first.Name() = %q, want %q", got, "lookup-orders")
	}

	if got := second.ID(); got != "another-orders" {
		t.Fatalf("second.ID() = %q, want %q", got, "another-orders")
	}
}

func TestDeclareNamedRejectsEmptyName(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	err := DeclareNamed("", func() lookupService {
		return &lookupServiceImpl{}
	})

	if !errors.Is(err, ErrInvalidServiceName) {
		t.Fatalf("expected ErrInvalidServiceName, got %v", err)
	}
}

func TestGetNamedRejectsUnnamedContract(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	_, err := GetNamed(DefaultLookupContract)

	if !errors.Is(err, ErrInvalidServiceName) {
		t.Fatalf("expected ErrInvalidServiceName, got %v", err)
	}
}

func TestGetRejectsNamedContract(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	_, err := Get(OrdersLookupContract)

	if !errors.Is(err, ErrNamedContractRequiresGetNamed) {
		t.Fatalf(
			"expected ErrNamedContractRequiresGetNamed, got %v",
			err,
		)
	}
}

func TestGetNamedRejectsNilContractFunction(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	_, err := GetNamed[lookupService](nil)

	if !errors.Is(err, ErrNilContractFunction) {
		t.Fatalf(
			"expected ErrNilContractFunction, got %v",
			err,
		)
	}
}

func TestGetNamedReturnsErrorForMissingNamedService(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	_, err := GetNamed(OrdersLookupContract)

	if !errors.Is(err, ErrServiceNotDeclared) {
		t.Fatalf(
			"expected ErrServiceNotDeclared, got %v",
			err,
		)
	}
}

func TestGetNamedCachesNamedServiceIndependentlyFromDefault(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	var defaultCalls atomic.Int32
	var ordersCalls atomic.Int32

	if err := Declare(func() lookupService {
		defaultCalls.Add(1)

		return &lookupServiceImpl{name: "default"}
	}); err != nil {
		t.Fatalf("Declare() error = %v", err)
	}

	if err := DeclareNamed("orders", func() lookupService {
		ordersCalls.Add(1)

		return &lookupServiceImpl{name: "orders"}
	}); err != nil {
		t.Fatalf("DeclareNamed() error = %v", err)
	}

	if _, err := Get(DefaultLookupContract); err != nil {
		t.Fatalf("first Get() error = %v", err)
	}

	if _, err := Get(DefaultLookupContract); err != nil {
		t.Fatalf("second Get() error = %v", err)
	}

	if _, err := GetNamed(OrdersLookupContract); err != nil {
		t.Fatalf("first GetNamed() error = %v", err)
	}

	if _, err := GetNamed(OrdersLookupContract); err != nil {
		t.Fatalf("second GetNamed() error = %v", err)
	}

	if got := defaultCalls.Load(); got != 1 {
		t.Fatalf("default constructor calls = %d, want 1", got)
	}

	if got := ordersCalls.Load(); got != 1 {
		t.Fatalf("orders constructor calls = %d, want 1", got)
	}
}

func TestGetNamedConstructsNamedSingletonOnlyOnceConcurrently(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	var calls atomic.Int32

	started := make(chan struct{})
	release := make(chan struct{})

	err := DeclareNamed("orders", func() lookupService {
		calls.Add(1)

		close(started)
		<-release

		return &lookupServiceImpl{name: "orders"}
	})
	if err != nil {
		t.Fatalf("DeclareNamed() error = %v", err)
	}

	const goroutines = 20

	results := make(chan lookupService, goroutines)
	errs := make(chan error, goroutines)

	var waitGroup sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			service, err := GetNamed(OrdersLookupContract)
			if err != nil {
				errs <- err
				return
			}

			results <- service
		}()
	}

	<-started
	close(release)

	waitGroup.Wait()
	close(results)
	close(errs)

	for err := range errs {
		t.Fatalf("GetNamed() error = %v", err)
	}

	var first lookupService

	for service := range results {
		if first == nil {
			first = service
			continue
		}

		if service != first {
			t.Fatal(
				"concurrent GetNamed() calls returned different singleton instances",
			)
		}
	}

	if got := calls.Load(); got != 1 {
		t.Fatalf("constructor calls = %d, want 1", got)
	}
}

func TestMustGetNamedResolvesNamedService(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := DeclareNamed("orders", func() lookupService {
		return &lookupServiceImpl{name: "orders"}
	}); err != nil {
		t.Fatalf("DeclareNamed() error = %v", err)
	}

	service := MustGetNamed(OrdersLookupContract)

	if got := service.Name(); got != "orders" {
		t.Fatalf("service.Name() = %q, want %q", got, "orders")
	}
}

func TestMustGetNamedPanicsForMissingNamedService(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	defer func() {
		recovered := recover()

		if recovered == nil {
			t.Fatal("expected MustGetNamed() to panic")
		}

		err, ok := recovered.(error)
		if !ok {
			t.Fatalf("panic value should be an error, got %T", recovered)
		}

		if !errors.Is(err, ErrServiceNotDeclared) {
			t.Fatalf(
				"expected panic to wrap ErrServiceNotDeclared, got %v",
				err,
			)
		}
	}()

	MustGetNamed(OrdersLookupContract)
}

func TestMustGetPanicsForNamedContract(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	defer func() {
		recovered := recover()

		if recovered == nil {
			t.Fatal("expected MustGet() to panic")
		}

		err, ok := recovered.(error)
		if !ok {
			t.Fatalf("panic value should be an error, got %T", recovered)
		}

		if !errors.Is(err, ErrNamedContractRequiresGetNamed) {
			t.Fatalf(
				"expected panic to wrap ErrNamedContractRequiresGetNamed, got %v",
				err,
			)
		}
	}()

	MustGet(OrdersLookupContract)
}
