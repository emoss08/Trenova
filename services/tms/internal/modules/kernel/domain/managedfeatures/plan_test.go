package managedfeatures_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addOnNames(addOns []platformcatalog.AddOn) []string {
	out := make([]string, 0, len(addOns))
	for _, addOn := range addOns {
		out = append(out, addOn.Name)
	}
	return out
}

func TestReplaceRecordsTheFirstListAsTheStartingPlan(t *testing.T) {
	t.Parallel()

	r := managedfeatures.Replace(nil, []platformcatalog.FeatureKey{"ai.desk", "tms.core"}, 100)

	require.True(t, r.Changed())
	assert.Nil(t, r.Previous)
	assert.Equal(t, []platformcatalog.FeatureKey{"ai.desk"}, r.Current.Features)
	assert.True(t, r.Change.Starting)
	assert.Equal(t, []platformcatalog.FeatureKey{"ai.desk"}, r.Change.Added)
	assert.Equal(t, []platformcatalog.FeatureKey{}, r.Change.Removed)
	assert.Equal(t, int64(100), r.Change.ChangedAt)
}

func TestReplaceRecordsNothingForAnUnchangedList(t *testing.T) {
	t.Parallel()

	held := &managedfeatures.State{
		ID:       managedfeatures.CurrentStateID,
		Features: []platformcatalog.FeatureKey{"ai.desk", "tms.accounting"},
	}
	r := managedfeatures.Replace(held, []platformcatalog.FeatureKey{
		"tms.accounting", "ai.desk", "tms.core", "tms.teleport",
	}, 200)

	assert.False(t, r.Changed())
	assert.Same(t, held, r.Current)
}

func TestReplaceRecordsWhatWasAddedAndRemoved(t *testing.T) {
	t.Parallel()

	held := &managedfeatures.State{Features: []platformcatalog.FeatureKey{"tms.accounting"}}
	r := managedfeatures.Replace(held, []platformcatalog.FeatureKey{"ai.desk"}, 300)

	require.True(t, r.Changed())
	assert.False(t, r.Change.Starting)
	assert.Equal(t, []platformcatalog.FeatureKey{"ai.desk"}, r.Change.Added)
	assert.Equal(t, []platformcatalog.FeatureKey{"tms.accounting"}, r.Change.Removed)
}

func TestShiftNamesAnAddOnThatLeftOnlyWhenEveryFeatureIsOff(t *testing.T) {
	t.Parallel()

	partly := managedfeatures.Shift(
		[]platformcatalog.FeatureKey{"tms.accounting", "tms.factoring"},
		[]platformcatalog.FeatureKey{"tms.accounting"},
	)
	assert.True(t, partly.Empty(), "an add-on that keeps a feature has not left")

	gone := managedfeatures.Shift(
		[]platformcatalog.FeatureKey{"tms.accounting", "tms.factoring", "ai.desk"},
		[]platformcatalog.FeatureKey{"ai.desk"},
	)
	assert.Equal(t, []string{"Accounting & Cash"}, addOnNames(gone.Left))
	assert.Empty(t, gone.Joined)
}

func TestShiftNamesAnAddOnThatJoined(t *testing.T) {
	t.Parallel()

	shift := managedfeatures.Shift(nil, []platformcatalog.FeatureKey{
		"tms.dispatch_intelligence", "ai.watchtower",
	})
	assert.Equal(t, []string{"Dispatch Intelligence", "Trenova AI"}, addOnNames(shift.Joined))
	assert.Empty(t, shift.Left)
}

func TestShiftIgnoresAFeatureAddedToAnAddOnThatStaysOn(t *testing.T) {
	t.Parallel()

	shift := managedfeatures.Shift(
		[]platformcatalog.FeatureKey{"ai.desk"},
		[]platformcatalog.FeatureKey{"ai.desk", "ai.watchtower"},
	)
	assert.True(t, shift.Empty())
}

func TestViewListsAddOnsOnAndOffThePlanInCatalogueOrder(t *testing.T) {
	t.Parallel()

	view := managedfeatures.View([]platformcatalog.FeatureKey{"ai.desk", "tms.accounting"})

	included := make([]string, 0, len(view.Included))
	for _, entry := range view.Included {
		included = append(included, entry.AddOn.Name)
	}
	assert.Equal(t, []string{"Accounting & Cash", "Trenova AI"}, included)
	assert.Len(t, view.Available, len(platformcatalog.AddOns())-2)
	assert.Equal(t, "Rating & Revenue Recovery", view.Available[0].AddOn.Name)
	assert.Equal(t, "Freight", view.Base[0].Name)
	assert.Equal(t, "Accounting", view.Included[0].Features[0].Name)
	assert.Equal(t, "Factoring", view.Included[0].Features[1].Name)
}

func TestHistoryWalksBackFromTheHeldList(t *testing.T) {
	t.Parallel()

	changes := []*managedfeatures.Change{
		{
			Added:     []platformcatalog.FeatureKey{"ai.watchtower"},
			Removed:   []platformcatalog.FeatureKey{},
			ChangedAt: 400,
		},
		{
			Added:     []platformcatalog.FeatureKey{"tms.accounting"},
			Removed:   []platformcatalog.FeatureKey{"tms.dispatch_intelligence"},
			ChangedAt: 300,
		},
		{
			Added:     []platformcatalog.FeatureKey{"ai.desk", "tms.dispatch_intelligence"},
			Removed:   []platformcatalog.FeatureKey{},
			Starting:  true,
			ChangedAt: 200,
		},
	}

	history := managedfeatures.History(
		[]platformcatalog.FeatureKey{"ai.desk", "ai.watchtower", "tms.accounting"},
		changes,
	)

	require.Len(t, history, 2, "a change inside an add-on that stays on is not listed")
	assert.Equal(t, int64(300), history[0].ChangedAt)
	assert.Equal(t, []string{"Dispatch Intelligence"}, addOnNames(history[0].Left))
	assert.Equal(t, []string{"Accounting & Cash"}, addOnNames(history[0].Joined))
	assert.True(t, history[1].Starting)
	assert.Equal(t, []string{"Dispatch Intelligence", "Trenova AI"}, addOnNames(history[1].Joined))
	assert.Empty(t, history[1].Left)
}
