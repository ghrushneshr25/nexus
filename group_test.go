package nexus

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

type groupHandler interface {
	Name() string
}

type groupHandlerImpl struct {
	name string
}

func (h *groupHandlerImpl) Name() string {
	return h.name
}

type anotherGroupHandler interface {
	ID() string
}

type anotherGroupHandlerImpl struct {
	id string
}

func (h *anotherGroupHandlerImpl) ID() string {
	return h.id
}

type groupDependency interface {
	Value() string
}

type groupDependencyImpl struct{}

func (groupDependencyImpl) Value() string {
	return "dependency"
}

type groupHandlerWithDependency struct {
	dependency groupDependency
	name       string
}

func (h *groupHandlerWithDependency) Name() string {
	return h.name + ":" + h.dependency.Value()
}

type groupDefaultHandler struct{}

func (groupDefaultHandler) Name() string {
	return "default"
}

func TestGetGroupResolvesMembersInDeclarationOrder(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := DeclareGroup("handlers", func() groupHandler {
		return &groupHandlerImpl{name: "first"}
	}); err != nil {
		t.Fatalf("first DeclareGroup() error = %v", err)
	}

	if err := DeclareGroup("handlers", func() groupHandler {
		return &groupHandlerImpl{name: "second"}
	}); err != nil {
		t.Fatalf("second DeclareGroup() error = %v", err)
	}

	if err := DeclareGroup("handlers", func() groupHandler {
		return &groupHandlerImpl{name: "third"}
	}); err != nil {
		t.Fatalf("third DeclareGroup() error = %v", err)
	}

	handlers, err := GetGroup[groupHandler]("handlers")
	if err != nil {
		t.Fatalf("GetGroup() error = %v", err)
	}

	if got, want := len(handlers), 3; got != want {
		t.Fatalf("len(handlers) = %d, want %d", got, want)
	}

	wantNames := []string{"first", "second", "third"}

	for index, handler := range handlers {
		if got := handler.Name(); got != wantNames[index] {
			t.Fatalf(
				"handlers[%d].Name() = %q, want %q",
				index,
				got,
				wantNames[index],
			)
		}
	}
}

func TestGetGroupAllowsSameInterfaceInDifferentGroups(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := DeclareGroup("request-handlers", func() groupHandler {
		return &groupHandlerImpl{name: "request"}
	}); err != nil {
		t.Fatalf("DeclareGroup(request-handlers) error = %v", err)
	}

	if err := DeclareGroup("event-handlers", func() groupHandler {
		return &groupHandlerImpl{name: "event"}
	}); err != nil {
		t.Fatalf("DeclareGroup(event-handlers) error = %v", err)
	}

	requestHandlers, err := GetGroup[groupHandler]("request-handlers")
	if err != nil {
		t.Fatalf("GetGroup(request-handlers) error = %v", err)
	}

	eventHandlers, err := GetGroup[groupHandler]("event-handlers")
	if err != nil {
		t.Fatalf("GetGroup(event-handlers) error = %v", err)
	}

	if got, want := len(requestHandlers), 1; got != want {
		t.Fatalf("len(requestHandlers) = %d, want %d", got, want)
	}

	if got, want := len(eventHandlers), 1; got != want {
		t.Fatalf("len(eventHandlers) = %d, want %d", got, want)
	}

	if got, want := requestHandlers[0].Name(), "request"; got != want {
		t.Fatalf("request handler name = %q, want %q", got, want)
	}

	if got, want := eventHandlers[0].Name(), "event"; got != want {
		t.Fatalf("event handler name = %q, want %q", got, want)
	}
}

func TestGetGroupAllowsSameGroupNameForDifferentInterfaces(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := DeclareGroup("handlers", func() groupHandler {
		return &groupHandlerImpl{name: "handler"}
	}); err != nil {
		t.Fatalf("DeclareGroup(groupHandler) error = %v", err)
	}

	if err := DeclareGroup("handlers", func() anotherGroupHandler {
		return &anotherGroupHandlerImpl{id: "another"}
	}); err != nil {
		t.Fatalf("DeclareGroup(anotherGroupHandler) error = %v", err)
	}

	handlers, err := GetGroup[groupHandler]("handlers")
	if err != nil {
		t.Fatalf("GetGroup(groupHandler) error = %v", err)
	}

	anotherHandlers, err := GetGroup[anotherGroupHandler]("handlers")
	if err != nil {
		t.Fatalf("GetGroup(anotherGroupHandler) error = %v", err)
	}

	if got, want := handlers[0].Name(), "handler"; got != want {
		t.Fatalf("handler.Name() = %q, want %q", got, want)
	}

	if got, want := anotherHandlers[0].ID(), "another"; got != want {
		t.Fatalf("anotherHandler.ID() = %q, want %q", got, want)
	}
}

