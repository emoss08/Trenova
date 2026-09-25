package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/internal/core/services/reporting/canned"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type savingReports struct {
	fakeReportWriter

	saved *report.ReportDefinition
	guard writeGuard
}

func (f *savingReports) CreateDefinition(
	_ context.Context,
	req *reporting.SaveDefinitionRequest,
) (*report.ReportDefinition, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.saved = reporting.NewDefinition(req)
	f.saved.ID = pulid.MustNew("rdef_")

	return f.saved, nil
}

func (f *savingReports) UpdateDefinition(
	_ context.Context,
	req *reporting.SaveDefinitionRequest,
) (*report.ReportDefinition, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	for _, definition := range f.definitions {
		if definition.ID == req.DefinitionID {
			updated := *definition
			reporting.ApplyDefinitionSave(&updated, req)
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
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	entry, err := f.GetCanned(req.CannedKey)
	if err != nil {
		return nil, err
	}
	f.saved = reporting.NewCannedFork(entry, req)
	f.saved.ID = pulid.MustNew("rdef_")

	return f.saved, nil
}

func TestCreateReport_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	reports := &savingReports{}
	tool := newCreateReportTool(reports).(*createReportTool)
	params := executeParams(map[string]any{
		"name":       "Revenue by customer",
		"visibility": "shared",
		"definition": definitionArgument(),
	})

	preview := previewWithoutWrites(t, &reports.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, change.Operation)
	assert.Equal(t, permission.ResourceReport, change.Resource)
	assert.Equal(t, "Revenue by customer", change.Label)
	assert.Equal(t, "shipment", fieldByPath(t, change, "source").After)
	assert.Contains(t, fieldByPath(t, change, "columns").After, "Revenue")
	assert.Equal(t, "Filter conditions", fieldByPath(t, change, "filters").Label)
	assert.Contains(t, preview.Summary, "every colleague")

	require.NoError(t, tool.Execute(t.Context(), params))
	requireCreateParity(t, change, reportViewOf(reports.saved),
		toolpreview.Labels(reportViewLabels))
}

func TestCreateReport_PreviewWarnsWhenTheDefinitionWouldNotCompile(t *testing.T) {
	t.Parallel()

	reports := &savingReports{fakeReportWriter: fakeReportWriter{
		invalid: errortypes.NewValidationError("definition", errortypes.ErrInvalid, "no field"),
	}}
	tool := newCreateReportTool(reports).(*createReportTool)

	preview := previewWithoutWrites(t, &reports.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), executeParams(map[string]any{
			"name":       "Broken",
			"definition": definitionArgument(),
		}))
	})

	require.Len(t, preview.Changes, 1)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}

func TestUpdateReport_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	params := executeParams(map[string]any{})
	existing := ownedReport(params.Actor.UserID)
	reports := &savingReports{fakeReportWriter: fakeReportWriter{
		definitions: []*report.ReportDefinition{existing},
	}}
	tool := newUpdateReportTool(reports).(*updateReportTool)
	params.Params = map[string]any{
		"definitionId": existing.ID.String(),
		"name":         "Revenue by customer, last 90 days",
		"visibility":   "shared",
	}

	preview := previewWithoutWrites(t, &reports.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := previewChange(t, preview, 0)
	assert.Equal(t, existing.ID, change.EntityID)
	require.NotNil(t, change.Version)
	assert.Equal(t, int64(4), *change.Version)
	assert.Equal(t, "Revenue by customer", fieldByPath(t, change, "name").Before)
	assert.Equal(t, "Revenue by customer, last 90 days", fieldByPath(t, change, "name").After)
	assert.Equal(t, "shared", fieldByPath(t, change, "visibility").After)

	require.NoError(t, tool.Execute(t.Context(), params))
	requireUpdateParity(t, change, reportViewOf(existing), reportViewOf(reports.saved),
		toolpreview.Labels(reportViewLabels))
}

func TestForkReport_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	entry := &canned.Entry{
		Key:           "revenue_by_customer",
		Version:       "3",
		Name:          "Revenue by customer",
		Description:   "Completed revenue per customer.",
		DefaultFormat: report.FormatCSV,
		Definition: &report.Definition{
			Entity: "shipment",
			Columns: []report.ColumnSpec{{
				ID:    "c1",
				Ref:   report.FieldRef{Field: "totalChargeAmount"},
				Kind:  report.ColumnKindMeasure,
				Label: "Revenue",
			}},
		},
	}
	reports := &savingReports{fakeReportWriter: fakeReportWriter{entries: []*canned.Entry{entry}}}
	tool := newForkReportTool(reports).(*forkReportTool)
	params := executeParams(map[string]any{"reportKey": "revenue_by_customer", "name": "My revenue"})

	preview := previewWithoutWrites(t, &reports.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := previewChange(t, preview, 0)
	assert.Equal(t, "My revenue", fieldByPath(t, change, "name").After)
	assert.Equal(t, "private", fieldByPath(t, change, "visibility").After)
	assert.Contains(t, preview.Summary, "Revenue by customer")

	require.NoError(t, tool.Execute(t.Context(), params))
	requireCreateParity(t, change, reportViewOf(reports.saved),
		toolpreview.Labels(reportViewLabels))
}
