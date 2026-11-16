package managedfeatures_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	planURL = "https://console.trenova.cloud/instances/0193a1b2-0000-7000-8000-000000000001/plan"
	dec14   = int64(1_797_206_400)
	jan14   = int64(1_799_884_800)
)

func ending(addOn platformcatalog.AddOnKey, at int64) managedfeatures.AddOnEnding {
	return managedfeatures.AddOnEnding{AddOn: addOn, EndsAt: at}
}

func endings(list ...managedfeatures.AddOnEnding) *[]managedfeatures.AddOnEnding {
	return &list
}

func text(s string) *string {
	return &s
}

func TestAPlanURLIsAcceptedOnlyAsAnAbsoluteWebAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
		ok   bool
	}{
		{name: "https with a path", url: planURL, ok: true},
		{name: "http with a port", url: "http://localhost:5173/instances/1/plan", ok: true},
		{name: "empty", url: ""},
		{name: "relative", url: "/instances/1/plan"},
		{name: "no host", url: "https:///instances/1/plan"},
		{name: "a port and no host", url: "https://:443/plan"},
		{name: "another scheme", url: "javascript:alert(1)"},
		{name: "a file", url: "file:///etc/passwd"},
		{name: "user information", url: "https://user:secret@console.trenova.cloud/plan"},
		{name: "a control character", url: "https://console.trenova.cloud/\nplan"},
		{name: "too long", url: "https://console.trenova.cloud/" + strings.Repeat("a", 2048)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := (&managedfeatures.Received{PlanURL: text(tt.url)}).Validate(today)
			if tt.ok {
				assert.NoError(t, err)
				return
			}
			_, refused := errortypes.RefusalOf(err)
			assert.True(t, refused, "%q must be refused", tt.url)
		})
	}
}

func TestAddOnEndingsAreRefusedWhenMalformed(t *testing.T) {
	t.Parallel()

	tooMany := make([]managedfeatures.AddOnEnding, managedfeatures.MaxAddOnEndings+1)
	for i := range tooMany {
		tooMany[i] = ending(platformcatalog.AddOnAccounting, dec14)
	}
	tests := []struct {
		name    string
		endings []managedfeatures.AddOnEnding
		ok      bool
	}{
		{name: "none", endings: []managedfeatures.AddOnEnding{}, ok: true},
		{name: "a known add-on", endings: []managedfeatures.AddOnEnding{ending("ACC", dec14)}, ok: true},
		{name: "a well-shaped unknown add-on", endings: []managedfeatures.AddOnEnding{ending("ZZZ9", dec14)}, ok: true},
		{name: "lower case", endings: []managedfeatures.AddOnEnding{ending("acc", dec14)}},
		{name: "one letter", endings: []managedfeatures.AddOnEnding{ending("A", dec14)}},
		{name: "too long", endings: []managedfeatures.AddOnEnding{ending("ABCDEFGHIJK", dec14)}},
		{name: "no end", endings: []managedfeatures.AddOnEnding{ending("ACC", 0)}},
		{name: "more than fifty", endings: tooMany},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := (&managedfeatures.Received{AddOnEndings: &tt.endings}).Validate(today)
			if tt.ok {
				assert.NoError(t, err)
				return
			}
			_, refused := errortypes.RefusalOf(err)
			assert.True(t, refused)
		})
	}
}

func TestNormalizeEndingsDropsUnknownAddOnsAndKeepsTheEarliestEnd(t *testing.T) {
	t.Parallel()

	got := managedfeatures.NormalizeEndings([]managedfeatures.AddOnEnding{
		ending("SEC", jan14),
		ending("ZZZ", dec14),
		ending("ACC", jan14),
		ending("ACC", dec14),
		ending("BASE", dec14),
	})

	assert.Equal(t, []managedfeatures.AddOnEnding{ending("ACC", dec14), ending("SEC", jan14)}, got.Endings)
	assert.Equal(t, []platformcatalog.AddOnKey{"BASE", "ZZZ"}, got.Unknown)
}

func TestReplaceKeepsThePlanURLAndEndingsWhenARequestLeavesThemOut(t *testing.T) {
	t.Parallel()

	held := &managedfeatures.State{
		ID:           managedfeatures.CurrentStateID,
		Features:     []platformcatalog.FeatureKey{"tms.accounting"},
		AddOnEndings: []managedfeatures.AddOnEnding{ending("ACC", dec14)},
		PlanURL:      planURL,
	}

	r := managedfeatures.Replace(held, features("tms.accounting"), 500)

	assert.False(t, r.Updated())
	assert.Same(t, held, r.Current)
}

func TestReplaceStoresANewPlanURLWithoutRecordingAPlanChange(t *testing.T) {
	t.Parallel()

	held := &managedfeatures.State{Features: []platformcatalog.FeatureKey{"tms.accounting"}}

	r := managedfeatures.Replace(held, &managedfeatures.Received{
		Features: []platformcatalog.FeatureKey{"tms.accounting"},
		PlanURL:  text(planURL),
	}, 500)

	require.True(t, r.Updated())
	assert.False(t, r.Changed(), "a new address is not a change to the plan")
	assert.False(t, r.FeaturesChanged())
	assert.Equal(t, planURL, r.Current.PlanURL)
}

