package nexus

import (
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
)

type resolvedService interface {
	Run() string
}

type resolvedServiceImpl struct {
	value string
}

func (s *resolvedServiceImpl) Run() string {
	return s.value
}

func ResolvedServiceContract() Contract[resolvedService] {
	return ContractOf[resolvedService]()
}

type repository interface {
	Name() string
}

type repositoryImpl struct {
	name string
}

func (r *repositoryImpl) Name() string {
	return r.name
}

func RepositoryContract() Contract[repository] {
	return ContractOf[repository]()
}

type serviceWithRepository interface {
	RepositoryName() string
}

type serviceWithRepositoryImpl struct {
	repository repository
}

func (s *serviceWithRepositoryImpl) RepositoryName() string {
	return s.repository.Name()
}

func ServiceWithRepositoryContract() Contract[serviceWithRepository] {
	return ContractOf[serviceWithRepository]()
}

type missingDependency interface {
	Value() string
}

type serviceWithMissingDependency interface {
	Run() string
}

type serviceWithMissingDependencyImpl struct {
	dependency missingDependency
}

func (s *serviceWithMissingDependencyImpl) Run() string {
	return s.dependency.Value()
}

func ServiceWithMissingDependencyContract() Contract[serviceWithMissingDependency] {
	return ContractOf[serviceWithMissingDependency]()
}

type resolverConfig struct {
	Prefix string
}

type serviceWithConfig interface {
	Run() string
}

type serviceWithConfigImpl struct {
	config resolverConfig
}

func (s *serviceWithConfigImpl) Run() string {
	return s.config.Prefix
}

func ServiceWithConfigContract() Contract[serviceWithConfig] {
	return ContractOf[serviceWithConfig]()
}

func TestGetRejectsNilContractFunction(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	_, err := Get[resolvedService](nil)

	if !errors.Is(err, ErrNilContractFunction) {
		t.Fatalf(
			"expected ErrNilContractFunction, got %v",
			err,
		)
	}
}

func TestMustGetPanicsForNilContractFunction(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	defer func() {
		recovered := recover()

		if recovered == nil {
			t.Fatal("expected MustGet to panic")
		}

		err, ok := recovered.(error)
		if !ok {
			t.Fatalf("panic value should be an error, got %T", recovered)
		}

		if !errors.Is(err, ErrNilContractFunction) {
			t.Fatalf(
				"expected panic to wrap ErrNilContractFunction, got %v",
				err,
			)
		}
	}()

	MustGet[resolvedService](nil)
}

func TestGetReturnsErrorForUndeclaredService(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	_, err := Get(ResolvedServiceContract)

	if !errors.Is(err, ErrServiceNotDeclared) {
		t.Fatalf(
			"expected ErrServiceNotDeclared, got %v",
			err,
		)
	}
}

func TestGetReturnsCachedInstance(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	instance := &resolvedServiceImpl{
		value: "cached",
	}

	contractType := reflect.TypeOf((*resolvedService)(nil)).Elem()

	globalRegistry.mu.Lock()
	globalRegistry.instances[serviceKey{
		contract: contractType,
	}] = reflect.ValueOf(instance)
	globalRegistry.mu.Unlock()

	service, err := Get(ResolvedServiceContract)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if service != instance {
		t.Fatal("Get() did not return the cached instance")
	}
}

func TestGetInvokesNoDependencyConstructor(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	var calls atomic.Int32

	err := Declare(func() resolvedService {
		calls.Add(1)

		return &resolvedServiceImpl{
			value: "created",
		}
	})
	if err != nil {
		t.Fatalf("Declare() error = %v", err)
	}

	service, err := Get(ResolvedServiceContract)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got := service.Run(); got != "created" {
		t.Fatalf("service.Run() = %q, want %q", got, "created")
	}

	if got := calls.Load(); got != 1 {
		t.Fatalf("constructor calls = %d, want 1", got)
	}
}

