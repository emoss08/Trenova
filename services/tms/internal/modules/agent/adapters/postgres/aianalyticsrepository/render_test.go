package aianalyticsrepository

import (
	"testing"

	"github.com/emoss08/keel"
	keelsnap "github.com/emoss08/keel/keeltest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics/aianalyticsdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/stretchr/testify/require"
)

func TestReadQueriesRender(t *testing.T) {
	t.Parallel()

	filter := aianalyticsports.Filter{Since: 1_790_000_000, Task: "AssistantChat"}
	byProvider := filter
	byProvider.CompareBy = aianalytics.CompareByProvider
	tools := aianalyticsports.ToolFailureRequest{
		Filter:    byProvider,
		ToolName:  "search",
		SeriesKey: "aip_1",
		Limit:     5,
	}
	for name, query := range map[string]keeltest.Renderer{
		"totals":            totalsQuery(&filter),
		"deployments":       deploymentsQuery(&filter),
		"outcome durations": outcomeDurationsQuery(&filter, aianalytics.GroupingProvider),
		"trend": trendQuery(&aianalyticsports.TrendRequest{
			Filter:       filter,
			BucketStarts: []int64{1_790_000_000, 1_790_086_400},
		}),
		"tool cells":        toolCellsQuery(&filter),
		"task cells":        taskCellsQuery(&filter),
		"decisions":         decisionsQuery(&filter),
		"ratings":           ratingsQuery(&filter),
		"suggestions":       suggestionsQuery(&aianalyticsports.SuggestionRequest{Filter: filter}),
		"provider calls":    providerCallsQuery(&filter),
		"first token":       firstTokenQuery(&filter),
		"provider hours":    providerHoursQuery(&filter),
		"provider versions": providerVersionsQuery(&filter),
		"tool causes":       toolCausesQuery(&tools),
		"tool examples":     toolExamplesQuery(&tools),
		"palette":           paletteQuery(1_790_000_000),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			keeltest.Snapshot(t, query)
		})
	}
}

func TestRefreshStatementsRender(t *testing.T) {
	t.Parallel()

	refresh := &unitRefresh{from: 1_790_000_000, to: 1_790_003_600, stamp: 7}
	for name, step := range map[string]struct {
		source          *keel.Query[*keel.Row]
		withDecisionRun bool
	}{
		"runs":  {ownedUnitsSource(&runSource, refresh), true},
		"turns": {ownedUnitsSource(&turnSource, refresh), true},
		"calls": {callUnitsSource(refresh), false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			w := aianalyticsdb.WorkUnits
			sql, args, err := w.UpsertFromSQL(
				keeltest.TenantContext(t),
				keeltest.RenderDB(t),
				step.source,
				unitConflict(),
				unitInsertColumns(w, step.withDecisionRun)...,
			)
			require.NoError(t, err)
			keelsnap.Snapshot(t, sql, args)
		})
	}

	t.Run("clear unit tools", func(t *testing.T) {
		t.Parallel()
		keeltest.Snapshot(t, clearUnitToolsQuery(7))
	})

	t.Run("unit tools", func(t *testing.T) {
		t.Parallel()
		tools := aianalyticsdb.UnitTools
		sql, args, err := tools.InsertFromSQL(
			keeltest.TenantContext(t),
			keeltest.RenderDB(t),
			unitToolsSource(7),
			unitToolColumns(tools)...,
		)
		require.NoError(t, err)
		keelsnap.Snapshot(t, sql, args)
	})

	t.Run("provider hours", func(t *testing.T) {
		t.Parallel()
		hours := aianalyticsdb.ProviderHours
		sql, args, err := hours.InsertFromSQL(
			keeltest.TenantContext(t),
			keeltest.RenderDB(t),
			providerHoursSource(1_789_996_400, 1_790_003_600),
			providerHourColumns(hours)...,
		)
		require.NoError(t, err)
		keelsnap.Snapshot(t, sql, args)
	})
}
