package agentquerytoolservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/report"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/reporting"
)

// PreviewedReport is the report a preview_report call ran, read again from
// the call's own arguments: the definition, its parameters as the preview
// passed them, and its name.
type PreviewedReport struct {
	Name       string
	Definition *report.Definition
	Params     map[string]any
}

// ReportOfPreview resolves what a preview_report call previewed, the same
// way the tool did, so a download of the preview is the same report whole
// rather than a near relation of it.
func ReportOfPreview(
	ctx context.Context,
	reports *reporting.Service,
	actor *serviceports.RequestActor,
	arguments map[string]any,
) (*PreviewedReport, error) {
	tool := &previewReportTool{reports: reports}
	params := &serviceports.QueryToolParams{
		OrganizationID: actor.OrganizationID,
		BusinessUnitID: actor.BusinessUnitID,
		Actor:          actor,
		Params:         arguments,
	}

	source, err := tool.source(ctx, params)
	if err != nil {
		return nil, err
	}
	if source.Definition == nil {
		return nil, errors.New("the preview names no report")
	}

	values := normalizeReportParameters(source.Definition, optionalObject(arguments, "parameters"))
	if err = requireReportParameters(source.Name, source.Definition, values); err != nil {
		return nil, err
	}

	return &PreviewedReport{Name: source.Name, Definition: source.Definition, Params: values}, nil
}

// ReportingRequestFor is the reporting request a person's actor makes.
func ReportingRequestFor(actor *serviceports.RequestActor) reporting.Request {
	return reportingRequestFor(&serviceports.QueryToolParams{
		OrganizationID: actor.OrganizationID,
		BusinessUnitID: actor.BusinessUnitID,
		Actor:          actor,
	})
}
