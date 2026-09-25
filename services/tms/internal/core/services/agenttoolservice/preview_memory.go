package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentmemoryservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

var (
	_ serviceports.ToolPreviewer = (*rememberTool)(nil)
	_ serviceports.ToolPreviewer = (*forgetMemoryTool)(nil)
)

const memoryLabelRunes = 80

var rememberedFields = []string{
	fieldKind,
	fieldSubjectType,
	"subjectLabel",
	fieldContent,
	"expiresAt",
	"tainted",
}

var rememberedLabels = map[string]string{"subjectLabel": "About"}

var rememberedTypes = map[string]assistantartifact.DisplayType{
	"expiresAt": assistantartifact.DisplayDate,
}

var retiredMemoryFields = []string{fieldStatus, "retiredAt"}

func memoryRecord(memory *agent.Memory) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceAgentMemory,
		ID:       memory.ID,
		Label:    stringutils.Ellipsize(memory.Content, memoryLabelRunes),
		Version:  previewVersion(memory.Version),
	}
}

func (t *rememberTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	summary := "Would record a memory every later run of every agent reads."
	plan, err := t.memories.PreviewRemember(ctx, request, params.Actor)
	if err != nil {
		if isRefusal(err) {
			return warnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}
	if plan.Existing != nil {
		return toolpreview.Build(
			"This is already remembered as it stands, so nothing would be recorded again.",
		), nil
	}

	change, err := toolpreview.Create(
		memoryRecord(plan.Memory),
		plan.Memory,
		toolpreview.Only(rememberedFields...),
		toolpreview.Labels(rememberedLabels),
		toolpreview.Types(rememberedTypes),
	)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(summary, change), nil
}

func (t *forgetMemoryTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	memory, err := t.memories.GetByID(ctx, repositories.GetAgentMemoryByIDRequest{
		ID:         request.ID,
		TenantInfo: request.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	change := agentmemoryservice.StatusChange{
		Status:   request.Status,
		ByUserID: agentmemoryservice.StatusActor(params.Actor),
		At:       timeutils.NowUnix(),
	}
	plan, err := planArchive(memoryRecord(memory), memory, func(retired *agent.Memory) error {
		return agentmemoryservice.PlanStatus(retired, change)
	}, toolpreview.Only(retiredMemoryFields...), toolpreview.Volatile("retiredAt"))
	if err != nil {
		return nil, err
	}

	return plan.preview(
		"Would retire the memory \"" + stringutils.Ellipsize(memory.Content, memoryLabelRunes) +
			"\" so no later run reads it. It stays readable in AI Control and can be restored.",
	), nil
}
