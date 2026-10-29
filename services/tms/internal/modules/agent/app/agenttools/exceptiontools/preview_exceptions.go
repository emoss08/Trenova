package exceptiontools

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentexceptionservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentexceptionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

var raisedExceptionFields = []string{
	changepreview.FieldCategory,
	changepreview.FieldSeverity,
	changepreview.FieldSubjectType,
	changepreview.FieldSubjectID,
	changepreview.FieldAttemptSummary,
	changepreview.FieldBlastRadius,
	"resolutionState",
}

var raisedExceptionLabels = map[string]string{
	changepreview.FieldSubjectID:      "About",
	changepreview.FieldAttemptSummary: "What was tried",
	changepreview.FieldBlastRadius:    "Records affected",
}

func raisedExceptionOptions(subjectType agent.SubjectType) []toolpreview.Option {
	opts := []toolpreview.Option{
		toolpreview.Only(raisedExceptionFields...),
		toolpreview.Labels(raisedExceptionLabels),
	}
	if resource, ok := subjectType.Resource(); ok {
		opts = append(opts, toolpreview.WithRefs(map[string]permission.Resource{
			changepreview.FieldSubjectID: resource,
		}))
	}

	return opts
}

func previewRaisedException(
	request *agentexceptionports.FlagAgentExceptionRequest,
) (*agentpreview.ToolPreview, error) {
	summary := fmt.Sprintf(
		"Would hand %s to a person as a %s %s exception.",
		request.SubjectType.NounWithArticle(),
		request.Severity,
		request.Category,
	)
	entity, err := agentexceptionservice.PlanException(request)
	if err != nil {
		return changepreview.WarnWouldFail(toolpreview.Build(summary), err), nil
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
) (*agentpreview.ToolPreview, error) {
	if err := toolguard.GuardPreview(t, &params); err != nil {
		return nil, err
	}

	request, err := t.request(ctx, &params)
	if err != nil {
		if errors.Is(err, ErrMissingRun) || errors.Is(err, ErrSubjectUncheckable) {
			return changepreview.WarnWouldFail(
				toolpreview.Build("Would raise an exception."),
				err,
			), nil
		}

		return nil, err
	}

	return previewRaisedException(request)
}
