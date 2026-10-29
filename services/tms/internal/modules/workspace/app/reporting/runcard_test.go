package reporting

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/emoss08/keel/analytics/export"
	"github.com/emoss08/keel/analytics/format"
	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/keel/analytics/table"
	"github.com/emoss08/keel/analytics/value"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/canned"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reportingtest"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type cardRuns struct {
	reportports.ReportRunRepository

	run      *report.ReportRun
	previous *report.ReportRun
	asked    *reportports.GetComparableReportRunRequest
}

func (r *cardRuns) GetByID(
	context.Context,
	*reportports.GetReportRunRequest,
) (*report.ReportRun, error) {
	return r.run, nil
}

func (r *cardRuns) LatestComparable(
	_ context.Context,
	req *reportports.GetComparableReportRunRequest,
) (*report.ReportRun, error) {
	r.asked = req
	if r.previous == nil {
		return nil, errortypes.NewNotFoundError("ReportRun not found")
	}
	return r.previous, nil
}

type cardDefinitions struct {
	reportports.ReportDefinitionRepository

	definition *ir.Definition
}

func (d *cardDefinitions) GetByID(
	context.Context,
	*reportports.GetReportDefinitionRequest,
) (*report.ReportDefinition, error) {
	return nil, errortypes.NewNotFoundError("ReportDefinition not found")
}

func (d *cardDefinitions) GetRevision(
	_ context.Context,
	req *reportports.GetReportRevisionRequest,
) (*report.ReportDefinitionRevision, error) {
	return &report.ReportDefinitionRevision{ID: req.RevisionID, Definition: d.definition}, nil
}

func cardDefinition() *ir.Definition {
	return &ir.Definition{
		Version: ir.Version,
		Entity:  "shipment",
		Columns: []ir.Column{
			{ID: "c_customer", Label: "Customer", Dimension: &ir.Dimension{Field: "customer.name"}},
			{
				ID:      "c_revenue",
				Label:   "Revenue",
				Measure: &ir.Measure{Field: "totalChargeAmount", Agg: ir.AggSum},
			},
		},
		Filters: &ir.Filter{And: []ir.Filter{
			{Field: "createdAt", Op: ir.OpLastNDays, Param: "windowDays"},
			{Field: "status", Op: ir.OpIn, Param: "statuses"},
		}},
		Sort: []ir.Sort{{Column: "c_revenue", Desc: true}},
		Parameters: []ir.Parameter{
			{Name: "windowDays", Label: "Window (days)", Type: "int", Default: 90},
			{
				Name:    "statuses",
				Label:   "Statuses",
				Type:    "enum",
				Multi:   true,
				Default: []any{"New"},
				Allowed: []any{"New", "Completed"},
			},
		},
		Charts: []ir.Chart{{
			ID:     "revenue",
			Type:   ir.ChartBar,
			X:      "c_customer",
			Series: []string{"c_revenue"},
		}},
	}
}

func cardSample(t *testing.T, total string) []byte {
	t.Helper()
	return sampleOf(t, total, false)
}

func sampleOf(t *testing.T, total string, truncated bool) []byte {
	t.Helper()
	columns := []table.Column{
		{
			ID:      "c_customer",
			Label:   "Customer",
			Type:    value.String,
			Display: format.Resolve(value.String, format.Hint{}, nil),
		},
		{
			ID:      "c_revenue",
			Label:   "Revenue",
			Type:    value.Decimal,
			Display: format.Resolve(value.Decimal, format.Hint{Kind: format.Money}, nil),
		},
	}
	builder := table.NewBuilder(columns)
	require.NoError(t, builder.Append("ACME", total))
	require.NoError(t, builder.AppendLevel(table.Total, nil, total))
	result := builder.Table()
	result.Truncated = truncated
	var buf bytes.Buffer
	_, err := export.JSON(t.Context(), &buf, result.Reader(), export.Options{})
	require.NoError(t, err)

	return buf.Bytes()
}

func cardService(t *testing.T, runs *cardRuns) *Service {
	t.Helper()
	svc, _ := engineService(t, engineOptions{})
	svc.runRepo = runs
	svc.defRepo = &cardDefinitions{definition: cardDefinition()}
	svc.canned = canned.NewRegistry([]*canned.Entry{{
		Key:         "revenue-by-customer",
		Version:     "1.0.0",
		Name:        "Revenue by customer",
		Description: "Charged revenue per customer.",
		Category:    "Billing",
		Definition:  cardDefinition(),
	}})
	return svc
}

