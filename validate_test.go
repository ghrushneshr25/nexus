package nexus

import (
	"errors"
	"sync/atomic"
	"testing"
)

type validateRepository interface {
	Find(id string) string
}

type validateRepositoryImpl struct{}

func (validateRepositoryImpl) Find(id string) string {
	return id
}

type validateUserService interface {
	Get(id string) string
}

type validateUserServiceImpl struct {
	repository validateRepository
}

func (s validateUserServiceImpl) Get(id string) string {
	return s.repository.Find(id)
}

type validateNamedService interface {
	Name() string
}

type validateNamedServiceImpl struct {
	name string
}

func (s validateNamedServiceImpl) Name() string {
	return s.name
}

type validateConfig struct {
	Prefix string
}

type validateConfigService interface {
	Value() string
}

type validateConfigServiceImpl struct {
	config validateConfig
}

func (s validateConfigServiceImpl) Value() string {
	return s.config.Prefix
}

type validateMissingDependency interface {
	Run() error
}

type validateMissingConsumer interface {
	Run() error
}

type validateMissingConsumerImpl struct {
	dependency validateMissingDependency
}

func (s validateMissingConsumerImpl) Run() error {
	return s.dependency.Run()
}

type validateDirectCycleA interface {
	A() string
}

type validateDirectCycleB interface {
	B() string
}

type validateDirectCycleAImpl struct {
	dependency validateDirectCycleB
}

func (validateDirectCycleAImpl) A() string {
	return "a"
}

type validateDirectCycleBImpl struct {
	dependency validateDirectCycleA
}

func (validateDirectCycleBImpl) B() string {
	return "b"
}

type validateIndirectCycleA interface {
	A() string
}

type validateIndirectCycleB interface {
	B() string
}

type validateIndirectCycleC interface {
	C() string
}

type validateIndirectCycleAImpl struct {
	dependency validateIndirectCycleB
}

func (validateIndirectCycleAImpl) A() string {
	return "a"
}

type validateIndirectCycleBImpl struct {
	dependency validateIndirectCycleC
}

func (validateIndirectCycleBImpl) B() string {
	return "b"
}

type validateIndirectCycleCImpl struct {
	dependency validateIndirectCycleA
}

func (validateIndirectCycleCImpl) C() string {
	return "c"
}

type validateNamedDependency interface {
	Run() error
}

type validateNamedConsumer interface {
	Consume() error
}

type validateNamedConsumerImpl struct {
	dependency validateNamedDependency
}

func (s validateNamedConsumerImpl) Consume() error {
	return s.dependency.Run()
}

