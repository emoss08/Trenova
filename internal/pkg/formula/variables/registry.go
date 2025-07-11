package variables

import (
	"fmt"
	"sync"

	"github.com/emoss08/trenova/internal/pkg/formula/errors"
)

// * Registry manages formula variables
type Registry struct {
	mu        sync.RWMutex
	variables map[string]Variable
	
	// * Group variables by category for easier discovery
	byCategory map[string][]Variable
}

// * NewRegistry creates a new variable registry
func NewRegistry() *Registry {
	return &Registry{
		variables:  make(map[string]Variable),
		byCategory: make(map[string][]Variable),
	}
}

// * Register adds a variable to the registry
func (r *Registry) Register(variable Variable) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	name := variable.Name()
	if name == "" {
		return fmt.Errorf("variable name cannot be empty")
	}
	
	// * Check for duplicates
	if _, exists := r.variables[name]; exists {
		return fmt.Errorf("variable %s already registered", name)
	}
	
	// * Register the variable
	r.variables[name] = variable
	
	// * Add to category index
	category := variable.Category()
	r.byCategory[category] = append(r.byCategory[category], variable)
	
	return nil
}

// * Get retrieves a variable by name
func (r *Registry) Get(name string) (Variable, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	variable, exists := r.variables[name]
	if !exists {
		return nil, errors.NewVariableError(name, "registry", fmt.Errorf("variable not found"))
	}
	
	return variable, nil
}

// * GetByCategory returns all variables in a category
func (r *Registry) GetByCategory(category string) []Variable {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	vars := r.byCategory[category]
	// * Return a copy to prevent external modification
	result := make([]Variable, len(vars))
	copy(result, vars)
	return result
}

// * List returns all registered variables
func (r *Registry) List() []Variable {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	result := make([]Variable, 0, len(r.variables))
	for _, v := range r.variables {
		result = append(result, v)
	}
	return result
}

// * ListNames returns all registered variable names
func (r *Registry) ListNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	names := make([]string, 0, len(r.variables))
	for name := range r.variables {
		names = append(names, name)
	}
	return names
}

// * Categories returns all unique categories
func (r *Registry) Categories() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	categories := make([]string, 0, len(r.byCategory))
	for cat := range r.byCategory {
		categories = append(categories, cat)
	}
	return categories
}

// * Clear removes all variables from the registry
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.variables = make(map[string]Variable)
	r.byCategory = make(map[string][]Variable)
}

// * MustRegister registers a variable and panics on error
func (r *Registry) MustRegister(variable Variable) {
	if err := r.Register(variable); err != nil {
		panic(fmt.Sprintf("failed to register variable: %v", err))
	}
}

// * DefaultRegistry is the global variable registry
var DefaultRegistry = NewRegistry()

// * Register adds a variable to the default registry
func Register(variable Variable) error {
	return DefaultRegistry.Register(variable)
}

// * MustRegister adds a variable to the default registry and panics on error
func MustRegister(variable Variable) {
	DefaultRegistry.MustRegister(variable)
}

// * Get retrieves a variable from the default registry
func Get(name string) (Variable, error) {
	return DefaultRegistry.Get(name)
}

// * GetByCategory returns variables by category from the default registry
func GetByCategory(category string) []Variable {
	return DefaultRegistry.GetByCategory(category)
}

// * List returns all variables from the default registry
func List() []Variable {
	return DefaultRegistry.List()
}