func TestGetCachesSuccessfulConstructorResult(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	var calls atomic.Int32

	err := Declare(func() resolvedService {
		calls.Add(1)

		return &resolvedServiceImpl{
			value: "singleton",
		}
	})
	if err != nil {
		t.Fatalf("Declare() error = %v", err)
	}

	first, err := Get(ResolvedServiceContract)
	if err != nil {
		t.Fatalf("first Get() error = %v", err)
	}

	second, err := Get(ResolvedServiceContract)
	if err != nil {
		t.Fatalf("second Get() error = %v", err)
	}

	if first != second {
		t.Fatal("Get() returned different singleton instances")
	}

	if got := calls.Load(); got != 1 {
		t.Fatalf("constructor calls = %d, want 1", got)
	}
}

func TestGetReturnsConstructorErrorAndDoesNotCache(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	expectedErr := errors.New("constructor failed")
	var calls atomic.Int32

	err := Declare(func() (resolvedService, error) {
		calls.Add(1)

		return nil, expectedErr
	})
	if err != nil {
		t.Fatalf("Declare() error = %v", err)
	}

	_, err = Get(ResolvedServiceContract)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected constructor error %v, got %v", expectedErr, err)
	}

	_, err = Get(ResolvedServiceContract)

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected constructor error on second call %v, got %v",
			expectedErr,
			err,
		)
	}

	if got := calls.Load(); got != 2 {
		t.Fatalf(
			"constructor calls = %d, want 2 because failures are not cached",
			got,
		)
	}
}

func TestGetRejectsNilConstructorResult(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	err := Declare(func() resolvedService {
		return nil
	})
	if err != nil {
		t.Fatalf("Declare() error = %v", err)
	}

	_, err = Get(ResolvedServiceContract)

	if !errors.Is(err, ErrConstructorReturnedNil) {
		t.Fatalf(
			"expected ErrConstructorReturnedNil, got %v",
			err,
		)
	}
}

func TestGetResolvesInterfaceDependency(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	var repositoryCalls atomic.Int32
	var serviceCalls atomic.Int32

	err := Declare(func() repository {
		repositoryCalls.Add(1)

		return &repositoryImpl{
			name: "primary-repository",
		}
	})
	if err != nil {
		t.Fatalf("Declare(repository) error = %v", err)
	}

	err = Declare(func(repo repository) serviceWithRepository {
		serviceCalls.Add(1)

		return &serviceWithRepositoryImpl{
			repository: repo,
		}
	})
	if err != nil {
		t.Fatalf("Declare(service) error = %v", err)
	}

	service, err := Get(ServiceWithRepositoryContract)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got := service.RepositoryName(); got != "primary-repository" {
		t.Fatalf(
			"service.RepositoryName() = %q, want %q",
			got,
			"primary-repository",
		)
	}

	if got := repositoryCalls.Load(); got != 1 {
		t.Fatalf("repository constructor calls = %d, want 1", got)
	}

	if got := serviceCalls.Load(); got != 1 {
		t.Fatalf("service constructor calls = %d, want 1", got)
	}
}

func TestGetResolvesConcreteValueDependency(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	config := resolverConfig{
		Prefix: "configured",
	}

	if err := DeclareValue(config); err != nil {
		t.Fatalf("DeclareValue() error = %v", err)
	}

	err := Declare(func(cfg resolverConfig) serviceWithConfig {
		return &serviceWithConfigImpl{
			config: cfg,
		}
	})
	if err != nil {
		t.Fatalf("Declare() error = %v", err)
	}

	service, err := Get(ServiceWithConfigContract)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got := service.Run(); got != config.Prefix {
		t.Fatalf("service.Run() = %q, want %q", got, config.Prefix)
	}
}

func TestGetReturnsErrorForMissingInterfaceDependency(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	err := Declare(func(
		dependency missingDependency,
	) serviceWithMissingDependency {
		return &serviceWithMissingDependencyImpl{
			dependency: dependency,
		}
	})
	if err != nil {
		t.Fatalf("Declare() error = %v", err)
	}

	_, err = Get(ServiceWithMissingDependencyContract)

	if !errors.Is(err, ErrDependencyNotDeclared) {
		t.Fatalf(
			"expected ErrDependencyNotDeclared, got %v",
			err,
		)
	}

	if !errors.Is(err, ErrServiceNotDeclared) {
		t.Fatalf(
			"expected wrapped ErrServiceNotDeclared, got %v",
			err,
		)
	}
}

