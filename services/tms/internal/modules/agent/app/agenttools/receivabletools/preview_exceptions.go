package receivabletools

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentexceptionservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

var raisedExceptionFields = []string{
	common.FieldCategory,
	common.FieldSeverity,
	common.FieldSubjectType,
	common.FieldSubjectID,
	common.FieldAttemptSummary,
	common.FieldBlastRadius,
	"resolutionState",
}

var raisedExceptionLabels = map[string]string{
	common.FieldSubjectID:      "About",
	common.FieldAttemptSummary: "What was tried",
	common.FieldBlastRadius:    "Records affected",
}

func raisedExceptionOptions(subjectType agent.SubjectType) []toolpreview.Option {
	opts := []toolpreview.Option{
		toolpreview.Only(raisedExceptionFields...),
		toolpreview.Labels(raisedExceptionLabels),
	}
	if resource, ok := subjectType.Resource(); ok {
		opts = append(opts, toolpreview.WithRefs(map[string]permission.Resource{
			common.FieldSubjectID: resource,
		}))
	}

	return opts
}

func previewRaisedException(
	request *agentports.FlagAgentExceptionRequest,
) (*agent.ToolPreview, error) {
	summary := fmt.Sprintf(
		"Would hand %s to a person as a %s %s exception.",
		request.SubjectType.NounWithArticle(),
		request.Severity,
		request.Category,
	)
	entity, err := agentexceptionservice.PlanException(request)
	if err != nil {
		return common.WarnWouldFail(toolpreview.Build(summary), err), nil
	}

	change, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceAgentException,
		Label:    fmt.Sprintf("%s exception", entity.Category),
	}, entity, raisedExceptionOptions(entity.SubjectType)...)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(summary, change), nil
}

func (t *RaiseExceptionTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	if err := common.GuardPreview(t, &params); err != nil {
		return nil, err
	}

	request, err := t.request(ctx, &params)
	if err != nil {
		if errors.Is(err, ErrMissingRun) || errors.Is(err, ErrSubjectUncheckable) {
			return common.WarnWouldFail(toolpreview.Build("Would raise an exception."), err), nil
		}

		return nil, err
	}

	return previewRaisedException(request)
}
