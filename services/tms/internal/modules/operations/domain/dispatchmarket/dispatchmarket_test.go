package dispatchmarket_test

import (
	"math"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	mondayMidnight = int64(1_793_577_600)
	hour           = int64(3_600)
)

var (
	chicago = dispatchmarket.Point{Latitude: 41.8781, Longitude: -87.6298}
	joliet  = dispatchmarket.Point{Latitude: 41.525, Longitude: -88.0817}
	dallas  = dispatchmarket.Point{Latitude: 32.7767, Longitude: -96.7970}
)

func TestSlotOf_CountsFourHourBlocksFromMondayMidnightUTC(t *testing.T) {
	t.Parallel()

	assert.Equal(t, int16(0), dispatchmarket.SlotOf(mondayMidnight))
	assert.Equal(t, int16(0), dispatchmarket.SlotOf(mondayMidnight+4*hour-1))
	assert.Equal(t, int16(1), dispatchmarket.SlotOf(mondayMidnight+4*hour))
	assert.Equal(t, int16(41), dispatchmarket.SlotOf(mondayMidnight+7*24*hour-1))
	assert.Equal(t, int16(0), dispatchmarket.SlotOf(mondayMidnight+7*24*hour))
	assert.Equal(
		t,
		dispatchmarket.WeekOf(mondayMidnight)+1,
		dispatchmarket.WeekOf(mondayMidnight+7*24*hour),
	)
}

func TestCellOf_GroupsNearbyPointsAndRefusesUnknownOnes(t *testing.T) {
	t.Parallel()

	chicagoCell, ok := dispatchmarket.CellOf(chicago)
	require.True(t, ok)
	jolietCell, ok := dispatchmarket.CellOf(joliet)
	require.True(t, ok)
	dallasCell, ok := dispatchmarket.CellOf(dallas)
	require.True(t, ok)

	assert.Equal(t, chicagoCell, jolietCell, "Joliet is in Chicago's market")
	assert.NotEqual(t, chicagoCell, dallasCell)

	_, ok = dispatchmarket.CellOf(dispatchmarket.Point{})
	assert.False(t, ok, "a zero position is an unknown one")
	_, ok = dispatchmarket.CellOf(dispatchmarket.Point{Latitude: 91, Longitude: 0})
	assert.False(t, ok)
	_, ok = dispatchmarket.CellOf(dispatchmarket.Point{Latitude: math.NaN(), Longitude: 1})
	assert.False(t, ok)
}

func sample(cents float64, drivers int32, observedAt int64) *dispatchmarket.Sample {
	return &dispatchmarket.Sample{SumCents: cents, Drivers: drivers, ObservedAt: observedAt}
}

func TestFold_AveragesAWeekThenFoldsWeeksAsAMovingAverage(t *testing.T) {
	t.Parallel()

	value := &dispatchmarket.MarketValue{}
	require.True(t, value.Fold(sample(30_000, 2, mondayMidnight)))
	require.True(t, value.Fold(sample(10_000, 2, mondayMidnight+hour)))

	assert.InDelta(t, 10_000.0, value.Estimate(), 1e-9, "the first week is its own mean")
	assert.Equal(t, int32(0), value.Weeks)
	assert.Equal(t, int64(2), value.Observations)

	nextWeek := mondayMidnight + 7*24*hour
	require.True(t, value.Fold(sample(20_000, 1, nextWeek)))

	assert.Equal(t, int32(1), value.Weeks)
	assert.InDelta(t, 10_000.0, value.ValueCents, 1e-9)
	assert.InDelta(t, 15_000.0, value.Estimate(), 1e-9, "the second week weighs a half")
}

func TestFold_IgnoresAReplayAndAnOlderWeek(t *testing.T) {
	t.Parallel()

	value := &dispatchmarket.MarketValue{}
	nextWeek := mondayMidnight + 7*24*hour
	require.True(t, value.Fold(sample(10_000, 1, nextWeek)))

	assert.False(t, value.Fold(sample(10_000, 1, nextWeek)), "a retried plan folds once")
	assert.False(t, value.Fold(sample(99_000, 1, mondayMidnight)))
	assert.False(t, value.Fold(sample(99_000, 0, nextWeek+hour)), "no drivers, no sample")
	assert.Equal(t, int64(1), value.Observations)
	assert.InDelta(t, 10_000.0, value.Estimate(), 1e-9)
}

func TestEstimate_SettlesOnAFloorWeight(t *testing.T) {
	t.Parallel()

	value := &dispatchmarket.MarketValue{
		ValueCents:   10_000,
		Weeks:        10,
		WeekSumCents: 20_000,
		WeekSamples:  1,
	}

	assert.InDelta(t, 12_000.0, value.Estimate(), 1e-9)
	assert.InDelta(t, 0.5, dispatchmarket.Alpha(1), 1e-12)
	assert.InDelta(t, 0.2, dispatchmarket.Alpha(50), 1e-12)
}

