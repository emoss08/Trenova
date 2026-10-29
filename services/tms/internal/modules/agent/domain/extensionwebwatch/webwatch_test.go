package extensionwebwatch_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionwebwatch"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func watch() *extensionwebwatch.WebWatch {
	return &extensionwebwatch.WebWatch{
		Query:          " FMCSA final rule broker transparency ",
		Purpose:        " FMCSA finalizing the broker transparency rule ",
		IncludeDomains: []string{"  FMCSA.dot.gov. ", "federalregister.gov", "fmcsa.dot.gov"},
		Cadence:        extensionwebwatch.WebWatchCadenceDaily,
		Status:         extensionwebwatch.WebWatchStatusActive,
		EndsAt:         2_000,
	}
}

func TestAWatchIsTrimmedAndNamesItsSitesByHostname(t *testing.T) {
	t.Parallel()

	w := watch()
	w.Normalize()

	assert.Equal(t, "FMCSA final rule broker transparency", w.Query)
	assert.Equal(t, "FMCSA finalizing the broker transparency rule", w.Purpose)
	assert.Equal(t, []string{"fmcsa.dot.gov", "federalregister.gov"}, w.IncludeDomains)

	multiErr := errortypes.NewMultiError()
	w.Validate(multiErr)
	assert.False(t, multiErr.HasErrors())
}

func TestAWatchNeedsASearchAPurposeAndACadence(t *testing.T) {
	t.Parallel()

	w := &extensionwebwatch.WebWatch{Cadence: "Hourly"}
	multiErr := errortypes.NewMultiError()
	w.Validate(multiErr)

	require.True(t, multiErr.HasErrors())
	fields := make([]string, 0, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields = append(fields, fieldErr.Field)
	}
	assert.ElementsMatch(t, []string{"query", "purpose", "cadence"}, fields)
}

func TestAWatchNamesAtMostTwentySitesAndNoWildcards(t *testing.T) {
	t.Parallel()

	tooMany := watch()
	tooMany.IncludeDomains = make([]string, 0, extensionwebwatch.MaxWebWatchDomains+1)
	for idx := range extensionwebwatch.MaxWebWatchDomains + 1 {
		tooMany.IncludeDomains = append(tooMany.IncludeDomains,
			"site"+strings.Repeat("a", idx+1)+".com")
	}
	multiErr := errortypes.NewMultiError()
	tooMany.Validate(multiErr)
	assert.True(t, multiErr.HasErrors())

	wildcard := watch()
	wildcard.IncludeDomains = []string{"*.dot.gov"}
	multiErr = errortypes.NewMultiError()
	wildcard.Validate(multiErr)
	require.True(t, multiErr.HasErrors())
	assert.Equal(t, "sites", multiErr.Errors[0].Field)
}

func TestAWatchLastsAMonthByDefaultAndAtMostAQuarter(t *testing.T) {
	t.Parallel()

	days, err := extensionwebwatch.WatchDays(0)
	require.NoError(t, err)
	assert.Equal(t, 30, days)

	days, err = extensionwebwatch.WatchDays(90)
	require.NoError(t, err)
	assert.Equal(t, 90, days)

	for _, raw := range []int{-1, 91, 365} {
		_, err = extensionwebwatch.WatchDays(raw)
		assert.Error(t, err, raw)
	}
}

func TestAWatchEndsOnceAndKeepsHowItEndedFirst(t *testing.T) {
	t.Parallel()

	w := watch()
	assert.False(t, w.Due(1_999))
	assert.True(t, w.Due(2_000))

	w.End(extensionwebwatch.WebWatchStatusFailed, "  Exa turned the watch off  ", 2_100)
	w.End(extensionwebwatch.WebWatchStatusStopped, "", 2_200)

	assert.Equal(t, extensionwebwatch.WebWatchStatusFailed, w.Status)
	assert.Equal(t, "Exa turned the watch off", w.Failure)
	require.NotNil(t, w.FinishedAt)
	assert.Equal(t, int64(2_100), *w.FinishedAt)
	assert.False(t, w.Active())
	assert.False(t, w.Due(3_000), "an ended watch is never due")
}

func TestEveryRunIsBilledWhetherOrNotItFoundAnything(t *testing.T) {
	t.Parallel()

	w := watch()
	w.CountRun(1_200, extensionwebwatch.ExaMonitorRunCostUSD)
	w.CountRun(1_100, extensionwebwatch.ExaMonitorRunCostUSD)

	assert.Equal(t, 2, w.RunCount)
	assert.True(t, decimal.RequireFromString("0.03").Equal(w.CostUSD))
	require.NotNil(t, w.LastRunAt)
	assert.Equal(t, int64(1_200), *w.LastRunAt, "the newest run is the last one")
}

