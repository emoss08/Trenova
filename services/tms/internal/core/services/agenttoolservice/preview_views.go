package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tableconfiguration"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
)

var _ serviceports.ToolPreviewer = (*saveTableViewTool)(nil)

var savedViewFields = []string{"name", "resource", "visibility"}

var savedViewLabels = map[string]string{"resource": "Table"}

func savedViewOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(savedViewFields...),
		toolpreview.Labels(savedViewLabels),
	}
}

func savedViewAudience(visibility tableconfiguration.Visibility) string {
	if visibility == tableconfiguration.VisibilityPublic {
		return "in every colleague's view picker"
	}

	return "in your own view picker"
}

func (t *saveTableViewTool) Preview(
	_ context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}

	draft, err := t.draft(params)
	if err != nil {
		return warnWouldFail(toolpreview.Build("Would save a table view."), err), nil
	}

	change, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceTableConfiguration,
		Label:    draft.name,
	}, draft.configuration(params, nil), savedViewOptions()...)
	if err != nil {
		return nil, err
	}

	preview := toolpreview.Build(fmt.Sprintf(
		"Would save the view %q of %s %s. Its filters are worked out from %q when it is "+
			"saved, and it is refused if nothing in that names a field on the table.",
		draft.name,
		draft.resource.Entity,
		savedViewAudience(draft.visibility),
		draft.prompt,
	), change)
	preview.Partial = true

	return warnRefusal(preview, t.validateArgs(params.Params))
}
