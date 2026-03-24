package seeder

import (
	"fmt"
	"slices"
	"sync"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
)

type Registry struct {
	mu       sync.RWMutex
	seeds    map[string]Seed
	envSeeds map[common.Environment][]string
}

func NewRegistry() *Registry {
	return &Registry{
		seeds:    make(map[string]Seed),
		envSeeds: make(map[common.Environment][]string),
	}
}

func (r *Registry) Register(seed Seed) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := seed.Name()
	if _, exists := r.seeds[name]; exists {
		return fmt.Errorf("%w: %s", ErrSeedAlreadyRegistered, name)
	}

	r.seeds[name] = seed

	for _, env := range seed.Environments() {
		r.envSeeds[env] = append(r.envSeeds[env], name)
	}

	return nil
}

func (r *Registry) MustRegister(seed Seed) {
	if err := r.Register(seed); err != nil {
		panic(err)
	}
}

func (r *Registry) RegisterForEnv(env common.Environment, seeds ...Seed) error {
	for _, seed := range seeds {
		if !slices.Contains(seed.Environments(), env) {
			return fmt.Errorf("seed %q does not support environment %s", seed.Name(), env)
		}
		if err := r.Register(seed); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) Get(name string) (Seed, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	seed, exists := r.seeds[name]
	return seed, exists
}

func (r *Registry) GetForEnvironment(env common.Environment) []Seed {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Seed
	for _, seed := range r.seeds {
		if slices.Contains(seed.Environments(), env) {
			result = append(result, seed)
		}
	}
	return result
}

func (r *Registry) Validate() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for name, seed := range r.seeds {
		for _, dep := range seed.Dependencies() {
			if _, exists := r.seeds[dep]; !exists {
				return NewMissingDependencyError(name, []string{dep})
			}
		}
	}

	graph := NewGraph()
	for _, seed := range r.seeds {
		graph.AddNode(seed)
	}
	for _, seed := range r.seeds {
		for _, dep := range seed.Dependencies() {
			graph.AddEdge(seed.Name(), dep)
		}
	}

	if cycle := graph.DetectCycle(); len(cycle) > 0 {
		return NewCircularDependencyError(cycle)
	}

	return nil
}

func (r *Registry) GetExecutionOrder(env common.Environment, target string) ([]Seed, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	envSeeds := make(map[string]Seed)
	for _, seed := range r.seeds {
		if slices.Contains(seed.Environments(), env) {
			envSeeds[seed.Name()] = seed
		}
	}

	if len(envSeeds) == 0 {
		return nil, nil
	}

	graph := NewGraph()
	for _, seed := range envSeeds {
		graph.AddNode(seed)
	}
	for _, seed := range envSeeds {
		for _, dep := range seed.Dependencies() {
			if _, exists := envSeeds[dep]; exists {
				graph.AddEdge(seed.Name(), dep)
			}
		}
	}

	if err := graph.Validate(); err != nil {
		return nil, err
	}

	order, err := graph.TopologicalSort()
	if err != nil {
		return nil, err
	}

	if target != "" {
		if _, exists := envSeeds[target]; !exists {
			return nil, fmt.Errorf("%w: %s", ErrSeedNotFound, target)
		}

		deps, err := graph.GetDependenciesFor(target)
		if err != nil {
			return nil, err
		}

		needed := make(map[string]bool)
		needed[target] = true
		for _, dep := range deps {
			needed[dep] = true
		}

		var filtered []string
		for _, name := range order {
			if needed[name] {
				filtered = append(filtered, name)
			}
		}
		order = filtered
	}

	result := make([]Seed, 0, len(order))
	for _, name := range order {
		result = append(result, envSeeds[name])
	}

	return result, nil
}

func (r *Registry) All() []Seed {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Seed, 0, len(r.seeds))
	for _, seed := range r.seeds {
		result = append(result, seed)
	}
	return result
}

func (r *Registry) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.seeds)
}
