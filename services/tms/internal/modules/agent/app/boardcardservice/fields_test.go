package boardcardservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingboard"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/operationsboard"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	entryKind  = recordkind.Kind(permission.ResourceJournalEntry)
	periodKind = recordkind.Kind(permission.ResourceFiscalPeriod)
)

type closeFixture struct {
	*fixture
	period  pulid.ID
	closing *fakeMetric
	entries *fakeRecords
	amount  *fakeMeasure
}

func newCloseFixture() *closeFixture {
	f := &closeFixture{fixture: newFixture(), period: pulid.MustNew("fp_")}
	periods := &fakeRecords{kind: periodKind, known: map[pulid.ID]boardreading.Summary{
		f.period: {Label: "November"},
	}}
	f.entries = &fakeRecords{kind: entryKind, known: map[pulid.ID]boardreading.Summary{}}
	f.amount = &fakeMeasure{kind: entryKind, measure: "amount"}
	f.closing = &fakeMetric{
		name: accountingboard.MetricPeriodClose,
		figures: map[string]*boardcatalog.Value{
			"blocking":      value(boardcatalog.ValueCount, "4"),
			"pendingAmount": value(boardcatalog.ValueMoney, "886508.28"),
		},
	}
	f.svc.readers.Records[periodKind] = periods
	f.svc.readers.Records[entryKind] = f.entries
	f.svc.readers.Measures[boardreaderports.MeasureKey{Kind: entryKind, Measure: "amount"}] = f.amount
	f.svc.readers.Metrics[accountingboard.MetricPeriodClose] = f.closing
	f.permissions.allowed[permission.ResourceFiscalPeriod] = true
	f.permissions.allowed[permission.ResourceJournalEntry] = true

	return f
}

func (f *closeFixture) asAgent(
	shown *board.Board,
	dataAccess permission.FieldSensitivity,
) *assistantports.BoardFillRequest {
	return &assistantports.BoardFillRequest{
		Actor: &permissionports.RequestActor{
			PrincipalType:  permissionports.PrincipalTypeAgent,
			PrincipalID:    pulid.MustNew("agd_"),
			OrganizationID: f.actor.OrganizationID,
			BusinessUnitID: f.actor.BusinessUnitID,
		},
		ThreadID: f.thread,
		Board:    shown,
		Agent:    &assistantports.BoardAgentRead{DataAccess: dataAccess},
	}
}

func closeBoard(t *testing.T, period pulid.ID) *board.Board {
	t.Helper()

	return parsed(t, map[string]any{
		"title": "The close", "verdict": "Here is [what holds it][1].",
		"blocks": []any{map[string]any{
			"type": "metric", "metric": accountingboard.MetricPeriodClose, "for": period.String(),
		}},
	})
}

func entriesBoard(t *testing.T, entry pulid.ID) *board.Board {
	t.Helper()

	return parsed(t, map[string]any{
		"title": "Waiting", "verdict": "These [wait][1].",
		"blocks": []any{map[string]any{
			"type": "records", "records": []any{entry.String()}, "measure": "amount",
		}},
	})
}

func TestAnAgentIsShownNoFigureAboveItsDataAccess(t *testing.T) {
	t.Parallel()

	f := newCloseFixture()

	filled, err := f.svc.Fill(t.Context(), f.asAgent(
		closeBoard(t, f.period), permission.SensitivityInternal,
	))

	require.NoError(t, err)
	require.NotNil(t, filled.Blocks[0].Metric)
	figures := filled.Blocks[0].Metric.Read.Figures
	assert.Nil(t, figures["pendingAmount"],
		"the money waiting is ranked as the journal's totals, above Internal data access")
	require.NotNil(t, figures["blocking"], "the count of checks is not")
	assert.Equal(t, "4", board.FormatValue(figures["blocking"]))
}

func TestAnAgentWhoseDataAccessReachesTheTotalsReadsThem(t *testing.T) {
	t.Parallel()

	f := newCloseFixture()

	filled, err := f.svc.Fill(t.Context(), f.asAgent(
		closeBoard(t, f.period), permission.SensitivityRestricted,
	))

	require.NoError(t, err)
	assert.NotNil(t, filled.Blocks[0].Metric.Read.Figures["pendingAmount"])
}

func TestAPersonsScreenShowsTheFiguresTheirGroupsAllow(t *testing.T) {
	t.Parallel()

	f := newCloseFixture()

	filled, err := f.svc.Fill(t.Context(), f.request(closeBoard(t, f.period)))

	require.NoError(t, err)
	assert.NotNil(t, filled.Blocks[0].Metric.Read.Figures["pendingAmount"],
		"a person's screen is held to the protected groups, as the journal page is")
}