func TestGetReturnsErrorForMissingConcreteValueDependency(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	err := Declare(func(
		config resolverConfig,
	) serviceWithConfig {
		return &serviceWithConfigImpl{
			config: config,
		}
	})
	if err != nil {
		t.Fatalf("Declare() error = %v", err)
	}

	_, err = Get(ServiceWithConfigContract)

	if !errors.Is(err, ErrDependencyNotDeclared) {
		t.Fatalf(
			"expected ErrDependencyNotDeclared, got %v",
			err,
		)
	}
}

func TestGetCachesNestedDependencies(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	var repositoryCalls atomic.Int32
	var serviceCalls atomic.Int32

	err := Declare(func() repository {
		repositoryCalls.Add(1)

		return &repositoryImpl{
			name: "nested",
		}
	})
	if err != nil {
		t.Fatalf("Declare(repository) error = %v", err)
	}

	err = Declare(func(repo repository) serviceWithRepository {
		serviceCalls.Add(1)

		return &serviceWithRepositoryImpl{
			repository: repo,
		}
	})
	if err != nil {
		t.Fatalf("Declare(service) error = %v", err)
	}

	first, err := Get(ServiceWithRepositoryContract)
	if err != nil {
		t.Fatalf("first Get() error = %v", err)
	}

	second, err := Get(ServiceWithRepositoryContract)
	if err != nil {
		t.Fatalf("second Get() error = %v", err)
	}

	if first != second {
		t.Fatal("Get() returned different top-level singleton instances")
	}

	if got := repositoryCalls.Load(); got != 1 {
		t.Fatalf("repository constructor calls = %d, want 1", got)
	}

	if got := serviceCalls.Load(); got != 1 {
		t.Fatalf("service constructor calls = %d, want 1", got)
	}
}

func TestGetConstructsSingletonOnlyOnceConcurrently(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	var calls atomic.Int32

	started := make(chan struct{})
	release := make(chan struct{})

	err := Declare(func() resolvedService {
		calls.Add(1)

		// Signal that construction has started, then block long enough for
		// every test goroutine to attempt resolution concurrently.
		close(started)
		<-release

		return &resolvedServiceImpl{
			value: "concurrent-singleton",
		}
	})
	if err != nil {
		t.Fatalf("Declare() error = %v", err)
	}

	const goroutines = 20

	results := make(chan resolvedService, goroutines)
	errors := make(chan error, goroutines)

	var waitGroup sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		waitGroup.Add(1)

		go func() {
			defer waitGroup.Done()

			service, err := Get(ResolvedServiceContract)
			if err != nil {
				errors <- err
				return
			}

			results <- service
		}()
	}

	<-started
	close(release)

	waitGroup.Wait()
	close(results)
	close(errors)

	for err := range errors {
		t.Fatalf("Get() error = %v", err)
	}

	var first resolvedService

	for service := range results {
		if first == nil {
			first = service
			continue
		}

		if service != first {
			t.Fatal("concurrent Get() calls returned different singleton instances")
		}
	}

	if got := calls.Load(); got != 1 {
		t.Fatalf("constructor calls = %d, want 1", got)
	}
}

type cycleA interface {
	A()
}

type cycleB interface {
	B()
}

type cycleAImpl struct{}

func (*cycleAImpl) A() {}

type cycleBImpl struct{}

func (*cycleBImpl) B() {}

func CycleAContract() Contract[cycleA] {
	return ContractOf[cycleA]()
}

func TestGetDetectsDirectCircularDependency(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	err := Declare(func(self cycleA) cycleA {
		return &cycleAImpl{}
	})
	if err != nil {
		t.Fatalf("Declare() error = %v", err)
	}

	_, err = Get(CycleAContract)

	if !errors.Is(err, ErrCircularDependency) {
		t.Fatalf("expected ErrCircularDependency, got %v", err)
	}
}

func TestGetDetectsIndirectCircularDependency(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := Declare(func(b cycleB) cycleA {
		return &cycleAImpl{}
	}); err != nil {
		t.Fatalf("Declare(cycleA) error = %v", err)
	}

	if err := Declare(func(a cycleA) cycleB {
		return &cycleBImpl{}
	}); err != nil {
		t.Fatalf("Declare(cycleB) error = %v", err)
	}

	_, err := Get(CycleAContract)

	if !errors.Is(err, ErrCircularDependency) {
		t.Fatalf("expected ErrCircularDependency, got %v", err)
	}
}
