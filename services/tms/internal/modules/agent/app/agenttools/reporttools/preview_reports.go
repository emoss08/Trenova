package reporttools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/canned"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/shared/stringutils"
)

var reportViewLabels = map[string]string{
	common.FieldSource:        "Reports on",
	"columns":                 "Columns",
	"filters":                 "Filter conditions",
	"charts":                  "Charts",
	"rowLimit":                "Row limit",
	common.FieldDefaultFormat: "Download format",
}

type reportDefinitionView struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Category      string   `json:"category"`
	Tags          []string `json:"tags"`
	Visibility    string   `json:"visibility"`
	Status        string   `json:"status"`
	DefaultFormat string   `json:"defaultFormat"`
	Source        string   `json:"source"`
	Columns       []string `json:"columns"`
	Filters       int      `json:"filters"`
	Charts        []string `json:"charts"`
	RowLimit      int      `json:"rowLimit"`
}

func reportViewOf(entity *report.ReportDefinition) *reportDefinitionView {
	view := &reportDefinitionView{
		Name:          entity.Name,
		Description:   entity.Description,
		Category:      entity.Category,
		Tags:          entity.Tags,
		Visibility:    string(entity.Visibility),
		Status:        string(entity.Status),
		DefaultFormat: string(entity.DefaultFormat),
	}
	definition := entity.Definition
	if definition == nil {
		return view
	}

	view.Source = definition.Entity
	view.RowLimit = definition.Limit
	view.Columns = make([]string, 0, len(definition.Columns))
	for idx := range definition.Columns {
		column := &definition.Columns[idx]
		view.Columns = append(
			view.Columns,
			stringutils.FirstNonEmpty(column.Label, column.Ref.String(), column.ID),
		)
	}
	view.Charts = make([]string, 0, len(definition.Charts))
	for idx := range definition.Charts {
		chart := &definition.Charts[idx]
		view.Charts = append(
			view.Charts,
			stringutils.FirstNonEmpty(chart.Title, string(chart.Type)),
		)
	}
	view.Filters = countFilters(definition.Filters) + countFilters(definition.Having)

	return view
}

func countFilters(group *report.FilterGroup) int {
	if group == nil {
		return 0
	}

	count := len(group.Filters)
	for idx := range group.Groups {
		count += countFilters(&group.Groups[idx])
	}

	return count
}

func reportRecord(entity *report.ReportDefinition) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceReport,
		ID:       entity.ID,
		Label:    entity.Name,
		Version:  common.PreviewVersion(entity.Version),
	}
}

func reportAudience(visibility report.Visibility) string {
	if visibility == report.VisibilityShared {
		return "on every colleague's Reports page"
	}

	return "on your own Reports page"
}

func (t *CreateReportTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	save, err := t.prepare(params)
	if err != nil {
		return nil, err
	}

	planned := reporting.NewDefinition(save)
	change, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceReport,
		Label:    planned.Name,
	}, reportViewOf(planned), toolpreview.Labels(reportViewLabels))
	if err != nil {
		return nil, err
	}

	preview := toolpreview.Build(fmt.Sprintf(
		"Would save the report %q %s.",
		planned.Name,
		reportAudience(planned.Visibility),
	), change)

	return warnUnsaveableReport(ctx, t.reports, preview, save)
}

func warnUnsaveableReport(
	ctx context.Context,
	reports reportDefinitionWriter,
	preview *agent.ToolPreview,
	save *reporting.SaveDefinitionRequest,
) (*agent.ToolPreview, error) {
	if save.Definition == nil {
		return preview, nil
	}

	return common.WarnRefusal(preview, reports.ValidateDefinition(ctx, save))
}

func (t *UpdateReportTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	save, existing, err := t.prepare(ctx, params)
	if err != nil {
		return nil, err
	}

	updated := *existing
	reporting.ApplyDefinitionSave(&updated, save)
	change, err := toolpreview.Changed(
		reportRecord(existing),
		reportViewOf(existing),
		reportViewOf(&updated),
		toolpreview.Labels(reportViewLabels),
	)
	if err != nil {
		return nil, err
	}

	preview := toolpreview.Build(fmt.Sprintf(
		"Would change the saved report %q, which shows %s.",
		existing.Name,
		reportAudience(updated.Visibility),
	), change)

	return warnUnsaveableReport(ctx, t.reports, preview, save)
}

func (t *ForkReportTool) request(
	params *agentports.ToolExecuteParams,
) (*reporting.ForkCannedRequest, *canned.Entry, error) {
	key, err := common.RequireString(params.Params, "reportKey")
	if err != nil {
		return nil, nil, err
	}
	key = strings.TrimSpace(key)

	entry, err := t.reports.GetCanned(key)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"there is no built-in report with the key %q; call report_list for the keys that exist",
			key,
		)
	}

	return &reporting.ForkCannedRequest{
		Request:   reportingRequestFrom(*params),
		CannedKey: key,
		Name:      strings.TrimSpace(common.OptionalString(params.Params, paramReportName)),
	}, entry, nil
}

func (t *ForkReportTool) Preview(
	_ context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	if err := common.GuardPreview(t, &params); err != nil {
		return nil, err
	}

	request, entry, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	planned := reporting.NewCannedFork(entry, request)
	change, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceReport,
		Label:    planned.Name,
	}, reportViewOf(planned), toolpreview.Labels(reportViewLabels))
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would copy the built-in report %q into a report of your own named %q, private to you.",
		entry.Name,
		planned.Name,
	), change), nil
}
