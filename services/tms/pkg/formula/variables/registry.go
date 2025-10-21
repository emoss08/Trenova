package variables

import (
	"fmt"
	"sync"

	"github.com/emoss08/trenova/pkg/formula/errors"
)

type Registry struct {
	mu         sync.RWMutex
	variables  map[string]Variable
	byCategory map[string][]Variable
}

func NewRegistry() *Registry {
	return &Registry{
		variables:  make(map[string]Variable),
		byCategory: make(map[string][]Variable),
	}
}

func (r *Registry) Register(variable Variable) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := variable.Name()
	if name == "" {
		return ErrVariableNameEmpty
	}

	if _, exists := r.variables[name]; exists {
		return fmt.Errorf("variable %s already registered", name)
	}

	r.variables[name] = variable

	category := variable.Category()
	r.byCategory[category] = append(r.byCategory[category], variable)

	return nil
}

func (r *Registry) Get(name string) (Variable, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	variable, exists := r.variables[name]
	if !exists {
		return nil, errors.NewVariableError(name, "registry", ErrVariableNotFound)
	}

	return variable, nil
}

func (r *Registry) GetByCategory(category string) []Variable {
	r.mu.RLock()
	defer r.mu.RUnlock()

	vars := r.byCategory[category]
	result := make([]Variable, len(vars))
	copy(result, vars)
	return result
}

func (r *Registry) List() []Variable {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Variable, 0, len(r.variables))
	for _, v := range r.variables {
		result = append(result, v)
	}
	return result
}

func (r *Registry) ListNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.variables))
	for name := range r.variables {
		names = append(names, name)
	}
	return names
}

func (r *Registry) Categories() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	categories := make([]string, 0, len(r.byCategory))
	for cat := range r.byCategory {
		categories = append(categories, cat)
	}
	return categories
}

func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.variables = make(map[string]Variable)
	r.byCategory = make(map[string][]Variable)
}

func (r *Registry) MustRegister(variable Variable) {
	if err := r.Register(variable); err != nil {
		panic(fmt.Sprintf("failed to register variable: %v", err))
	}
}

var DefaultRegistry = NewRegistry()

func Register(variable Variable) error {
	return DefaultRegistry.Register(variable)
}

func MustRegister(variable Variable) {
	DefaultRegistry.MustRegister(variable)
}

func Get(name string) (Variable, error) {
	return DefaultRegistry.Get(name)
}

func GetByCategory(category string) []Variable {
	return DefaultRegistry.GetByCategory(category)
}

func List() []Variable {
	return DefaultRegistry.List()
}
