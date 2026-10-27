package reporting

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/canned"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/pkg/reportrows"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type cardRuns struct {
	reportports.ReportRunRepository

	run      *report.ReportRun
	previous *report.ReportRun
	asked    *reportports.GetComparableReportRunRequest
}

func (r *cardRuns) GetByID(
	_ context.Context,
	_ *reportports.GetReportRunRequest,
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

	definition *report.Definition
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

func cardDefinition() *report.Definition {
	chart := report.ChartSpec{ID: "revenue", Type: report.ChartBar, XColumnID: "customer"}
	return &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "invoice",
		Columns: []report.ColumnSpec{
			{ID: "customer", Ref: report.FieldRef{Field: "name"}, Kind: report.ColumnKindDimension},
		},
		Filters: &report.FilterGroup{
			Op: report.BoolOpAnd,
			Filters: []report.FieldFilter{
				{
					Ref:      report.FieldRef{Field: "invoiceDate"},
					Operator: dbtype.OpLastNDays,
					Param:    "windowDays",
				},
				{Ref: report.FieldRef{Field: "status"}, Operator: dbtype.OpIn, Param: "statuses"},
			},
		},
		Sort: []report.SortSpec{{ColumnID: "revenue", Direction: dbtype.SortDirectionDesc}},
		Parameters: []report.ParameterDef{
			{Name: "windowDays", Label: "Window (days)", Type: reportcatalog.FieldInt, Default: 90.0},
			{
				Name:          "statuses",
				Label:         "Statuses",
				Type:          reportcatalog.FieldEnum,
				Multi:         true,
				Default:       []any{"Posted"},
				AllowedValues: []string{"Posted", "Draft"},
			},
		},
		Charts: []report.ChartSpec{chart},
	}
}

func cardSample(total string) *reportrows.Sample {
	return &reportrows.Sample{
		Schema: []reportrows.Column{
			{ID: "customer", Label: "Customer", Type: reportcatalog.FieldString},
			{ID: "revenue", Label: "Revenue", Type: reportcatalog.FieldDecimal},
		},
		Rows:   [][]any{{"ACME", total}},
		Totals: []any{nil, total},
	}
}

func cardService(runs *cardRuns, compile *recordingCompiler) *Service {
	return &Service{
		l:        zap.NewNop(),
		runRepo:  runs,
		defRepo:  &cardDefinitions{definition: cardDefinition()},
		orgRepo:  previewOrganizations{},
		compiler: compile,
		canned: canned.NewRegistry([]*canned.Entry{{
			Key:         "revenue-by-customer",
			Version:     "1.0.0",
			Name:        "Revenue by customer",
			Description: "Invoiced revenue per customer.",
			Category:    "Billing",
			Definition:  cardDefinition(),
		}}),
	}
}

func cardRun() *report.ReportRun {
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
		Sample:        cardSample("3541.5"),
	}
}