func TestGetGroupCachesEachMemberIndependently(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	var firstCalls atomic.Int32
	var secondCalls atomic.Int32

	if err := DeclareGroup("handlers", func() groupHandler {
		firstCalls.Add(1)
		return &groupHandlerImpl{name: "first"}
	}); err != nil {
		t.Fatalf("first DeclareGroup() error = %v", err)
	}

	if err := DeclareGroup("handlers", func() groupHandler {
		secondCalls.Add(1)
		return &groupHandlerImpl{name: "second"}
	}); err != nil {
		t.Fatalf("second DeclareGroup() error = %v", err)
	}

	first, err := GetGroup[groupHandler]("handlers")
	if err != nil {
		t.Fatalf("first GetGroup() error = %v", err)
	}

	second, err := GetGroup[groupHandler]("handlers")
	if err != nil {
		t.Fatalf("second GetGroup() error = %v", err)
	}

	if got, want := firstCalls.Load(), int32(1); got != want {
		t.Fatalf("first constructor calls = %d, want %d", got, want)
	}

	if got, want := secondCalls.Load(), int32(1); got != want {
		t.Fatalf("second constructor calls = %d, want %d", got, want)
	}

	if first[0] != second[0] {
		t.Fatal("first group member was not cached as a singleton")
	}

	if first[1] != second[1] {
		t.Fatal("second group member was not cached as a singleton")
	}
}

func TestGetGroupResolvesDependencies(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := Declare(func() groupDependency {
		return groupDependencyImpl{}
	}); err != nil {
		t.Fatalf("Declare(groupDependency) error = %v", err)
	}

	if err := DeclareGroup("handlers", func(
		dependency groupDependency,
	) groupHandler {
		return &groupHandlerWithDependency{
			dependency: dependency,
			name:       "handler",
		}
	}); err != nil {
		t.Fatalf("DeclareGroup() error = %v", err)
	}

	handlers, err := GetGroup[groupHandler]("handlers")
	if err != nil {
		t.Fatalf("GetGroup() error = %v", err)
	}

	if got, want := handlers[0].Name(), "handler:dependency"; got != want {
		t.Fatalf("handler.Name() = %q, want %q", got, want)
	}
}

func TestGetGroupDoesNotBecomeDefaultService(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := DeclareGroup("handlers", func() groupHandler {
		return &groupHandlerImpl{name: "group"}
	}); err != nil {
		t.Fatalf("DeclareGroup() error = %v", err)
	}

	_, err := Get(func() Contract[groupHandler] {
		return ContractOf[groupHandler]()
	})

	if !errors.Is(err, ErrServiceNotDeclared) {
		t.Fatalf("expected ErrServiceNotDeclared, got %v", err)
	}
}

func TestGetGroupDoesNotUseDefaultService(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := Declare(func() groupHandler {
		return groupDefaultHandler{}
	}); err != nil {
		t.Fatalf("Declare() error = %v", err)
	}

	if err := DeclareGroup("handlers", func() groupHandler {
		return &groupHandlerImpl{name: "group"}
	}); err != nil {
		t.Fatalf("DeclareGroup() error = %v", err)
	}

	handlers, err := GetGroup[groupHandler]("handlers")
	if err != nil {
		t.Fatalf("GetGroup() error = %v", err)
	}

	if got, want := len(handlers), 1; got != want {
		t.Fatalf("len(handlers) = %d, want %d", got, want)
	}

	if got, want := handlers[0].Name(), "group"; got != want {
		t.Fatalf("handler.Name() = %q, want %q", got, want)
	}
}

