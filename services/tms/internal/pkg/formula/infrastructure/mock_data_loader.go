/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package infrastructure

import (
	"context"
	"fmt"
	"sync"

	"github.com/emoss08/trenova/internal/pkg/formula/ports"
	"github.com/emoss08/trenova/internal/pkg/formula/schema"
	"github.com/emoss08/trenova/internal/pkg/formula/variables"
)

// MockDataLoader implements DataLoader for testing purposes
type MockDataLoader struct {
	mu               sync.RWMutex
	entities         map[string]map[string]any // schemaID -> entityID -> entity
	LastRequirements *ports.DataRequirements
	variableContexts map[string]variables.VariableContext
	loadEntityFunc   func(ctx context.Context, schemaID string, entityID string) (any, error)
	loadWithReqsFunc func(ctx context.Context, schemaID string, entityID string, requirements *ports.DataRequirements) (any, error)
	schemaRegistry   *schema.SchemaRegistry
}

// NewMockDataLoader creates a new mock data loader
func NewMockDataLoader(schemaRegistry *schema.SchemaRegistry) *MockDataLoader {
	return &MockDataLoader{
		entities:         make(map[string]map[string]any),
		variableContexts: make(map[string]variables.VariableContext),
		schemaRegistry:   schemaRegistry,
	}
}

// AddEntity adds a test entity to the mock loader
func (m *MockDataLoader) AddEntity(schemaID string, entityID string, entity any) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.entities[schemaID] == nil {
		m.entities[schemaID] = make(map[string]any)
	}
	m.entities[schemaID][entityID] = entity
}

// SetVariableContext sets a variable context for a specific entity
func (m *MockDataLoader) SetVariableContext(entityID string, ctx variables.VariableContext) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.variableContexts[entityID] = ctx
}

// LoadEntity implements the DataLoader interface
func (m *MockDataLoader) LoadEntity(
	ctx context.Context,
	schemaID string,
	entityID string,
) (any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// If custom function is set, use it
	if m.loadEntityFunc != nil {
		return m.loadEntityFunc(ctx, schemaID, entityID)
	}

	// Otherwise use stored entities
	if schemaEntities, ok := m.entities[schemaID]; ok {
		if entity, ok := schemaEntities[entityID]; ok {
			return entity, nil
		}
	}

	return nil, fmt.Errorf("entity not found: %s/%s", schemaID, entityID)
}

// LoadEntityWithRequirements implements the DataLoader interface
func (m *MockDataLoader) LoadEntityWithRequirements(
	ctx context.Context,
	schemaID string,
	entityID string,
	requirements *ports.DataRequirements,
) (any, error) {
	m.mu.Lock()
	m.LastRequirements = requirements
	m.mu.Unlock()

	// If custom function is set, use it
	if m.loadWithReqsFunc != nil {
		return m.loadWithReqsFunc(ctx, schemaID, entityID, requirements)
	}

	// Otherwise delegate to LoadEntity
	return m.LoadEntity(ctx, schemaID, entityID)
}

// SetLoadEntityFunc sets a custom function for LoadEntity
func (m *MockDataLoader) SetLoadEntityFunc(
	fn func(ctx context.Context, schemaID string, entityID string) (any, error),
) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loadEntityFunc = fn
}

// SetLoadWithRequirementsFunc sets a custom function for LoadEntityWithRequirements
func (m *MockDataLoader) SetLoadWithRequirementsFunc(
	fn func(ctx context.Context, schemaID string, entityID string, requirements *ports.DataRequirements) (any, error),
) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loadWithReqsFunc = fn
}

// GetLastRequirements returns the last requirements passed to LoadEntityWithRequirements
func (m *MockDataLoader) GetLastRequirements() *ports.DataRequirements {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.LastRequirements
}

// Clear removes all stored entities and resets state
func (m *MockDataLoader) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entities = make(map[string]map[string]any)
	m.variableContexts = make(map[string]variables.VariableContext)
	m.LastRequirements = nil
	m.loadEntityFunc = nil
	m.loadWithReqsFunc = nil
}

// SimulateError configures the mock to return an error
func (m *MockDataLoader) SimulateError(err error) {
	m.SetLoadEntityFunc(func(ctx context.Context, schemaID string, entityID string) (any, error) {
		return nil, err
	})
	m.SetLoadWithRequirementsFunc(
		func(ctx context.Context, schemaID string, entityID string, requirements *ports.DataRequirements) (any, error) {
			return nil, err
		},
	)
}

// GetEntity retrieves a stored entity (for test verification)
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

// GetVariableContext retrieves a stored variable context
func (m *MockDataLoader) GetVariableContext(entityID string) (variables.VariableContext, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ctx, ok := m.variableContexts[entityID]
	return ctx, ok
}
