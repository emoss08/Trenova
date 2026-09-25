package agenttoolservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentexceptionservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
)

var (
	_ serviceports.ToolPreviewer = (*raiseExceptionTool)(nil)
	_ serviceports.ToolPreviewer = (*flagManualReviewTool)(nil)
)

var raisedExceptionFields = []string{
	"category",
	"severity",
	"subjectType",
	"subjectId",
	"attemptSummary",
	"blastRadius",
	"resolutionState",
}

var raisedExceptionLabels = map[string]string{
	"subjectId":      "About",
	"attemptSummary": "What was tried",
	"blastRadius":    "Records affected",
}

func raisedExceptionOptions(subjectType agent.SubjectType) []toolpreview.Option {
	opts := []toolpreview.Option{
		toolpreview.Only(raisedExceptionFields...),
		toolpreview.Labels(raisedExceptionLabels),
	}
	if resource, ok := subjectType.Resource(); ok {
		opts = append(opts, toolpreview.WithRefs(map[string]permission.Resource{
			"subjectId": resource,
		}))
	}

	return opts
}

func previewRaisedException(
	request *serviceports.FlagAgentExceptionRequest,
) (*agent.ToolPreview, error) {
	summary := fmt.Sprintf(
		"Would hand %s to a person as a %s %s exception.",
		request.SubjectType.NounWithArticle(),
		request.Severity,
		request.Category,
	)
	entity, err := agentexceptionservice.PlanException(request)
	if err != nil {
		return warnWouldFail(toolpreview.Build(summary), err), nil
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

func (t *raiseExceptionTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}

	request, err := t.request(ctx, params)
	if err != nil {
		if errors.Is(err, ErrMissingRun) || errors.Is(err, ErrSubjectUncheckable) {
			return warnWouldFail(toolpreview.Build("Would raise an exception."), err), nil
		}

		return nil, err
	}

	return previewRaisedException(request)
}

func (t *flagManualReviewTool) Preview(
	_ context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}

	request, err := t.request(params)
	if err != nil {
		return nil, err
	}

	return previewRaisedException(request)
}
