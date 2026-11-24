package boardcardservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingboard"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFillReadsUnderTheReadersOwnRowScope(t *testing.T) {
	t.Parallel()

	f := newFixture()
	shipment := pulid.MustNew("shp_")
	f.shipments.known[shipment] = boardreading.Summary{Label: "S2610-0412"}
	shown := parsed(t, map[string]any{
		"title": "One load", "verdict": "It is [late][1].",
		"blocks": []any{map[string]any{"type": "records", "records": []any{shipment.String()}}},
	})

	_, err := f.svc.Fill(t.Context(), f.request(shown))

	require.NoError(t, err)
	require.Len(t, f.scopes.principals, 1, "the reader's own read scope is bound once per fill")
	assert.Equal(t, f.actor.UserID, f.scopes.principals[0].UserID)
	require.Len(t, f.shipments.actors, 1)
	assert.Same(t, f.actor, f.shipments.actors[0], "the reader is told who reads")
	assert.Equal(t, []bool{true}, f.shipments.scoped,
		"the reader's query runs under the reader's row scope")
}

func TestFillWithholdsWhatARecordOutsideTheReadersScopeWouldHaveRead(t *testing.T) {
	t.Parallel()

	f := newFixture()
	seen, hidden := pulid.MustNew("shp_"), pulid.MustNew("shp_")
	f.shipments.known[seen] = boardreading.Summary{Label: "S-SEEN"}
	f.late.readings = map[pulid.ID]boardreading.Reading{
		seen:   lateness(600),
		hidden: lateness(6000),
	}
	records := []any{seen.String(), hidden.String()}
	shown := parsed(t, map[string]any{
		"title": "Late loads",
		"readings": []any{
			map[string]any{
				"key": "late", "label": "Late", "records": records, "aggregate": "count",
			},
			map[string]any{
				"key": "furthest", "label": "Furthest", "records": records,
				"measure": "lateness", "aggregate": "max",
			},
		},
		"verdict": "{late} are late, the furthest {furthest} behind, [here][1].",
		"blocks": []any{
			map[string]any{"type": "records", "records": records, "measure": "lateness"},
		},
	})

	filled, err := f.svc.Fill(t.Context(), f.request(shown))

	require.NoError(t, err)
	rows := filled.Blocks[0].Rows
	require.Len(t, rows, 2)
	var outside int
	for _, row := range rows {
		if row.ID != hidden {
			continue
		}
		outside++
		assert.False(t, row.Readable)
		assert.Empty(t, row.Label, "a record outside the scope never shows its label")
		assert.Empty(t, row.Status)
		assert.Nil(t, row.Measure, "nor its measure")
	}
	assert.Equal(t, 1, outside)
	assert.Nil(t, filled.Reading("late").Value,
		"a count over a record the reader may not see is withheld, not one short")
	assert.Nil(t, filled.Reading("furthest").Value,
		"the furthest behind of records the reader cannot all see is withheld")
}

func TestFillReadsNoMetricAboutARecordOutsideTheReadersScope(t *testing.T) {
	t.Parallel()

	f := newFixture()
	periodKind := recordkind.Kind(permission.ResourceFiscalPeriod)
	periods := &fakeRecords{kind: periodKind, known: map[pulid.ID]boardreading.Summary{}}
	closing := &fakeMetric{
		name:    accountingboard.MetricPeriodClose,
		figures: map[string]*boardcatalog.Value{"blocking": value(boardcatalog.ValueCount, "3")},
	}
	f.svc.readers.Records[periodKind] = periods
	f.svc.readers.Metrics[accountingboard.MetricPeriodClose] = closing
	f.permissions.allowed[permission.ResourceFiscalPeriod] = true
	f.permissions.allowed[permission.ResourceJournalEntry] = true
	seen, hidden := pulid.MustNew("fp_"), pulid.MustNew("fp_")
	periods.known[seen] = boardreading.Summary{Label: "October"}
	board := func(period pulid.ID) map[string]any {
		return map[string]any{
			"title": "The close", "verdict": "Here is [what holds it][1].",
			"blocks": []any{map[string]any{
				"type": "metric", "metric": accountingboard.MetricPeriodClose,
				"for": period.String(),
			}},
		}
	}

	filled, err := f.svc.Fill(t.Context(), f.request(parsed(t, board(hidden))))
	require.NoError(t, err)
	assert.Zero(t, closing.calls, "a metric about a record the reader cannot see is not read")
	assert.Nil(t, filled.Blocks[0].Metric)

	_, err = f.svc.Fill(t.Context(), f.request(parsed(t, board(seen))))
	require.NoError(t, err)
	assert.Equal(t, 1, closing.calls)
	assert.Same(t, f.actor, closing.asked[0].Actor)
}
