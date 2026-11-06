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
	"github.com/emoss08/trenova/internal/modules/operations/ports/demandforecastports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeOutlook struct {
	outlook *demandforecastports.Outlook
	lastReq *demandforecastports.OutlookRequest
}

func (f *fakeOutlook) Outlook(
	_ context.Context,
	req *demandforecastports.OutlookRequest,
) (*demandforecastports.Outlook, error) {
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
	assert.Equal(t, 1, hours[toolschema.KeyMinimum])
	assert.Equal(t, demandforecastports.MaxOutlookHours, hours[toolschema.KeyMaximum])
	clamped, _ := hours[toolschema.KeyClamp].(bool)
	assert.True(t, clamped, "a value past the bounds is clamped, not refused")
	assert.Contains(t, hours[toolschema.KeyDescription], "48 unless you ask, at most 96")

	limit, ok := properties[querytool.ParamLimit].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, demandforecastports.MaxOutlookLimit, limit[toolschema.KeyMaximum])
	assert.Contains(t, limit[toolschema.KeyDescription], "20 unless you ask, at most 50")

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

func TestForecastFreight_DefaultsAndBoundsHoursAndLimit(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		params map[string]any
		hours  int
		limit  int
	}{
		{name: "defaults", params: map[string]any{}, hours: 48, limit: 20},
		{
			name:   "past the bounds",
			params: map[string]any{"hoursAhead": 500, "limit": 500},
			hours:  96,
			limit:  50,
		},
		{name: "zero", params: map[string]any{"hoursAhead": 0, "limit": 0}, hours: 48, limit: 20},
		{name: "negative", params: map[string]any{"hoursAhead": -6}, hours: 48, limit: 20},
		{
			name:   "numbers sent as text",
			params: map[string]any{"hoursAhead": "12", "limit": "5"},
			hours:  12,
			limit:  5,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			outlook := &fakeOutlook{outlook: &demandforecastports.Outlook{Quality: readyQuality()}}
			result, err := NewForecastDemandTool(outlook).Query(
				t.Context(),
				querytooltest.TestParams(tc.params),
			)
			require.NoError(t, err)
			require.NotNil(t, outlook.lastReq)
			assert.Equal(t, tc.hours, outlook.lastReq.Hours)
			assert.Equal(t, tc.limit, outlook.lastReq.Limit)
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

			outlook := &fakeOutlook{outlook: &demandforecastports.Outlook{
				Quality: tc.quality,
				Windows: []*demandforecastports.OutlookWindow{{
					AreaLabel:   "Dallas, TX",
					WindowStart: tuesdayMorning(),
					WindowEnd:   tuesdayMorning() + 12*3600,
					TypicalHigh: 4,
				}},
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
			assert.Contains(t, view.Note, "no windows are listed")
			assert.Equal(t, 0, view.Count, "an unproven forecast names no area")
			assert.NotNil(t, view.Windows)
			assert.Empty(t, view.Windows)
		})
	}
}

func TestForecastFreight_ReadsAReadyOutlookByAreaAndWindow(t *testing.T) {
	t.Parallel()

	acme := pulid.MustNew("cus_")
	start := tuesdayMorning()
	outlook := &fakeOutlook{outlook: &demandforecastports.Outlook{
		Quality:     readyQuality(),
		GeneratedAt: start - 18*3600,
		Windows: []*demandforecastports.OutlookWindow{
			{
				Cell:             41_234,
				AreaLabel:        "Dallas, TX",
				WindowStart:      start,
				WindowEnd:        start + 12*3600,
				TypicalLow:       3,
				TypicalHigh:      5,
				Booked:           1,
				ExpectedUnbooked: 3.44,
				DriversExpected:  1,
				Shortfall:        3.06,
				ValueCents:       24_800,
				Customers: []demandforecast.CustomerShare{
					{CustomerID: acme, Name: "Acme Foods", Expected: 2.96},
				},
				Explanation: "Ending near Dallas, TX is worth $248: 3–5 loads usually appear " +
					"there Tue 06:00–18:00 (Acme Foods about 3), 1 already booked, 1 other " +
					"driver expected.",
			},
			nil,
			{
				WindowStart: start + 4*3600,
				WindowEnd:   start + 16*3600,
				TypicalHigh: 1,
			},
		},
	}}

	result, err := NewForecastDemandTool(outlook).Query(
		t.Context(),
		querytooltest.TestParamsIn("America/Chicago", map[string]any{"hoursAhead": 24}),
	)
	require.NoError(t, err)

	view, ok := result.(*freightOutlookView)
	require.True(t, ok)
	assert.Equal(t, forecastQualityView{
		Status: "Ready",
		Meaning: "Tested on the last 8 weeks of this organization's own history: 82% of " +
			"actual counts fell inside the forecast range, and its misses were 31% smaller " +
			"than repeating the same slot last week. Plans use it when Anticipate freight is on.",
		HistoryWeeks:            26,
		HistoryLoads:            1_840,
		WeeksTested:             8,
		IntervalCoveragePercent: 82,
		SkillPercent:            31,
		FittedAt:                "Sun 08 Nov 2026 18:00 CST",
	}, view.Quality)
	assert.Equal(t, 24, view.Hours)
	assert.Equal(t, "Mon 09 Nov 2026 12:00 CST", view.GeneratedAt)
	assert.Equal(t, 2, view.Count, "a nil window is skipped")
	assert.Contains(t, view.Note, "2 windows in the next 24 hours, the most short of drivers")

	require.Len(t, view.Windows, 2)
	assert.Equal(t, freightWindowView{
		Area:               "Dallas, TX",
		WindowStart:        "Tue 10 Nov 2026 06:00 CST",
		WindowEnd:          "Tue 10 Nov 2026 18:00 CST",
		TypicalLow:         3,
		TypicalHigh:        5,
		Booked:             1,
		ExpectedUnbooked:   3.4,
		DriversExpected:    1,
		Shortfall:          3.1,
		OneMoreDriverWorth: "$248",
		Customers: []forecastCustomerView{
			{CustomerID: acme.String(), Name: "Acme Foods", ExpectedLoads: 3},
		},
		Explanation: outlook.outlook.Windows[0].Explanation,
	}, view.Windows[0])

	unnamed := view.Windows[1]
	assert.Equal(t, unnamedArea, unnamed.Area)
	assert.Equal(t, "$0", unnamed.OneMoreDriverWorth)
	assert.NotNil(t, unnamed.Customers, "an empty list, never null")
}

func TestForecastFreight_SaysWhenNothingIsLikely(t *testing.T) {
	t.Parallel()

	outlook := &fakeOutlook{outlook: &demandforecastports.Outlook{Quality: readyQuality()}}
	result, err := NewForecastDemandTool(outlook).Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{}),
	)
	require.NoError(t, err)

	view, ok := result.(*freightOutlookView)
	require.True(t, ok)
	assert.Equal(t, 0, view.Count)
	assert.NotNil(t, view.Windows)
	assert.Equal(t,
		"The forecast lists no area with freight beyond the drivers expected there in the "+
			"next 48 hours.",
		view.Note,
	)
}