func TestAWatchsChecksFollowItsCadence(t *testing.T) {
	t.Parallel()

	assert.Equal(t, int64(120), extensionwebwatch.WebWatchChecks(
		extensionwebwatch.WebWatchCadenceSixHours, 30))
	assert.Equal(t, int64(30), extensionwebwatch.WebWatchChecks(
		extensionwebwatch.WebWatchCadenceDaily, 30))
	assert.Equal(t, int64(4), extensionwebwatch.WebWatchChecks(
		extensionwebwatch.WebWatchCadenceWeekly, 30))
	assert.Zero(t, extensionwebwatch.WebWatchChecks("Hourly", 30))

	assert.Equal(t, "every six hours", extensionwebwatch.WebWatchCadenceSixHours.Words())
	assert.Equal(t, "once a day", extensionwebwatch.WebWatchCadenceDaily.Words())
	assert.Equal(t, "once a week", extensionwebwatch.WebWatchCadenceWeekly.Words())
}

func TestOnlyPagesAtAWebAddressAreKept(t *testing.T) {
	t.Parallel()

	raw := []extensionwebwatch.WebWatchResult{
		{Title: "  Final rule  ", URL: " https://www.federalregister.gov/d/2026-1 "},
		{Title: "Script", URL: "javascript:alert(1)"},
		{Title: "Data", URL: "data:text/html,<b>x</b>"},
		{Title: "Relative", URL: "/news/rule"},
		{Title: "No address", URL: ""},
		{Title: "Too long", URL: "https://example.com/" + strings.Repeat("a", 2048)},
		{Title: "Plain", URL: "http://fmcsa.dot.gov/news", PublishedDate: " 2026-10-01 "},
	}

	kept := extensionwebwatch.BoundedResults(raw)

	assert.Equal(t, []extensionwebwatch.WebWatchResult{
		{Title: "Final rule", URL: "https://www.federalregister.gov/d/2026-1"},
		{Title: "Plain", URL: "http://fmcsa.dot.gov/news", PublishedDate: "2026-10-01"},
	}, kept)
}

func TestARunKeepsAtMostTenPages(t *testing.T) {
	t.Parallel()

	raw := make([]extensionwebwatch.WebWatchResult, 0, 15)
	for range 15 {
		raw = append(raw, extensionwebwatch.WebWatchResult{URL: "https://example.com/a"})
	}

	assert.Len(t, extensionwebwatch.BoundedResults(raw), extensionwebwatch.WebWatchResultsPerRun)
}

func TestTheWebhookPathNamesTheTenantAndTheWatch(t *testing.T) {
	t.Parallel()

	org := pulid.MustNew("org_")
	bu := pulid.MustNew("bu_")
	id := pulid.MustNew(extensionwebwatch.WebWatchIDPrefix)

	assert.Equal(t,
		"/webhooks/agent-extensions/web-watches/"+org.String()+"/"+bu.String()+"/"+
			id.String()+"/",
		extensionwebwatch.WebWatchWebhookPath(org, bu, id))
}

func TestExaSetsHowManyWatchesRunAtOnce(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 25, extensionwebwatch.MaxActiveWebWatches(agentextension.TypeExa,
		map[string]string{agentextension.ConfigKeyMaxActiveWatches: "25"}))
	assert.Equal(t, agentextension.DefaultMaxActiveWebWatches,
		extensionwebwatch.MaxActiveWebWatches(agentextension.TypeExa, map[string]string{}))
	assert.Equal(t, agentextension.DefaultMaxActiveWebWatches,
		extensionwebwatch.MaxActiveWebWatches(agentextension.TypeExa,
			map[string]string{agentextension.ConfigKeyMaxActiveWatches: "500"}),
		"a stored value out of bounds falls back to the default")
}

func TestAVendorThatCannotWatchIsNeverOfferedTheWatchTools(t *testing.T) {
	t.Parallel()

	parallel := agentextension.Active{
		agentextension.CapabilityWebResearch: {
			Type:         agentextension.TypeParallel,
			Availability: agentextension.AvailabilityAllAgents,
		},
	}
	exa := agentextension.Active{
		agentextension.CapabilityWebResearch: {
			Type:         agentextension.TypeExa,
			Availability: agentextension.AvailabilityAllAgents,
		},
	}

	for _, name := range agentextension.WebWatchTools {
		_, served := parallel.Serving(name)
		assert.False(t, served, name)
		assert.NotContains(t, parallel.Grants(), name)

		_, served = exa.Serving(name)
		assert.True(t, served, name)
		assert.Contains(t, exa.Grants(), name)
	}
	_, served := parallel.Serving(agentextension.ToolWebSearch)
	assert.True(t, served, "the vendor still serves the rest of web research")
}
