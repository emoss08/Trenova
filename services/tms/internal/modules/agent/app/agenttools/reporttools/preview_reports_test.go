package reporttools

import (
	"context"
	"testing"

	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/canned"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type savingReports struct {
	fakeReportWriter

	saved *report.ReportDefinition
	guard agenttoolstest.WriteGuard
}

func (f *savingReports) CreateDefinition(
	_ context.Context,
	req *reporting.SaveDefinitionRequest,
) (*report.ReportDefinition, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.saved = f.NewDefinition(req)
	f.saved.ID = pulid.MustNew("rdef_")

	return f.saved, nil
}

func (f *savingReports) UpdateDefinition(
	_ context.Context,
	req *reporting.SaveDefinitionRequest,
) (*report.ReportDefinition, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	for _, definition := range f.definitions {
		if definition.ID == req.DefinitionID {
			updated := *definition
			f.ApplyDefinitionSave(&updated, req)
			f.saved = &updated

			return f.saved, nil
		}
	}

	return nil, errortypes.NewNotFoundError("ReportDefinition not found")
}

func (f *savingReports) ForkCanned(
	_ context.Context,
	req *reporting.ForkCannedRequest,
) (*report.ReportDefinition, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	entry, err := f.GetCanned(req.CannedKey)
	if err != nil {
		return nil, err
	}
	f.saved = f.NewCannedFork(entry, req)
	f.saved.ID = pulid.MustNew("rdef_")

	return f.saved, nil
}

func TestCreateReport_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	reports := &savingReports{}
	tool := NewCreateReportTool(reports).(*CreateReportTool)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"name":       "Revenue by customer",
		"visibility": "shared",
		"definition": definitionArgument(),
	})

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&reports.guard,
		func() (*agent.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, change.Operation)
	assert.Equal(t, permission.ResourceReport, change.Resource)
	assert.Equal(t, "Revenue by customer", change.Label)
	assert.Equal(t, "shipment", agenttoolstest.FieldByPath(t, change, "source").After)
	assert.Contains(t, agenttoolstest.FieldByPath(t, change, "columns").After, "Revenue")
	assert.Equal(t, "Filter conditions", agenttoolstest.FieldByPath(t, change, "filters").Label)
	assert.Contains(t, preview.Summary, "every colleague")

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireCreateParity(t, change, reportViewOf(reports.saved),
		toolpreview.Labels(reportViewLabels))
}

func TestCreateReport_PreviewWarnsWhenTheDefinitionWouldNotCompile(t *testing.T) {
	t.Parallel()

	reports := &savingReports{fakeReportWriter: fakeReportWriter{
		invalid: errortypes.NewValidationError("definition", errortypes.ErrInvalid, "no field"),
	}}
	tool := NewCreateReportTool(reports).(*CreateReportTool)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&reports.guard,
		func() (*agent.ToolPreview, error) {
			return tool.Preview(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
				"name":       "Broken",
				"definition": definitionArgument(),
			}))
		},
	)

	require.Len(t, preview.Changes, 1)
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)
}

func TestUpdateReport_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	params := agenttoolstest.ExecuteParams(map[string]any{})
	existing := ownedReport(params.Actor.UserID)
	reports := &savingReports{fakeReportWriter: fakeReportWriter{
		definitions: []*report.ReportDefinition{existing},
	}}
	tool := NewUpdateReportTool(reports).(*UpdateReportTool)
	params.Params = map[string]any{
		"definitionId": existing.ID.String(),
		"name":         "Revenue by customer, last 90 days",
		"visibility":   "shared",
	}

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&reports.guard,
		func() (*agent.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, existing.ID, change.EntityID)
	require.NotNil(t, change.Version)
	assert.Equal(t, int64(4), *change.Version)
	assert.Equal(t, "Revenue by customer", agenttoolstest.FieldByPath(t, change, "name").Before)
	assert.Equal(
		t,
		"Revenue by customer, last 90 days",
		agenttoolstest.FieldByPath(t, change, "name").After,
	)
	assert.Equal(t, "shared", agenttoolstest.FieldByPath(t, change, "visibility").After)

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireUpdateParity(
		t,
		change,
		reportViewOf(existing),
		reportViewOf(reports.saved),
		toolpreview.Labels(reportViewLabels),
	)
}

func TestForkReport_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	entry := &canned.Entry{
		Key:           "revenue_by_customer",
		Version:       "3",
		Name:          "Revenue by customer",
		Description:   "Completed revenue per customer.",
		DefaultFormat: report.FormatCSV,
		Definition: &ir.Definition{
			Version: ir.Version,
			Entity:  "shipment",
			Columns: []ir.Column{{
				ID:      "c1",
				Label:   "Revenue",
				Measure: &ir.Measure{Field: "totalChargeAmount", Agg: ir.AggSum},
			}},
		},
	}
	reports := &savingReports{fakeReportWriter: fakeReportWriter{entries: []*canned.Entry{entry}}}
	tool := NewForkReportTool(reports).(*ForkReportTool)
	params := agenttoolstest.ExecuteParams(
		map[string]any{"reportKey": "revenue_by_customer", "name": "My revenue"},
	)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		&reports.guard,
		func() (*agent.ToolPreview, error) {
			return tool.Preview(t.Context(), params)
		},
	)

	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, "My revenue", agenttoolstest.FieldByPath(t, change, "name").After)
	assert.Equal(t, "private", agenttoolstest.FieldByPath(t, change, "visibility").After)
	assert.Contains(t, preview.Summary, "Revenue by customer")

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireCreateParity(t, change, reportViewOf(reports.saved),
		toolpreview.Labels(reportViewLabels))
}