func cardRun(t *testing.T) *report.ReportRun {
	t.Helper()
	return &report.ReportRun{
		ID:            pulid.MustNew("rrun_"),
		RequestedByID: pulid.MustNew("usr_"),
		CannedKey:     "revenue-by-customer",
		Status:        report.RunStatusSucceeded,
		Format:        report.FormatXLSX,
		Params:        map[string]any{"windowDays": 30.0},
		QueuedAt:      time.Date(2026, 10, 27, 14, 59, 0, 0, time.UTC).Unix(),
		StartedAt:     time.Date(2026, 10, 27, 15, 0, 0, 0, time.UTC).Unix(),
		CompletedAt:   time.Date(2026, 10, 27, 15, 1, 0, 0, time.UTC).Unix(),
		Sample:        cardSample(t, "3541.5"),
	}
}

func TestRunCard_ReadsTheReportItsParametersAndTheRunItIsComparedWith(t *testing.T) {
	t.Parallel()

	run := cardRun(t)
	previous := &report.ReportRun{
		ID:          pulid.MustNew("rrun_"),
		CompletedAt: run.CompletedAt - 7*24*3600,
		RowCount:    208,
		Sample:      cardSample(t, "3100"),
	}
	runs := &cardRuns{run: run, previous: previous}

	card, err := cardService(t, runs).GetRunCard(t.Context(), rowsRequest())
	require.NoError(t, err)

	assert.Equal(t, RunCardReport{
		Name:        "Revenue by customer",
		Description: "Charged revenue per customer.",
		Category:    "Billing",
		CannedKey:   "revenue-by-customer",
	}, card.Report)
	assert.Equal(t, "America/Chicago", card.Timezone)

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	assert.Equal(t, []RunCardParameter{
		{
			Name:        "windowDays",
			Label:       "Window (days)",
			Type:        "int",
			Value:       30.0,
			WindowStart: time.Date(2026, 9, 27, 0, 0, 0, 0, chicago).Unix(),
			WindowEnd:   time.Date(2026, 10, 28, 0, 0, 0, 0, chicago).Unix(),
		},
		{
			Name:      "statuses",
			Label:     "Statuses",
			Type:      "enum",
			Value:     []any{"New"},
			IsDefault: true,
		},
	}, card.Parameters)

	assert.Equal(t, &ir.Chart{
		ID:     "revenue",
		Type:   ir.ChartBar,
		X:      "c_customer",
		Series: []string{"c_revenue"},
	}, card.Chart)
	assert.Equal(t, []ir.Sort{{Column: "c_revenue", Desc: true}}, card.Sort)

	assert.Empty(t, card.Withheld)
	require.NotNil(t, card.Sample)
	assert.Equal(t, "America/Chicago", card.Sample.Timezone)
	require.Len(t, card.Sample.Columns, 2)
	assert.Equal(t, "c_revenue", card.Sample.Columns[1].ID)
	assert.Equal(t, [][]any{{"ACME", "3541.5"}}, card.Sample.Rows)
	assert.Equal(t, []table.Level{table.Detail}, card.Sample.Levels)
	assert.Equal(t, []any{nil, "3541.5"}, card.Sample.Totals)
	assert.False(t, card.Sample.Truncated)

	assert.Equal(t, &RunCardPrevious{
		RunID:       previous.ID,
		CompletedAt: previous.CompletedAt,
		RowCount:    208,
		ColumnIDs:   []string{"c_customer", "c_revenue"},
		Totals:      []any{nil, "3100"},
	}, card.Previous)

	require.NotNil(t, runs.asked)
	assert.Equal(t, run.RequestedByID, runs.asked.RequestedBy)
	assert.True(t, runs.asked.DefinitionID.IsNil())
	assert.Equal(t, "revenue-by-customer", runs.asked.CannedKey)
	assert.Equal(t, run.Params, runs.asked.Params)
	assert.Equal(t, run.ID, runs.asked.ExcludeRunID)
	assert.Equal(t, run.CompletedAt, runs.asked.CompletedBefore)
}

