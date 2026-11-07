package dispatchquerytools

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/freightmove"
	"github.com/emoss08/trenova/internal/modules/operations/ports/freightoutlookports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeOutlook struct {
	outlook *freightoutlookports.Outlook
	lastReq *freightoutlookports.Request
}

func (f *fakeOutlook) Outlook(
	_ context.Context,
	req *freightoutlookports.Request,
) (*freightoutlookports.Outlook, error) {
	f.lastReq = req

	return f.outlook, nil
}

func tuesdayMorning() int64 {
	return time.Date(2026, time.November, 10, 12, 0, 0, 0, time.UTC).Unix()
}

func readyQuality() *demandforecast.Quality {
	return &demandforecast.Quality{
		Status:           demandforecast.StatusReady,
		HistoryWeeks:     26,
		HistoryLoads:     1_840,
		WeeksTested:      8,
		LoadsTested:      560,
		IntervalCoverage: 0.82,
		Skill:            0.31,
		FittedAt:         tuesdayMorning() - 36*3600,
	}
}

func TestForecastFreight_DeclaresBoundedHoursAndLimitAsATenantRead(t *testing.T) {
	t.Parallel()

	tool := NewForecastDemandTool(&fakeOutlook{})
	schema := tool.ParamSchema()
	additional, ok := schema[toolschema.KeyAdditionalProperties].(bool)
	require.True(t, ok)
	assert.False(t, additional)
	properties, ok := schema[toolschema.KeyProperties].(map[string]any)
	require.True(t, ok)
	assert.Len(t, properties, 2)

	hours, ok := properties[paramHoursAhead].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, toolschema.TypeInteger, hours[toolschema.KeyType])
	assert.Equal(t, 4, hours[toolschema.KeyMinimum])
	assert.Equal(t, maxForecastHours, hours[toolschema.KeyMaximum])
	clamped, _ := hours[toolschema.KeyClamp].(bool)
	assert.True(t, clamped, "a value past the bounds is clamped, not refused")
	assert.Contains(t, hours[toolschema.KeyDescription], "24 unless you ask, at most 48")

	limit, ok := properties[querytool.ParamLimit].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, maxForecastLimit, limit[toolschema.KeyMaximum])

	searchable, ok := tool.(agenttoolports.SearchableTool)
	require.True(t, ok)
	assert.Contains(t, searchable.SearchTerms(), "outlook")

	policy := tool.Policy()
	assert.Equal(t, "dispatch_forecast_demand", policy.Name)
	assert.Equal(t, permission.ResourceShipmentMove, policy.Resource)
	assert.Equal(t, permission.OpRead, policy.Operation)
	assert.Equal(t, agenttooltrust.ToolKindQuery, policy.Kind)
	assert.Equal(t, agenttooltrust.ToolScopeTenant, policy.Scope)
	assert.Equal(t, agenttooltrust.ExternalReadNever, policy.ReadsExternal)
}

func TestForecastFreight_DefaultsAndBoundsHours(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		params map[string]any
		hours  int
	}{
		{name: "default", params: map[string]any{}, hours: 24},
		{name: "past the bounds", params: map[string]any{"hoursAhead": 500}, hours: 48},
		{name: "zero", params: map[string]any{"hoursAhead": 0}, hours: 24},
		{name: "negative", params: map[string]any{"hoursAhead": -6}, hours: 24},
		{name: "numbers sent as text", params: map[string]any{"hoursAhead": "12"}, hours: 12},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			outlook := &fakeOutlook{outlook: &freightoutlookports.Outlook{Quality: readyQuality()}}
			result, err := NewForecastDemandTool(outlook).Query(
				t.Context(),
				querytooltest.TestParams(tc.params),
			)
			require.NoError(t, err)
			require.NotNil(t, outlook.lastReq)
			assert.Equal(t, querytooltest.TestOrganizationID, outlook.lastReq.TenantInfo.OrgID)
			assert.Equal(t, querytooltest.TestBusinessUnitID, outlook.lastReq.TenantInfo.BuID)

			view, ok := result.(*freightOutlookView)
			require.True(t, ok)
			assert.Equal(t, tc.hours, view.Hours)
		})
	}
}

func TestForecastFreight_SaysPlainlyWhenTheForecastIsNotReady(t *testing.T) {
	t.Parallel()

	noSkill := readyQuality()
	noSkill.Status = demandforecast.StatusNoSkill
	noSkill.Skill = 0.02
	short := &demandforecast.Quality{
		Status:       demandforecast.StatusInsufficientHistory,
		HistoryWeeks: 5,
		HistoryLoads: 120,
		FittedAt:     tuesdayMorning(),
	}

	cases := []struct {
		name    string
		quality *demandforecast.Quality
		status  string
		meaning string
	}{
		{
			name:    "no skill",
			quality: noSkill,
			status:  "NoSkill",
			meaning: "not yet better than repeating the same slot last week",
		},
		{
			name:    "too little history",
			quality: short,
			status:  "InsufficientHistory",
			meaning: "5 weeks and 120 loads, where it needs at least 8 weeks and 200 loads",
		},
		{
			name:    "never fitted",
			quality: nil,
			status:  "NotFitted",
			meaning: "has not been fitted for this organization yet",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			outlook := &fakeOutlook{outlook: &freightoutlookports.Outlook{
				Quality:     tc.quality,
				GeneratedAt: tuesdayMorning(),
				Areas:       []*freightoutlookports.Area{shortArea("Dallas, TX", 2.6)},
				Steps: []freightmove.Step{
					{Kind: freightmove.KindSend, ToLabel: "Dallas, TX"},
				},
			}}
			result, err := NewForecastDemandTool(outlook).Query(
				t.Context(),
				querytooltest.TestParams(map[string]any{}),
			)
			require.NoError(t, err)

			view, ok := result.(*freightOutlookView)
			require.True(t, ok)
			assert.Equal(t, tc.status, view.Quality.Status)
			assert.Contains(t, view.Quality.Meaning, tc.meaning)
			assert.Contains(t, view.Note, "The forecast is not Ready (status "+tc.status+")")
			assert.Empty(t, view.Short, "an unproven forecast names no area")
			assert.Empty(t, view.Spare)
			assert.Empty(t, view.Moves, "an unproven forecast suggests no move")
		})
	}
}