func TestValidateReturnsNilForEmptyRegistry(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateReturnsNilForValidDefaultGraph(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := Declare(func() validateRepository {
		return validateRepositoryImpl{}
	}); err != nil {
		t.Fatalf("Declare(repository) error = %v", err)
	}

	if err := Declare(func(repository validateRepository) validateUserService {
		return validateUserServiceImpl{
			repository: repository,
		}
	}); err != nil {
		t.Fatalf("Declare(service) error = %v", err)
	}

	if err := Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateReturnsNilForNamedServiceGraph(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := DeclareNamed("orders", func() validateNamedService {
		return validateNamedServiceImpl{
			name: "orders",
		}
	}); err != nil {
		t.Fatalf("DeclareNamed() error = %v", err)
	}

	if err := Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateReturnsNilForConcreteValueDependency(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := DeclareValue(validateConfig{
		Prefix: "nexus",
	}); err != nil {
		t.Fatalf("DeclareValue() error = %v", err)
	}

	if err := Declare(func(config validateConfig) validateConfigService {
		return validateConfigServiceImpl{
			config: config,
		}
	}); err != nil {
		t.Fatalf("Declare() error = %v", err)
	}

	if err := Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateReturnsMissingInterfaceDependencyError(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := Declare(func(dependency validateMissingDependency) validateMissingConsumer {
		return validateMissingConsumerImpl{
			dependency: dependency,
		}
	}); err != nil {
		t.Fatalf("Declare() error = %v", err)
	}

	err := Validate()

	if !errors.Is(err, ErrDependencyNotDeclared) {
		t.Fatalf("expected ErrDependencyNotDeclared, got %v", err)
	}

	if !errors.Is(err, ErrServiceNotDeclared) {
		t.Fatalf("expected ErrServiceNotDeclared, got %v", err)
	}
}

func TestValidateReturnsMissingConcreteValueError(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := Declare(func(config validateConfig) validateConfigService {
		return validateConfigServiceImpl{
			config: config,
		}
	}); err != nil {
		t.Fatalf("Declare() error = %v", err)
	}

	err := Validate()

	if !errors.Is(err, ErrDependencyNotDeclared) {
		t.Fatalf("expected ErrDependencyNotDeclared, got %v", err)
	}
}

func TestValidateDetectsDirectCircularDependency(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := Declare(func(dependency validateDirectCycleB) validateDirectCycleA {
		return validateDirectCycleAImpl{
			dependency: dependency,
		}
	}); err != nil {
		t.Fatalf("Declare(A) error = %v", err)
	}

	if err := Declare(func(dependency validateDirectCycleA) validateDirectCycleB {
		return validateDirectCycleBImpl{
			dependency: dependency,
		}
	}); err != nil {
		t.Fatalf("Declare(B) error = %v", err)
	}

	err := Validate()

	if !errors.Is(err, ErrCircularDependency) {
		t.Fatalf("expected ErrCircularDependency, got %v", err)
	}
}

func TestValidateDetectsIndirectCircularDependency(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := Declare(func(dependency validateIndirectCycleB) validateIndirectCycleA {
		return validateIndirectCycleAImpl{
			dependency: dependency,
		}
	}); err != nil {
		t.Fatalf("Declare(A) error = %v", err)
	}

	if err := Declare(func(dependency validateIndirectCycleC) validateIndirectCycleB {
		return validateIndirectCycleBImpl{
			dependency: dependency,
		}
	}); err != nil {
		t.Fatalf("Declare(B) error = %v", err)
	}

	if err := Declare(func(dependency validateIndirectCycleA) validateIndirectCycleC {
		return validateIndirectCycleCImpl{
			dependency: dependency,
		}
	}); err != nil {
		t.Fatalf("Declare(C) error = %v", err)
	}

	err := Validate()

	if !errors.Is(err, ErrCircularDependency) {
		t.Fatalf("expected ErrCircularDependency, got %v", err)
	}
}

func TestValidateDoesNotInvokeConstructors(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	var calls atomic.Int32

	if err := Declare(func() validateUserService {
		calls.Add(1)
		return validateUserServiceImpl{}
	}); err != nil {
		t.Fatalf("Declare() error = %v", err)
	}

	if err := Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if got := calls.Load(); got != 0 {
		t.Fatalf("constructor calls = %d, want 0", got)
	}
}

func TestValidateDoesNotCreateSingletonInstances(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := Declare(func() validateUserService {
		return validateUserServiceImpl{}
	}); err != nil {
		t.Fatalf("Declare() error = %v", err)
	}

	if err := Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	globalRegistry.mu.RLock()
	instanceCount := len(globalRegistry.instances)
	globalRegistry.mu.RUnlock()

	if instanceCount != 0 {
		t.Fatalf("singleton instance count = %d, want 0", instanceCount)
	}
}

func TestValidateNamedServiceRequiresDefaultInterfaceDependency(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := DeclareNamed("orders", func(dependency validateNamedDependency) validateNamedConsumer {
		return validateNamedConsumerImpl{
			dependency: dependency,
		}
	}); err != nil {
		t.Fatalf("DeclareNamed() error = %v", err)
	}

	err := Validate()

	if !errors.Is(err, ErrDependencyNotDeclared) {
		t.Fatalf("expected ErrDependencyNotDeclared, got %v", err)
	}

	if !errors.Is(err, ErrServiceNotDeclared) {
		t.Fatalf("expected ErrServiceNotDeclared, got %v", err)
	}
}

func TestValidateAcceptsResolvableGroupMembers(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := Declare(func() validateRepository {
		return validateRepositoryImpl{}
	}); err != nil {
		t.Fatalf("Declare(repository) error = %v", err)
	}

	if err := DeclareGroup("handlers", func(repository validateRepository) validateUserService {
		return validateUserServiceImpl{
			repository: repository,
		}
	}); err != nil {
		t.Fatalf("DeclareGroup() error = %v", err)
	}

	if err := Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateFailsForMissingGroupMemberServiceDependency(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := DeclareGroup("handlers", func(dependency validateMissingDependency) validateUserService {
		return validateUserServiceImpl{}
	}); err != nil {
		t.Fatalf("DeclareGroup() error = %v", err)
	}

	err := Validate()

	if !errors.Is(err, ErrDependencyNotDeclared) {
		t.Fatalf("expected ErrDependencyNotDeclared, got %v", err)
	}

	if !errors.Is(err, ErrServiceNotDeclared) {
		t.Fatalf("expected ErrServiceNotDeclared, got %v", err)
	}
}

func TestValidateFailsForMissingGroupMemberValueDependency(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := DeclareGroup("handlers", func(config validateConfig) validateConfigService {
		return validateConfigServiceImpl{
			config: config,
		}
	}); err != nil {
		t.Fatalf("DeclareGroup() error = %v", err)
	}

	err := Validate()

	if !errors.Is(err, ErrDependencyNotDeclared) {
		t.Fatalf("expected ErrDependencyNotDeclared, got %v", err)
	}
}

func TestValidateDetectsCycleReachableFromGroupMember(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := Declare(func(dependency validateDirectCycleB) validateDirectCycleA {
		return validateDirectCycleAImpl{
			dependency: dependency,
		}
	}); err != nil {
		t.Fatalf("Declare(A) error = %v", err)
	}

	if err := Declare(func(dependency validateDirectCycleA) validateDirectCycleB {
		return validateDirectCycleBImpl{
			dependency: dependency,
		}
	}); err != nil {
		t.Fatalf("Declare(B) error = %v", err)
	}

	if err := DeclareGroup("handlers", func(dependency validateDirectCycleA) validateUserService {
		return validateUserServiceImpl{}
	}); err != nil {
		t.Fatalf("DeclareGroup() error = %v", err)
	}

	err := Validate()

	if !errors.Is(err, ErrCircularDependency) {
		t.Fatalf("expected ErrCircularDependency, got %v", err)
	}
}

func TestValidateDoesNotInvokeGroupConstructors(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	var calls atomic.Int32

	if err := DeclareGroup("handlers", func() validateUserService {
		calls.Add(1)
		return validateUserServiceImpl{}
	}); err != nil {
		t.Fatalf("DeclareGroup() error = %v", err)
	}

	if err := Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if got := calls.Load(); got != 0 {
		t.Fatalf("group constructor calls = %d, want 0", got)
	}
}

func TestValidateAcceptsMultipleResolvableGroupMembers(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	if err := Declare(func() validateRepository {
		return validateRepositoryImpl{}
	}); err != nil {
		t.Fatalf("Declare(repository) error = %v", err)
	}

	if err := DeclareGroup("handlers", func(repository validateRepository) validateUserService {
		return validateUserServiceImpl{
			repository: repository,
		}
	}); err != nil {
		t.Fatalf("first DeclareGroup() error = %v", err)
	}

	if err := DeclareGroup("handlers", func(repository validateRepository) validateUserService {
		return validateUserServiceImpl{
			repository: repository,
		}
	}); err != nil {
		t.Fatalf("second DeclareGroup() error = %v", err)
	}

	if err := Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}
