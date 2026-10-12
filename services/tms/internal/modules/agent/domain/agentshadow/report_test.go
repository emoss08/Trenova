package agentshadow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSetsEachWriteBesideWhatAPersonDid(t *testing.T) {
	t.Parallel()

	const day = 24 * 60 * 60
	proposals := []Proposal{
		{TargetID: "shp_1", CreatedAt: 1000, Fields: []string{"status"}},
		{TargetID: "shp_2", CreatedAt: 1000, Fields: []string{"moves.stops[0].locationId"}},
		{TargetID: "shp_3", CreatedAt: 1000, Fields: []string{"rate"}},
		{TargetID: "shp_4", CreatedAt: 1000},
		{TargetID: "shp_5", CreatedAt: 1000, Fields: []string{"status"}},
		{TargetID: "shp_6", CreatedAt: 1000, Failed: true},
		{CreatedAt: 1000},
	}
	changes := []PersonChange{
		{ResourceID: "shp_1", Timestamp: 2000, Fields: []string{"status", "updatedAt"}},
		{ResourceID: "shp_2", Timestamp: 2000, Fields: []string{"moves"}},
		{ResourceID: "shp_3", Timestamp: 2000, Fields: []string{"bol"}},
		{ResourceID: "shp_3", Timestamp: 3000, Fields: []string{"rate"}},
		{ResourceID: "shp_4", Timestamp: 1500, Fields: []string{"anything"}},
		{ResourceID: "shp_5", Timestamp: 900, Fields: []string{"status"}},
		{ResourceID: "shp_5", Timestamp: 1000 + 4*day, Fields: []string{"status"}},
	}

	report := Build(30, proposals, changes)

	assert.Equal(t, 7, report.Recorded)
	assert.Equal(t, 3, report.Matched, "shp_1, shp_2 by its top-level field, shp_4 named no fields")
	assert.Equal(t, 1, report.WouldReject, "shp_3: the first change was to something else")
	assert.Equal(t, 1, report.WouldFail)
	assert.Equal(t, 2, report.Unanswered, "shp_5 was changed only outside the window; one had no record")
	rate := report.MatchRate()
	require.NotNil(t, rate)
	assert.InDelta(t, 0.75, *rate, 0.0001)
}

func TestMatchRateIsAbsentUntilPeopleAnswerOne(t *testing.T) {
	t.Parallel()

	assert.Nil(t, Build(7, []Proposal{{TargetID: "shp_1", CreatedAt: 1}}, nil).MatchRate())
}

func TestClampDays(t *testing.T) {
	t.Parallel()

	assert.Equal(t, DefaultDays, ClampDays(0))
	assert.Equal(t, 7, ClampDays(7))
	assert.Equal(t, MaxDays, ClampDays(400))
}