func TestAggregate_SumsDriversByCellAndSlot(t *testing.T) {
	t.Parallel()

	samples := dispatchmarket.Aggregate(mondayMidnight, []dispatchmarket.DriverValue{
		{Start: chicago, AvailableAt: mondayMidnight, ValueCents: 12_000},
		{Start: joliet, AvailableAt: mondayMidnight + hour, ValueCents: 0},
		{Start: dallas, AvailableAt: mondayMidnight, ValueCents: 5_000},
		{Start: chicago, AvailableAt: mondayMidnight + 5*hour, ValueCents: 7_000},
		{Start: dispatchmarket.Point{}, AvailableAt: mondayMidnight, ValueCents: 9_000},
	})

	require.Len(t, samples, 3)
	chicagoKey, _ := dispatchmarket.KeyOf(chicago, mondayMidnight)
	var found bool
	for _, s := range samples {
		assert.Equal(t, mondayMidnight, s.ObservedAt)
		if s.Key == chicagoKey {
			found = true
			assert.InDelta(t, 12_000.0, s.SumCents, 1e-9)
			assert.Equal(t, int32(2), s.Drivers, "an idle driver counts as worth nothing")
		}
	}
	assert.True(t, found)
	for i := 1; i < len(samples); i++ {
		prev, cur := samples[i-1].Key, samples[i].Key
		assert.True(t, prev.Cell < cur.Cell || (prev.Cell == cur.Cell && prev.Slot < cur.Slot))
	}
}

func TestFoldAll_CreatesNewCellsForTheTenantAndFoldsExistingOnes(t *testing.T) {
	t.Parallel()

	chicagoKey, _ := dispatchmarket.KeyOf(chicago, mondayMidnight)
	dallasKey, _ := dispatchmarket.KeyOf(dallas, mondayMidnight)
	existing := &dispatchmarket.MarketValue{
		ID:           pulid.ID("dmv_1"),
		Cell:         chicagoKey.Cell,
		Slot:         chicagoKey.Slot,
		WeekIndex:    dispatchmarket.WeekOf(mondayMidnight),
		WeekSumCents: 4_000,
		WeekSamples:  1,
		Observations: 1,
		ObservedAt:   mondayMidnight - hour,
	}

	changed := dispatchmarket.FoldAll(&dispatchmarket.FoldParams{
		OrganizationID: pulid.ID("org_1"),
		BusinessUnitID: pulid.ID("bu_1"),
		Existing:       []*dispatchmarket.MarketValue{existing},
		Samples: []*dispatchmarket.Sample{
			{Key: chicagoKey, SumCents: 6_000, Drivers: 1, ObservedAt: mondayMidnight},
			{Key: dallasKey, SumCents: 3_000, Drivers: 1, ObservedAt: mondayMidnight},
		},
	})

	require.Len(t, changed, 2)
	assert.Same(t, existing, changed[0])
	assert.InDelta(t, 5_000.0, existing.Estimate(), 1e-9)
	assert.Equal(t, pulid.ID("org_1"), changed[1].OrganizationID)
	assert.Equal(t, pulid.ID("bu_1"), changed[1].BusinessUnitID)
	assert.Equal(t, dallasKey.Slot, changed[1].Slot)
	assert.True(t, changed[1].ID.IsNil(), "the repository assigns new rows their id")
}

func TestValuer_DiscountsAndShrinksWhatItHasSeenLittleOf(t *testing.T) {
	t.Parallel()

	chicagoKey, _ := dispatchmarket.KeyOf(chicago, mondayMidnight)
	valuer := dispatchmarket.NewValuer([]*dispatchmarket.MarketValue{{
		Cell:         chicagoKey.Cell,
		Slot:         chicagoKey.Slot,
		WeekSumCents: 10_000,
		WeekSamples:  1,
		Observations: 4,
	}})

	assert.Equal(t, int64(4_000), valuer.EndMarketCents(
		joliet.Latitude, joliet.Longitude, mondayMidnight+hour,
	), "half confident at four observations, then the 0.8 look-ahead discount")
	assert.Equal(t, int64(2_000), valuer.EndMarketCents(
		chicago.Latitude, chicago.Longitude, mondayMidnight+30*hour,
	), "another time of the week falls back to the cell's average at half weight")
	assert.Zero(t, valuer.EndMarketCents(dallas.Latitude, dallas.Longitude, mondayMidnight))
	assert.Zero(t, valuer.EndMarketCents(0, 0, mondayMidnight))

	var none *dispatchmarket.Valuer
	assert.Zero(t, none.EndMarketCents(chicago.Latitude, chicago.Longitude, mondayMidnight))
	assert.Zero(t, none.Len())
}

func TestCellsOf_ListsEachKnownCellOnce(t *testing.T) {
	t.Parallel()

	cells := dispatchmarket.CellsOf([]dispatchmarket.Point{chicago, joliet, dallas, {}})

	assert.Len(t, cells, 2)
	assert.True(t, cells[0] < cells[1])
}

func TestBoundsOf_HoldsTheCellsCenterAndCoversItsCorners(t *testing.T) {
	t.Parallel()

	dallas := dispatchmarket.Point{Latitude: 32.7767, Longitude: -96.797}
	cell, ok := dispatchmarket.CellOf(dallas)
	require.True(t, ok)
	bounds := dispatchmarket.BoundsOf(cell)

	assert.True(t, bounds.Contains(dallas))
	assert.True(t, bounds.Contains(dispatchmarket.CenterOf(cell)))
	assert.False(t, bounds.Contains(dispatchmarket.Point{Latitude: 30.2672, Longitude: -97.7431}),
		"Austin is in another area")
	assert.Greater(t, dispatchmarket.CoverMeters(cell), 50_000.0,
		"the circle reaches the corners of a tile over a degree wide")
	assert.Less(t, dispatchmarket.CoverMeters(cell), 150_000.0)
}
