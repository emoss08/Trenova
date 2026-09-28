package agenttoolschema_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnum_ListsTheSourceAndNamesIt(t *testing.T) {
	t.Parallel()

	property := agenttoolschema.Enum("What kind of stop.", agenttoolschema.StopTypes)

	assert.Equal(t, "string", property[toolschema.KeyType])
	assert.Equal(t, "What kind of stop.", property[toolschema.KeyDescription])
	assert.Equal(t,
		[]string{"Pickup", "Delivery", "SplitPickup", "SplitDelivery"},
		property[toolschema.KeyEnum])
	assert.Equal(t, "shipment.stopType", property[toolschema.KeyEnumOf])

	registered, ok := agenttoolschema.Registered("shipment.stopType")
	require.True(t, ok)
	assert.Equal(t, property[toolschema.KeyEnum], registered)
}

func TestSource_HoldsTheDomainsValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, shipment.CommentPriorityValues(), agenttoolschema.CommentPriorities.Values)
	assert.Contains(t, agenttoolschema.CommentPriorities.Names(), "Urgent")
}

func TestSource_RefusesASecondListUnderOneName(t *testing.T) {
	t.Parallel()

	agenttoolschema.Source("test.fixed", []string{"a", "b"})
	assert.NotPanics(t, func() { agenttoolschema.Source("test.fixed", []string{"a", "b"}) })
	assert.Panics(t, func() { agenttoolschema.Source("test.fixed", []string{"b"}) })
	assert.Panics(t, func() { agenttoolschema.Derived("test.fixed", []string{"a", "b"}) })
	assert.Panics(t, func() { agenttoolschema.Source("", []string{"a"}) })
}

// A list a tool reads from a catalog changes with the catalog, and a test
// that builds the tool over a stub must not be refused by the real one.
func TestDerived_TakesTheLatestList(t *testing.T) {
	t.Parallel()

	agenttoolschema.Derived("test.derived", []string{"a"})
	agenttoolschema.Derived("test.derived", []string{"a", "b"})

	registered, ok := agenttoolschema.Registered("test.derived")
	require.True(t, ok)
	assert.Equal(t, []string{"a", "b"}, registered)

	_, ok = agenttoolschema.Registered("test.never")
	assert.False(t, ok)
}
