package seedtest

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

type MockSeedContext struct {
	*seedhelpers.SeedContext
	t *testing.T
}

func NewMockSeedContext(t *testing.T, db bun.IDB) *MockSeedContext {
	t.Helper()

	return &MockSeedContext{
		SeedContext: seedhelpers.NewSeedContext(db, seedhelpers.NewNoOpLogger(), nil),
		t:           t,
	}
}

func (m *MockSeedContext) RequireSet(key string, value any) {
	m.t.Helper()
	err := m.Set(key, value)
	require.NoError(m.t, err, "failed to set shared state key %s", key)
}

func (m *MockSeedContext) RequireGet(key string) any {
	m.t.Helper()
	val, exists := m.Get(key)
	require.True(m.t, exists, "key %s should exist in shared state", key)
	return val
}

func (m *MockSeedContext) AssertKeyExists(key string) {
	m.t.Helper()
	_, exists := m.Get(key)
	require.True(m.t, exists, "key %s should exist in shared state", key)
}

func (m *MockSeedContext) AssertKeyNotExists(key string) {
	m.t.Helper()
	_, exists := m.Get(key)
	require.False(m.t, exists, "key %s should not exist in shared state", key)
}

func (m *MockSeedContext) AssertTrackedEntityCount(seedName string, expected int) {
	m.t.Helper()
	ctx := context.Background()
	entities, err := m.SeedContext.GetCreatedEntities(ctx, seedName)
	require.NoError(m.t, err)
	require.Len(m.t, entities, expected, "tracked entity count for seed %s should match", seedName)
}

func (m *MockSeedContext) GetTrackerCount() int {
	m.t.Helper()
	ctx := context.Background()
	entities, err := m.GetAllCreatedEntities(ctx)
	require.NoError(m.t, err)
	return len(entities)
}