func TestRunCard_ReadsTheReportItsParametersAndTheRunItIsComparedWith(t *testing.T) {
	t.Parallel()

	run := cardRun()
	previous := &report.ReportRun{
		ID:          pulid.MustNew("rrun_"),
		CompletedAt: run.CompletedAt - 7*24*3600,
		Sample:      cardSample("3100"),
	}
	runs := &cardRuns{run: run, previous: previous}

	card, err := cardService(runs, &recordingCompiler{}).GetRunCard(t.Context(), rowsRequest())
	require.NoError(t, err)

	assert.Equal(t, "Revenue by customer", card.Report.Name)
	assert.Equal(t, "Invoiced revenue per customer.", card.Report.Description)
	assert.Equal(t, "Billing", card.Report.Category)
	assert.Equal(t, "revenue-by-customer", card.Report.CannedKey)
	assert.Equal(t, "America/Chicago", card.Timezone)

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	require.Len(t, card.Parameters, 2)
	assert.Equal(t, RunCardParameter{
		Name:        "windowDays",
		Label:       "Window (days)",
		Type:        reportcatalog.FieldInt,
		Value:       30.0,
		WindowStart: time.Date(2026, 9, 27, 0, 0, 0, 0, chicago).Unix(),
		WindowEnd:   time.Date(2026, 10, 28, 0, 0, 0, 0, chicago).Unix(),
	}, card.Parameters[0])
	assert.Equal(t, RunCardParameter{
		Name:      "statuses",
		Label:     "Statuses",
		Type:      reportcatalog.FieldEnum,
		Value:     []any{"Posted"},
		IsDefault: true,
	}, card.Parameters[1])

	require.NotNil(t, card.Chart)
	assert.Equal(t, report.ChartBar, card.Chart.Type)
	assert.Equal(t, []report.SortSpec{
		{ColumnID: "revenue", Direction: dbtype.SortDirectionDesc},
	}, card.Sort)

	assert.Equal(t, run.Sample, card.Sample)
	assert.Empty(t, card.Withheld)
	require.NotNil(t, card.Previous)
	assert.Equal(t, previous.ID, card.Previous.RunID)
	assert.Equal(t, previous.CompletedAt, card.Previous.CompletedAt)
	assert.Equal(t, []any{nil, "3100"}, card.Previous.Totals)

	require.NotNil(t, runs.asked)
	assert.Equal(t, run.RequestedByID, runs.asked.RequestedBy)
	assert.Equal(t, "revenue-by-customer", runs.asked.CannedKey)
	assert.Equal(t, run.Params, runs.asked.Params)
	assert.Equal(t, run.ID, runs.asked.ExcludeRunID)
	assert.Equal(t, run.CompletedAt, runs.asked.CompletedBefore)
}

func TestRunCard_WithholdsTheResultWhenTheRequesterLostAccess(t *testing.T) {
	t.Parallel()

	runs := &cardRuns{run: cardRun(), previous: &report.ReportRun{Sample: cardSample("1")}}
	compile := &recordingCompiler{err: errAccessWithdrawn}

	card, err := cardService(runs, compile).GetRunCard(t.Context(), rowsRequest())
	require.NoError(t, err)

	assert.Equal(t, RunCardWithheldAccess, card.Withheld)
	assert.Nil(t, card.Sample)
	assert.Nil(t, card.Previous)
	assert.Nil(t, runs.asked)
	assert.Equal(t, report.RunStatusSucceeded, card.Run.Status)
}

func TestRunCard_WithholdsTheResultWhenTheReportNoLongerCompiles(t *testing.T) {
	t.Parallel()

	compile := &recordingCompiler{err: errortypes.NewBusinessError("This report has changed")}

	card, err := cardService(&cardRuns{run: cardRun()}, compile).
		GetRunCard(t.Context(), rowsRequest())
	require.NoError(t, err)

	assert.Equal(t, RunCardWithheldChanged, card.Withheld)
	assert.Nil(t, card.Sample)
}

func TestRunCard_ARunStillGoingHasNoResultToCheckOrCompare(t *testing.T) {
	t.Parallel()

	run := cardRun()
	run.Status = report.RunStatusRunning
	run.CompletedAt = 0
	run.Sample = nil
	runs := &cardRuns{run: run}
	compile := &recordingCompiler{}

	card, err := cardService(runs, compile).GetRunCard(t.Context(), rowsRequest())
	require.NoError(t, err)

	assert.Nil(t, card.Sample)
	assert.Nil(t, card.Previous)
	assert.Empty(t, compile.exports)
	assert.Nil(t, runs.asked)
	require.Len(t, card.Parameters, 2)
}

func TestRunCard_ASavedReportWhoseRecordIsGoneStillShowsTheRun(t *testing.T) {
	t.Parallel()

	run := cardRun()
	run.CannedKey = ""
	run.DefinitionID = pulid.MustNew("rdef_")
	run.RevisionID = pulid.MustNew("rrev_")

	card, err := cardService(&cardRuns{run: run}, &recordingCompiler{}).
		GetRunCard(t.Context(), rowsRequest())
	require.NoError(t, err)

	assert.Empty(t, card.Report.Name)
	assert.Equal(t, run.DefinitionID, card.Report.DefinitionID)
	require.Len(t, card.Parameters, 2)
	assert.Equal(t, run.Sample, card.Sample)
	assert.Nil(t, card.Previous)
}