func TestDeclareGroupRejectsEmptyGroupName(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	err := DeclareGroup("", func() groupHandler {
		return &groupHandlerImpl{name: "handler"}
	})

	if !errors.Is(err, ErrInvalidGroupName) {
		t.Fatalf("expected ErrInvalidGroupName, got %v", err)
	}
}

func TestGetGroupRejectsEmptyGroupName(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	_, err := GetGroup[groupHandler]("")

	if !errors.Is(err, ErrInvalidGroupName) {
		t.Fatalf("expected ErrInvalidGroupName, got %v", err)
	}
}

func TestGetGroupReturnsErrorForMissingGroup(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	_, err := GetGroup[groupHandler]("missing")

	if !errors.Is(err, ErrServiceNotDeclared) {
		t.Fatalf("expected ErrServiceNotDeclared, got %v", err)
	}
}

func TestDeclareGroupAllowsDifferentConstructorsForSameInterfaceAndGroup(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := DeclareGroup("handlers", func() groupHandler {
		return &groupHandlerImpl{name: "first"}
	}); err != nil {
		t.Fatalf("first DeclareGroup() error = %v", err)
	}

	if err := DeclareGroup("handlers", func() groupHandler {
		return &groupHandlerImpl{name: "second"}
	}); err != nil {
		t.Fatalf("second DeclareGroup() error = %v", err)
	}

	handlers, err := GetGroup[groupHandler]("handlers")
	if err != nil {
		t.Fatalf("GetGroup() error = %v", err)
	}

	if got, want := len(handlers), 2; got != want {
		t.Fatalf("len(handlers) = %d, want %d", got, want)
	}
}

func TestGetGroupRetriesFailedMemberConstruction(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	var calls atomic.Int32

	if err := DeclareGroup("handlers", func() (groupHandler, error) {
		if calls.Add(1) == 1 {
			return nil, errors.New("temporary failure")
		}

		return &groupHandlerImpl{name: "recovered"}, nil
	}); err != nil {
		t.Fatalf("DeclareGroup() error = %v", err)
	}

	_, err := GetGroup[groupHandler]("handlers")
	if err == nil {
		t.Fatal("expected first GetGroup() to fail")
	}

	handlers, err := GetGroup[groupHandler]("handlers")
	if err != nil {
		t.Fatalf("second GetGroup() error = %v", err)
	}

	if got, want := handlers[0].Name(), "recovered"; got != want {
		t.Fatalf("handler.Name() = %q, want %q", got, want)
	}

	if got, want := calls.Load(), int32(2); got != want {
		t.Fatalf("constructor calls = %d, want %d", got, want)
	}
}

func TestMustGetGroupPanicsWhenGroupIsMissing(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("MustGetGroup() did not panic")
		}
	}()

	_ = MustGetGroup[groupHandler]("missing")
}

func TestMustDeclareGroupPanicsForInvalidGroupName(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("MustDeclareGroup() did not panic")
		}
	}()

	MustDeclareGroup("", func() groupHandler {
		return &groupHandlerImpl{name: "handler"}
	})
}

func TestGetGroupConstructsMembersOnlyOnceConcurrently(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	var calls atomic.Int32

	started := make(chan struct{})
	release := make(chan struct{})

	if err := DeclareGroup("handlers", func() groupHandler {
		calls.Add(1)
		close(started)
		<-release

		return &groupHandlerImpl{name: "only"}
	}); err != nil {
		t.Fatalf("DeclareGroup() error = %v", err)
	}

	const goroutines = 20

	results := make(chan []groupHandler, goroutines)
	errs := make(chan error, goroutines)

	var waitGroup sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			handlers, err := GetGroup[groupHandler]("handlers")
			if err != nil {
				errs <- err
				return
			}

			results <- handlers
		}()
	}

	<-started
	close(release)

	waitGroup.Wait()
	close(results)
	close(errs)

	for err := range errs {
		t.Fatalf("GetGroup() error = %v", err)
	}

	var first groupHandler

	for handlers := range results {
		if got, want := len(handlers), 1; got != want {
			t.Fatalf("len(handlers) = %d, want %d", got, want)
		}

		if first == nil {
			first = handlers[0]
			continue
		}

		if handlers[0] != first {
			t.Fatal("group member was not shared across callers")
		}
	}

	if got, want := calls.Load(), int32(1); got != want {
		t.Fatalf("constructor calls = %d, want %d", got, want)
	}
}
