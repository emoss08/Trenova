package masterdatatools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tableconfiguration"
)

var savedViewFields = []string{common.FieldName, "resource", common.FieldVisibility}

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

func (t *SaveTableViewTool) Preview(
	_ context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	if err := common.GuardPreview(t, &params); err != nil {
		return nil, err
	}

	draft, err := t.Draft(&params)
	if err != nil {
		return common.WarnWouldFail(toolpreview.Build("Would save a table view."), err), nil
	}

	change, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceTableConfiguration,
		Label:    draft.name,
	}, draft.configuration(&params, nil), savedViewOptions()...)
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

	return common.WarnRefusal(preview, t.validateArgs(params.Params))
}