func shortArea(label string, short float64) *freightoutlookports.Area {
	slots := make([]demandforecast.SlotBalance, 12)
	for i := range slots {
		slots[i].Start = tuesdayMorning() + int64(i)*4*3600
	}
	slots[1].Short = short
	slots[1].Low, slots[1].High, slots[1].Booked, slots[1].Drivers = 3, 6, 2, 1

	return &freightoutlookports.Area{
		Balance:    demandforecast.AreaBalance{Label: label, Slots: slots},
		ValueCents: 61_000,
		Customers:  []freightoutlookports.AreaCustomer{{Name: "Acme Foods", Share: 0.6}},
	}
}

func spareArea(label string, spare float64, at int) *freightoutlookports.Area {
	slots := make([]demandforecast.SlotBalance, 12)
	for i := range slots {
		slots[i].Start = tuesdayMorning() + int64(i)*4*3600
	}
	slots[at].Spare = spare

	return &freightoutlookports.Area{
		Balance: demandforecast.AreaBalance{Label: label, Slots: slots},
	}
}

func TestForecastFreight_ReadsAReadyOutlookAsShortAreasSpareAreasAndMoves(t *testing.T) {
	t.Parallel()

	now := tuesdayMorning()
	outlook := &fakeOutlook{outlook: &freightoutlookports.Outlook{
		Quality:     readyQuality(),
		GeneratedAt: now,
		Areas: []*freightoutlookports.Area{
			shortArea("Dallas, TX", 2.6),
			spareArea("Kansas City, MO", 3.2, 2),
			spareArea("Long Beach, CA", 2, 10),
		},
		Steps: []freightmove.Step{
			{
				Kind: freightmove.KindSend, WorkerName: "Ana Ruiz", FromLabel: "Kansas City, MO",
				ToLabel: "Dallas, TX", SlotStart: now + 4*3600, Miles: 501.4, NetCents: 90_700,
			},
			{
				Kind: freightmove.KindKeep, WorkerName: "Dana Ortiz", ToLabel: "Dallas, TX",
				SlotStart: now + 4*3600, NetCents: 54_000, ProNumber: "PRO-1",
				PartnerName: "Joe Kowalski",
			},
			{
				Kind: freightmove.KindCall, ToLabel: "Dallas, TX", SlotStart: now + 4*3600,
				CustomerName: "Acme Foods", Loads: 3.96,
			},
			{
				Kind: freightmove.KindSend, WorkerName: "Late Mover", ToLabel: "Dallas, TX",
				SlotStart: now + 40*3600, NetCents: 10_000,
			},
		},
	}}

	result, err := NewForecastDemandTool(outlook).Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"hoursAhead": 24}),
	)
	require.NoError(t, err)
	view, ok := result.(*freightOutlookView)
	require.True(t, ok)

	require.Len(t, view.Short, 1)
	assert.Equal(t, "Dallas, TX", view.Short[0].Area)
	assert.InDelta(t, 2.6, view.Short[0].LoadsWithNoTruck, 1e-9)
	assert.Equal(t, 3, view.Short[0].LikelyLow)
	assert.Equal(t, 6, view.Short[0].LikelyHigh)
	assert.Equal(t, "$610", view.Short[0].OneMoreDriverWorth)
	assert.Equal(t, []string{"Acme Foods"}, view.Short[0].Customers)

	require.Len(t, view.Spare, 1, "a spare window past the hours asked is left out")
	assert.Equal(t, "Kansas City, MO", view.Spare[0].Area)
	assert.InDelta(t, 3.2, view.Spare[0].DriversToSpare, 1e-9)

	require.Len(t, view.Moves, 3, "a move for a window past the hours asked is left out")
	assert.Equal(t, forecastMoveView{
		Kind: "Send", Driver: "Ana Ruiz", From: "Kansas City, MO", To: "Dallas, TX",
		Window: view.Moves[0].Window, Miles: 501, Worth: "$907",
	}, view.Moves[0])
	assert.Equal(t, "Joe Kowalski", view.Moves[1].HandTo)
	assert.Equal(t, "PRO-1", view.Moves[1].ProNumber)
	assert.Equal(t, "Call", view.Moves[2].Kind)
	assert.Empty(t, view.Moves[2].Driver)
	assert.Equal(t, "Acme Foods", view.Moves[2].Customer)
	assert.InDelta(t, 4, view.Moves[2].LoadsMissed, 1e-9)
	assert.Contains(t, view.Note, "1 areas run short of trucks in the next 24 hours")
}

func TestForecastFreight_SaysWhenNoAreaIsShort(t *testing.T) {
	t.Parallel()

	outlook := &fakeOutlook{outlook: &freightoutlookports.Outlook{
		Quality:     readyQuality(),
		GeneratedAt: tuesdayMorning(),
		Areas:       []*freightoutlookports.Area{spareArea("Kansas City, MO", 3, 1)},
	}}
	result, err := NewForecastDemandTool(outlook).Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{}),
	)
	require.NoError(t, err)
	view, ok := result.(*freightOutlookView)
	require.True(t, ok)

	assert.Empty(t, view.Short)
	require.Len(t, view.Spare, 1)
	assert.Contains(t, view.Note, "No area is short of trucks in the next 24 hours")
}