func TestReplaceRecordsAnAddOnSetToEnd(t *testing.T) {
	t.Parallel()

	held := &managedfeatures.State{
		Features: []platformcatalog.FeatureKey{"ai.desk", "tms.accounting"},
	}

	r := managedfeatures.Replace(held, &managedfeatures.Received{
		Features: []platformcatalog.FeatureKey{"ai.desk", "tms.accounting"},
		AddOnEndings: endings(
			ending("ACC", dec14),
			ending("RTE", dec14),
			ending("ZZZ", dec14),
		),
	}, 600)

	require.True(t, r.Changed())
	assert.False(t, r.FeaturesChanged())
	assert.Equal(
		t,
		[]managedfeatures.AddOnEnding{ending("ACC", dec14)},
		r.Change.Ending,
		"only an add-on on the plan is recorded as ending",
	)
	assert.Empty(t, r.Change.Kept)
	assert.Empty(t, r.Change.Added)
	assert.Equal(
		t,
		[]managedfeatures.AddOnEnding{ending("ACC", dec14), ending("RTE", dec14)},
		r.Current.AddOnEndings,
	)
	assert.Equal(t, []platformcatalog.AddOnKey{"ZZZ"}, r.UnknownAddOns)
}

func TestReplaceRecordsAMovedEndAsANewEnding(t *testing.T) {
	t.Parallel()

	held := &managedfeatures.State{
		Features:     []platformcatalog.FeatureKey{"tms.accounting"},
		AddOnEndings: []managedfeatures.AddOnEnding{ending("ACC", dec14)},
	}

	r := managedfeatures.Replace(held, &managedfeatures.Received{
		Features:     []platformcatalog.FeatureKey{"tms.accounting"},
		AddOnEndings: endings(ending("ACC", jan14)),
	}, 700)

	require.True(t, r.Changed())
	assert.Equal(t, []managedfeatures.AddOnEnding{ending("ACC", jan14)}, r.Change.Ending)
	assert.Empty(t, r.Change.Kept)
}

func TestReplaceRecordsAnAddOnThatStaysWhenItsEndIsCalledOff(t *testing.T) {
	t.Parallel()

	held := &managedfeatures.State{
		Features:     []platformcatalog.FeatureKey{"tms.accounting"},
		AddOnEndings: []managedfeatures.AddOnEnding{ending("ACC", dec14)},
	}

	r := managedfeatures.Replace(held, &managedfeatures.Received{
		Features:     []platformcatalog.FeatureKey{"tms.accounting"},
		AddOnEndings: endings(),
	}, 800)

	require.True(t, r.Changed())
	assert.Equal(t, []platformcatalog.AddOnKey{"ACC"}, r.Change.Kept)
	assert.Empty(t, r.Change.Ending)
	assert.Empty(t, r.Current.AddOnEndings)
}

func TestReplaceRecordsAnAddOnThatLeftOnItsDayAsLeftNotKept(t *testing.T) {
	t.Parallel()

	held := &managedfeatures.State{
		Features:     []platformcatalog.FeatureKey{"ai.desk", "tms.accounting"},
		AddOnEndings: []managedfeatures.AddOnEnding{ending("ACC", dec14)},
	}

	r := managedfeatures.Replace(held, &managedfeatures.Received{
		Features:     []platformcatalog.FeatureKey{"ai.desk"},
		AddOnEndings: endings(),
	}, 900)

	require.True(t, r.Changed())
	assert.True(t, r.FeaturesChanged())
	assert.Equal(t, []platformcatalog.FeatureKey{"tms.accounting"}, r.Change.Removed)
	assert.Empty(t, r.Change.Kept)
}

func TestHistoryListsAnAddOnSetToEndAndOneThatStays(t *testing.T) {
	t.Parallel()

	changes := []*managedfeatures.Change{
		{
			Added:     []platformcatalog.FeatureKey{},
			Removed:   []platformcatalog.FeatureKey{},
			Kept:      []platformcatalog.AddOnKey{"ACC"},
			ChangedAt: 300,
		},
		{
			Added:     []platformcatalog.FeatureKey{},
			Removed:   []platformcatalog.FeatureKey{},
			Ending:    []managedfeatures.AddOnEnding{ending("ACC", dec14)},
			ChangedAt: 200,
		},
		{
			Added:     []platformcatalog.FeatureKey{"tms.accounting"},
			Removed:   []platformcatalog.FeatureKey{},
			Starting:  true,
			ChangedAt: 100,
		},
	}

	history := managedfeatures.History([]platformcatalog.FeatureKey{"tms.accounting"}, changes)

	require.Len(t, history, 3)
	assert.Equal(t, []string{"Accounting & Cash"}, addOnNames(history[0].Kept))
	assert.Empty(t, history[0].Ending)
	require.Len(t, history[1].Ending, 1)
	assert.Equal(t, "Accounting & Cash", history[1].Ending[0].AddOn.Name)
	assert.Equal(t, dec14, history[1].Ending[0].EndsAt)
	assert.True(t, history[2].Starting)
}

func TestViewMarksAnIncludedAddOnThatIsEnding(t *testing.T) {
	t.Parallel()

	view := managedfeatures.View(
		[]platformcatalog.FeatureKey{"ai.desk", "tms.accounting"},
		[]managedfeatures.AddOnEnding{ending("ACC", dec14), ending("RTE", dec14)},
	)

	require.Len(t, view.Included, 2)
	assert.Equal(t, dec14, view.Included[0].EndsAt)
	assert.Zero(t, view.Included[1].EndsAt)
	for _, entry := range view.Available {
		assert.Zero(t, entry.EndsAt, "%s is not on the plan, so it does not end", entry.AddOn.Name)
	}
}
