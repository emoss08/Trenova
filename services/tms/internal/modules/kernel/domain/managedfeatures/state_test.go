package managedfeatures_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/stretchr/testify/assert"
)

func TestNormalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		received []platformcatalog.FeatureKey
		features []platformcatalog.FeatureKey
		unknown  []platformcatalog.FeatureKey
	}{
		{
			name:     "nothing received",
			received: nil,
			features: []platformcatalog.FeatureKey{},
			unknown:  []platformcatalog.FeatureKey{},
		},
		{
			name: "unknown keys are dropped and reported once, sorted",
			received: []platformcatalog.FeatureKey{
				"tms.teleport", "ai.desk", "tms.teleport", "ai.oracle",
			},
			features: []platformcatalog.FeatureKey{"ai.desk"},
			unknown:  []platformcatalog.FeatureKey{"ai.oracle", "tms.teleport"},
		},
		{
			name:     "base keys are dropped",
			received: []platformcatalog.FeatureKey{"tms.core", "tms.accounting", "tms.dispatch"},
			features: []platformcatalog.FeatureKey{"tms.accounting"},
			unknown:  []platformcatalog.FeatureKey{},
		},
		{
			name: "duplicates are dropped and the order is stable",
			received: []platformcatalog.FeatureKey{
				"tms.rating", "ai.desk", "tms.accounting", "ai.desk", "tms.rating",
			},
			features: []platformcatalog.FeatureKey{"ai.desk", "tms.accounting", "tms.rating"},
			unknown:  []platformcatalog.FeatureKey{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := managedfeatures.Normalize(tt.received)
			assert.Equal(t, tt.features, got.Features)
			assert.Equal(t, tt.unknown, got.Unknown)
		})
	}
}

func TestNormalizeGivesTheSameListForTheSameSet(t *testing.T) {
	t.Parallel()

	a := managedfeatures.Normalize([]platformcatalog.FeatureKey{"tms.rating", "ai.desk"})
	b := managedfeatures.Normalize([]platformcatalog.FeatureKey{"ai.desk", "tms.rating", "ai.desk"})
	assert.Equal(t, a.Features, b.Features)
}

func TestChanges(t *testing.T) {
	t.Parallel()

	added, removed := managedfeatures.Changes(
		[]platformcatalog.FeatureKey{"ai.desk", "tms.accounting"},
		[]platformcatalog.FeatureKey{"tms.accounting", "tms.rating"},
	)
	assert.Equal(t, []platformcatalog.FeatureKey{"tms.rating"}, added)
	assert.Equal(t, []platformcatalog.FeatureKey{"ai.desk"}, removed)
}

func TestHas(t *testing.T) {
	t.Parallel()

	state := &managedfeatures.State{Features: []platformcatalog.FeatureKey{"ai.desk"}}
	assert.True(t, state.Has(platformcatalog.FeatureAIDesk))
	assert.False(t, state.Has(platformcatalog.FeatureAccounting))

	var none *managedfeatures.State
	assert.False(t, none.Has(platformcatalog.FeatureAIDesk))
}
