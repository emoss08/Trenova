package masterdatatools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentmemoryservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

const memoryLabelRunes = 80

var rememberedFields = []string{
	common.FieldKind,
	common.FieldSubjectType,
	"subjectLabel",
	common.FieldContent,
	"expiresAt",
	"tainted",
}

var rememberedLabels = map[string]string{"subjectLabel": "About"}

var rememberedTypes = map[string]assistantartifact.DisplayType{
	"expiresAt": assistantartifact.DisplayDate,
}

var retiredMemoryFields = []string{common.FieldStatus, "retiredAt"}

func memoryRecord(memory *agent.Memory) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceAgentMemory,
		ID:       memory.ID,
		Label:    stringutils.Ellipsize(memory.Content, memoryLabelRunes),
		Version:  common.PreviewVersion(memory.Version),
	}
}

func (t *RememberTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, err := t.request(ctx, &params)
	if err != nil {
		return nil, err
	}

	plan, err := t.memories.PreviewRemember(ctx, request, params.Actor)
	if err != nil {
		if common.IsRefusal(err) {
			summary := previewAudience(request.Scope, agent.MemoryStatusActive)
			return common.WarnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}
	summary := previewAudience(plan.Memory.Scope, plan.Memory.Status)
	if plan.Existing != nil {
		return toolpreview.Build(
			"This is already remembered as it stands, so that memory would be refreshed " +
				"rather than recorded again.",
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
	if plan.Replaced == nil {
		return toolpreview.Build(summary, change), nil
	}

	replaced := "\"" + stringutils.Ellipsize(plan.Replaced.Content, memoryLabelRunes) + "\""
	if plan.Held {
		return toolpreview.Build(
			summary+" It would replace "+replaced+", which is shared beyond this person, so "+
				"it would wait for someone allowed to change that memory.",
			change,
		), nil
	}

	retire := agentmemoryservice.StatusChange{
		Status:   agent.MemoryStatusRetired,
		ByUserID: agentmemoryservice.StatusActor(params.Actor),
		At:       timeutils.NowUnix(),
	}
	archived, err := common.PlanArchive(
		memoryRecord(plan.Replaced),
		plan.Replaced,
		func(retired *agent.Memory) error {
			return agentmemoryservice.PlanStatus(retired, retire)
		},
		toolpreview.Only(retiredMemoryFields...),
		toolpreview.Volatile("retiredAt"),
	)
	if err != nil {
		return nil, err
	}

	return archived.PlanPreview(
		summary+" It would replace "+replaced+", which would be retired.",
		change,
	), nil
}

func (t *ForgetMemoryTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	memory, err := t.memories.PreviewSetStatus(ctx, &request, params.Actor)
	if err != nil {
		if common.IsRefusal(err) {
			return common.WarnWouldFail(toolpreview.Build(
				"Would retire the memory so no later run reads it.",
			), err), nil
		}

		return nil, err
	}

	change := agentmemoryservice.StatusChange{
		Status:   request.Status,
		ByUserID: agentmemoryservice.StatusActor(params.Actor),
		At:       timeutils.NowUnix(),
	}
	plan, err := common.PlanArchive(
		memoryRecord(memory),
		memory,
		func(retired *agent.Memory) error {
			return agentmemoryservice.PlanStatus(retired, change)
		},
		toolpreview.Only(retiredMemoryFields...),
		toolpreview.Volatile("retiredAt"),
	)
	if err != nil {
		return nil, err
	}

	return plan.PlanPreview(
		"Would retire the memory \"" + stringutils.Ellipsize(memory.Content, memoryLabelRunes) +
			"\" so no later run reads it. It stays readable in AI Control and can be restored.",
	), nil
}

// previewAudience says whose later runs the memory would reach, and whether
// it would be kept at once or offered: to the person, for one of their own
// they asked to be asked about; to an administrator, for one shared beyond
// them.
func previewAudience(scope agent.MemoryScope, status agent.MemoryStatus) string {
	offered := status == agent.MemoryStatusSuggested
	switch scope {
	case agent.MemoryScopeUser:
		if offered {
			return "Would offer a memory for this person to keep, read in their later conversations once they do."
		}

		return "Would record a memory read in this person's later conversations."
	case agent.MemoryScopeRole:
		if offered {
			return "Would offer a memory for an administrator to keep, read in the later conversations of everyone in this person's role once they do."
		}

		return "Would record a memory read in the later conversations of everyone in this person's role."
	default:
		if offered {
			return "Would offer a memory for an administrator to keep, read by every later run of every agent once they do."
		}

		return "Would record a memory every later run of every agent reads."
	}
}