func TestRunCard_ACappedSampleSaysSo(t *testing.T) {
	t.Parallel()

	run := cardRun(t)
	run.Sample = sampleOf(t, "3541.5", true)

	card, err := cardService(t, &cardRuns{run: run}).GetRunCard(t.Context(), rowsRequest())
	require.NoError(t, err)

	require.NotNil(t, card.Sample)
	assert.True(t, card.Sample.Truncated)
	assert.Nil(t, card.Previous, "no earlier run is no comparison")
}

func TestRunCard_WithholdsTheResultWhenTheRequesterLostAccess(t *testing.T) {
	t.Parallel()

	runs := &cardRuns{run: cardRun(t), previous: &report.ReportRun{Sample: cardSample(t, "1")}}
	svc := cardService(t, runs)
	withdrawShipments(t, svc)

	card, err := svc.GetRunCard(t.Context(), rowsRequest())
	require.NoError(t, err)

	assert.Equal(t, RunCardWithheldAccess, card.Withheld)
	assert.Nil(t, card.Sample)
	assert.Nil(t, card.Previous)
	assert.Nil(t, runs.asked, "nothing is compared with a result that is not shown")
	assert.Equal(t, report.RunStatusSucceeded, card.Run.Status)
	assert.Len(t, card.Parameters, 2, "what was asked is still shown")
}

func TestRunCard_WithholdsTheResultWhenTheReportNoLongerCompiles(t *testing.T) {
	t.Parallel()

	run := cardRun(t)
	run.CannedKey = ""
	run.DefinitionID = pulid.MustNew("rdef_")
	run.RevisionID = pulid.MustNew("rrev_")
	svc := cardService(t, &cardRuns{run: run})
	changed := cardDefinition()
	changed.Columns = append(changed.Columns, ir.Column{
		ID:        "c_gone",
		Dimension: &ir.Dimension{Field: "noSuchField"},
	})
	svc.defRepo = &cardDefinitions{definition: changed}

	card, err := svc.GetRunCard(t.Context(), rowsRequest())
	require.NoError(t, err)

	assert.Equal(t, RunCardWithheldChanged, card.Withheld)
	assert.Nil(t, card.Sample)
}

func TestRunCard_ARunStillGoingHasNoResultToCheckOrCompare(t *testing.T) {
	t.Parallel()

	run := cardRun(t)
	run.Status = report.RunStatusRunning
	run.CompletedAt = 0
	run.Sample = nil
	runs := &cardRuns{run: run}
	svc := cardService(t, runs)
	engine, ok := svc.engine.(*reportingtest.Engine)
	require.True(t, ok)

	card, err := svc.GetRunCard(t.Context(), rowsRequest())
	require.NoError(t, err)

	assert.Nil(t, card.Sample)
	assert.Nil(t, card.Previous)
	assert.Empty(t, engine.Recorded("compile"), "no access check runs without a result")
	assert.Nil(t, runs.asked)
	assert.Len(t, card.Parameters, 2)
}

func TestRunCard_ASavedReportWhoseRecordIsGoneStillShowsTheRun(t *testing.T) {
	t.Parallel()

	run := cardRun(t)
	run.CannedKey = ""
	run.DefinitionID = pulid.MustNew("rdef_")
	run.RevisionID = pulid.MustNew("rrev_")
	runs := &cardRuns{run: run}

	card, err := cardService(t, runs).GetRunCard(t.Context(), rowsRequest())
	require.NoError(t, err)

	assert.Equal(t, RunCardReport{DefinitionID: run.DefinitionID}, card.Report)
	assert.Len(t, card.Parameters, 2)
	require.NotNil(t, card.Sample)
	assert.Equal(t, [][]any{{"ACME", "3541.5"}}, card.Sample.Rows)
	require.NotNil(t, runs.asked)
	assert.Equal(t, run.DefinitionID, runs.asked.DefinitionID)
	assert.Empty(t, runs.asked.CannedKey)
}

func TestRunCard_AnUnreadableSampleStillShowsTheRun(t *testing.T) {
	t.Parallel()

	run := cardRun(t)
	run.Sample = []byte("not an envelope")

	card, err := cardService(t, &cardRuns{run: run}).GetRunCard(t.Context(), rowsRequest())
	require.NoError(t, err)

	assert.Nil(t, card.Sample)
	assert.Empty(t, card.Withheld)
	assert.Equal(t, "Revenue by customer", card.Report.Name)
}
