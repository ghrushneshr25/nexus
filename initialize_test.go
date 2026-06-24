package nexus

import (
	"errors"
	"sync/atomic"
	"testing"
)

type initializeRepository interface {
	Name() string
}

type initializeRepositoryImpl struct{}

func (initializeRepositoryImpl) Name() string {
	return "repository"
}

type initializeService interface {
	Name() string
}

type initializeServiceImpl struct {
	repository initializeRepository
}

func (s initializeServiceImpl) Name() string {
	if s.repository == nil {
		return "service"
	}

	return s.repository.Name()
}

type initializeNamedService interface {
	Name() string
}

type initializeNamedServiceImpl struct {
	name string
}

func (s initializeNamedServiceImpl) Name() string {
	return s.name
}

type initializeGroupService interface {
	Name() string
}

type initializeGroupServiceImpl struct {
	name string
}

func (s initializeGroupServiceImpl) Name() string {
	return s.name
}

type initializeLifecycleService interface {
	Name() string
}

type initializeLifecycleServiceImpl struct {
	startCalls *atomic.Int32
	stopCalls  *atomic.Int32
}

func (s initializeLifecycleServiceImpl) Name() string {
	return "lifecycle"
}

func (s initializeLifecycleServiceImpl) Start() error {
	s.startCalls.Add(1)

	return nil
}

func (s initializeLifecycleServiceImpl) Stop() error {
	s.stopCalls.Add(1)

	return nil
}

func TestInitializeConstructsDefaultServices(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	var repositoryCalls atomic.Int32
	var serviceCalls atomic.Int32

	if err := Declare(func() initializeRepository {
		repositoryCalls.Add(1)

		return initializeRepositoryImpl{}
	}); err != nil {
		t.Fatalf("Declare(repository) error = %v", err)
	}

	if err := Declare(func(repository initializeRepository) initializeService {
		serviceCalls.Add(1)

		return initializeServiceImpl{
			repository: repository,
		}
	}); err != nil {
		t.Fatalf("Declare(service) error = %v", err)
	}

	if err := Initialize(); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	if got := repositoryCalls.Load(); got != 1 {
		t.Fatalf("repository constructor calls = %d, want 1", got)
	}

	if got := serviceCalls.Load(); got != 1 {
		t.Fatalf("service constructor calls = %d, want 1", got)
	}
}

func TestInitializeConstructsNamedServices(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	var calls atomic.Int32

	if err := DeclareNamed("orders", func() initializeNamedService {
		calls.Add(1)

		return initializeNamedServiceImpl{
			name: "orders",
		}
	}); err != nil {
		t.Fatalf("DeclareNamed() error = %v", err)
	}

	if err := Initialize(); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	if got := calls.Load(); got != 1 {
		t.Fatalf("named constructor calls = %d, want 1", got)
	}
}

func TestInitializeConstructsGroupMembers(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	var firstCalls atomic.Int32
	var secondCalls atomic.Int32

	if err := DeclareGroup("handlers", func() initializeGroupService {
		firstCalls.Add(1)

		return initializeGroupServiceImpl{
			name: "first",
		}
	}); err != nil {
		t.Fatalf("first DeclareGroup() error = %v", err)
	}

	if err := DeclareGroup("handlers", func() initializeGroupService {
		secondCalls.Add(1)

		return initializeGroupServiceImpl{
			name: "second",
		}
	}); err != nil {
		t.Fatalf("second DeclareGroup() error = %v", err)
	}

	if err := Initialize(); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	if got := firstCalls.Load(); got != 1 {
		t.Fatalf("first group constructor calls = %d, want 1", got)
	}

	if got := secondCalls.Load(); got != 1 {
		t.Fatalf("second group constructor calls = %d, want 1", got)
	}
}

func TestInitializeDoesNotConstructServicesTwice(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	var calls atomic.Int32

	if err := Declare(func() initializeService {
		calls.Add(1)

		return initializeServiceImpl{}
	}); err != nil {
		t.Fatalf("Declare() error = %v", err)
	}

	if err := Initialize(); err != nil {
		t.Fatalf("first Initialize() error = %v", err)
	}

	if err := Initialize(); err != nil {
		t.Fatalf("second Initialize() error = %v", err)
	}

	if got := calls.Load(); got != 1 {
		t.Fatalf("constructor calls = %d, want 1", got)
	}
}

func TestInitializeReturnsConstructorError(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	expected := errors.New("constructor failed")

	if err := Declare(func() (initializeService, error) {
		return nil, expected
	}); err != nil {
		t.Fatalf("Declare() error = %v", err)
	}

	err := Initialize()

	if !errors.Is(err, expected) {
		t.Fatalf("expected constructor error, got %v", err)
	}
}

func TestMustInitializePanicsOnConstructorError(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	expected := errors.New("constructor failed")

	if err := Declare(func() (initializeService, error) {
		return nil, expected
	}); err != nil {
		t.Fatalf("Declare() error = %v", err)
	}

	defer func() {
		recovered := recover()

		if recovered == nil {
			t.Fatal("MustInitialize() did not panic")
		}

		err, ok := recovered.(error)
		if !ok {
			t.Fatalf("panic value type = %T, want error", recovered)
		}

		if !errors.Is(err, expected) {
			t.Fatalf("expected constructor error, got %v", err)
		}
	}()

	MustInitialize()
}

func TestInitializeConstructsDependenciesBeforeDependents(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	order := make([]string, 0, 2)

	if err := Declare(func() initializeRepository {
		order = append(order, "repository")

		return initializeRepositoryImpl{}
	}); err != nil {
		t.Fatalf("Declare(repository) error = %v", err)
	}

	if err := Declare(func(repository initializeRepository) initializeService {
		order = append(order, "service")

		return initializeServiceImpl{
			repository: repository,
		}
	}); err != nil {
		t.Fatalf("Declare(service) error = %v", err)
	}

	if err := Initialize(); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	if len(order) != 2 {
		t.Fatalf("construction order length = %d, want 2", len(order))
	}

	if order[0] != "repository" || order[1] != "service" {
		t.Fatalf("construction order = %v, want [repository service]", order)
	}
}

func TestInitializeDoesNotCallLifecycleHooks(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	var startCalls atomic.Int32
	var stopCalls atomic.Int32

	if err := Declare(func() initializeLifecycleService {
		return initializeLifecycleServiceImpl{
			startCalls: &startCalls,
			stopCalls:  &stopCalls,
		}
	}); err != nil {
		t.Fatalf("Declare() error = %v", err)
	}

	if err := Initialize(); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	if got := startCalls.Load(); got != 0 {
		t.Fatalf("Start() calls = %d, want 0", got)
	}

	if got := stopCalls.Load(); got != 0 {
		t.Fatalf("Stop() calls = %d, want 0", got)
	}
}
