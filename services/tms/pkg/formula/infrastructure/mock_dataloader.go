package infrastructure

import (
	"context"
	"fmt"
	"sync"

	"github.com/emoss08/trenova/pkg/formula/ports"
	"github.com/emoss08/trenova/pkg/formula/schema"
	"github.com/emoss08/trenova/pkg/formula/variables"
)

type MockDataLoader struct {
	mu               sync.RWMutex
	entities         map[string]map[string]any // schemaID -> entityID -> entity
	LastRequirements *ports.DataRequirements
	variableContexts map[string]variables.VariableContext
	loadEntityFunc   func(ctx context.Context, schemaID string, entityID string) (any, error)
	loadWithReqsFunc func(ctx context.Context, schemaID string, entityID string, requirements *ports.DataRequirements) (any, error)
	schemaRegistry   *schema.Registry
}

func NewMockDataLoader(schemaRegistry *schema.Registry) *MockDataLoader {
	return &MockDataLoader{
		entities:         make(map[string]map[string]any),
		variableContexts: make(map[string]variables.VariableContext),
		schemaRegistry:   schemaRegistry,
	}
}

func (m *MockDataLoader) AddEntity(schemaID, entityID string, entity any) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.entities[schemaID] == nil {
		m.entities[schemaID] = make(map[string]any)
	}
	m.entities[schemaID][entityID] = entity
}

func (m *MockDataLoader) SetVariableContext(entityID string, ctx variables.VariableContext) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.variableContexts[entityID] = ctx
}

func (m *MockDataLoader) LoadEntity(
	ctx context.Context,
	schemaID string,
	entityID string,
) (any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.loadEntityFunc != nil {
		return m.loadEntityFunc(ctx, schemaID, entityID)
	}

	if schemaEntities, ok := m.entities[schemaID]; ok {
		if entity, ok := schemaEntities[entityID]; ok {
			return entity, nil
		}
	}

	return nil, fmt.Errorf("entity not found: %s/%s", schemaID, entityID)
}

func (m *MockDataLoader) LoadEntityWithRequirements(
	ctx context.Context,
	schemaID string,
	entityID string,
	requirements *ports.DataRequirements,
) (any, error) {
	m.mu.Lock()
	m.LastRequirements = requirements
	m.mu.Unlock()

	if m.loadWithReqsFunc != nil {
		return m.loadWithReqsFunc(ctx, schemaID, entityID, requirements)
	}

	return m.LoadEntity(ctx, schemaID, entityID)
}

func (m *MockDataLoader) SetLoadEntityFunc(
	fn func(ctx context.Context, schemaID string, entityID string) (any, error),
) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loadEntityFunc = fn
}

func (m *MockDataLoader) SetLoadWithRequirementsFunc(
	fn func(ctx context.Context, schemaID string, entityID string, requirements *ports.DataRequirements) (any, error),
) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loadWithReqsFunc = fn
}

func (m *MockDataLoader) GetLastRequirements() *ports.DataRequirements {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.LastRequirements
}

func (m *MockDataLoader) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entities = make(map[string]map[string]any)
	m.variableContexts = make(map[string]variables.VariableContext)
	m.LastRequirements = nil
	m.loadEntityFunc = nil
	m.loadWithReqsFunc = nil
}

func (m *MockDataLoader) SimulateError(err error) {
	m.SetLoadEntityFunc(func(_ context.Context, schemaID string, entityID string) (any, error) {
		return nil, err
	})
	m.SetLoadWithRequirementsFunc(
		func(ctx context.Context, schemaID string, entityID string, requirements *ports.DataRequirements) (any, error) {
			return nil, err
		},
	)
}

func (m *MockDataLoader) GetEntity(schemaID string, entityID string) (any, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if schemaEntities, ok := m.entities[schemaID]; ok {
		if entity, ok := schemaEntities[entityID]; ok {
			return entity, true
		}
	}

	return nil, false
}

func (m *MockDataLoader) GetVariableContext(entityID string) (variables.VariableContext, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ctx, ok := m.variableContexts[entityID]
	return ctx, ok
}