func TestAnAgentMayNotPlaceAFigureAboveItsDataAccess(t *testing.T) {
	t.Parallel()

	f := newCloseFixture()
	shown := parsed(t, map[string]any{
		"title": "The close", "verdict": "Waiting [to post][1].",
		"blocks": []any{map[string]any{"type": "figures", "figures": []any{
			map[string]any{
				"label": "Waiting", "metric": accountingboard.MetricPeriodClose,
				"figure": "pendingAmount", "for": f.period.String(),
			},
		}}},
	})

	err := f.svc.Check(t.Context(), f.asAgent(shown, permission.SensitivityInternal))

	mistake, ok := errors.AsType[*agentruntimeports.PublishMistake](err)
	require.True(t, ok, "expected a mistake for the model, got %v", err)
	assert.Contains(t, mistake.Reason, "may not read the pendingAmount figure")
	assert.NoError(t, f.svc.Check(t.Context(), f.asAgent(shown, permission.SensitivityRestricted)))
}

func TestAnAgentIsShownNoMeasureAboveItsDataAccess(t *testing.T) {
	t.Parallel()

	f := newCloseFixture()
	entry := pulid.MustNew("je_")
	f.entries.known[entry] = boardreading.Summary{Label: "JE-1042"}
	f.amount.readings = map[pulid.ID]boardreading.Reading{entry: {
		Value: boardcatalog.Money(120000, "USD"),
	}}
	shown := entriesBoard(t, entry)

	err := f.svc.Check(t.Context(), f.asAgent(shown, permission.SensitivityInternal))
	mistake, ok := errors.AsType[*agentruntimeports.PublishMistake](err)
	require.True(t, ok, "expected a mistake for the model, got %v", err)
	assert.Contains(t, mistake.Reason, "may not read the amount of")

	filled, err := f.svc.Fill(t.Context(), f.asAgent(shown, permission.SensitivityInternal))
	require.NoError(t, err)
	assert.Zero(t, f.amount.calls, "a measure the reader may not see is never read")
	require.Len(t, filled.Blocks[0].Rows, 1)
	assert.Equal(t, "JE-1042", filled.Blocks[0].Rows[0].Label)
	require.NotNil(t, filled.Blocks[0].Rows[0].Measure)
	assert.Nil(t, filled.Blocks[0].Rows[0].Measure.Value, "drawn as not yours to read, no value")

	_, err = f.svc.Fill(t.Context(), f.asAgent(shown, permission.SensitivityRestricted))
	require.NoError(t, err)
	assert.Equal(t, 1, f.amount.calls)
}

func TestAnAgentIsShownNoMarginItsDataAccessDoesNotReach(t *testing.T) {
	t.Parallel()

	f := newCloseFixture()
	shown := parsed(t, map[string]any{
		"title": "The day", "verdict": "Steady [today][1].",
		"blocks": []any{map[string]any{"type": "figures", "figures": []any{
			map[string]any{"label": "Active", "metric": "activeShipments"},
			map[string]any{"label": "Margin", "metric": "marginPercent"},
		}}},
	})

	filled, err := f.svc.Fill(t.Context(), f.asAgent(shown, permission.SensitivityInternal))
	require.NoError(t, err)
	assert.NotNil(t, filled.Blocks[0].Figures[0].Value)
	assert.Nil(t, filled.Blocks[0].Figures[1].Value,
		"a margin reaches an agent only where its data access reaches Restricted")

	filled, err = f.svc.Fill(t.Context(), f.asAgent(shown, permission.SensitivityRestricted))
	require.NoError(t, err)
	assert.NotNil(t, filled.Blocks[0].Figures[1].Value)
}

func TestARollupOfEveryShipmentIsNotReadByAReaderLimitedToTheirOwn(t *testing.T) {
	t.Parallel()

	f := newFixture()
	trend := &fakeMetric{
		name: operationsboard.MetricOnTimeByWeek,
		figures: map[string]*boardcatalog.Value{
			"average": value(boardcatalog.ValuePercent, "0.59"),
		},
	}
	f.svc.readers.Metrics[operationsboard.MetricOnTimeByWeek] = trend
	shown := parsed(t, map[string]any{
		"title": "On time", "verdict": "It [slipped][1].",
		"blocks": []any{
			map[string]any{"type": "metric", "metric": operationsboard.MetricOnTimeByWeek},
		},
	})
	f.scopes.grants = readscope.Grants{
		permission.ResourceShipment.String(): {Scope: readscope.ScopeOwn},
	}

	err := f.svc.Check(t.Context(), f.request(shown))
	mistake, ok := errors.AsType[*agentruntimeports.PublishMistake](err)
	require.True(t, ok, "expected a mistake for the model, got %v", err)
	assert.Contains(t, mistake.Reason, "may not read the onTimeByWeek metric")
	filled, err := f.svc.Fill(t.Context(), f.request(shown))
	require.NoError(t, err)
	assert.Zero(t, trend.calls, "the whole organization's figure is never read for them")
	assert.Nil(t, filled.Blocks[0].Metric)

	f.scopes.grants = nil
	require.NoError(t, f.svc.Check(t.Context(), f.request(shown)))
	_, err = f.svc.Fill(t.Context(), f.request(shown))
	require.NoError(t, err)
	assert.Equal(t, 1, trend.calls, "a reader of every shipment reads it")
}
