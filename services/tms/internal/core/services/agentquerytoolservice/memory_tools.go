package agentquerytoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/shared/pulid"
)

const maxRecallRows = 50

// memoryRow is a memory in the words the model needs: what it says, what it
// is about, and which kind it is so the model knows whether to follow it.
type memoryRow struct {
	ID         string       `json:"id"`
	Kind       string       `json:"kind"`
	Content    string       `json:"content"`
	About      string       `json:"about,omitempty"`
	Tool       string       `json:"tool,omitempty"`
	RecordedBy string       `json:"recordedBy"`
	RecordedOn optionalDate `json:"recordedOn"`
	ExpiresOn  optionalDate `json:"expiresOn"`
}

type recallMemoryTool struct {
	memories serviceports.AgentMemoryService
}

func newRecallMemoryTool(memories serviceports.AgentMemoryService) serviceports.AgentQueryTool {
	return &recallMemoryTool{memories: memories}
}

func (t *recallMemoryTool) Name() string { return "recall_memory" }

func (t *recallMemoryTool) Description() string {
	return "Read what this organization has recorded for its agents: standing " +
		"instructions, facts agents were told, and corrections people made to " +
		"earlier proposals. Search by text, or narrow to one customer, location, " +
		"driver or carrier with subjectType and subjectId, or to one tool. Call this " +
		"before acting on a customer or a driver you have not been told about in this " +
		"conversation, and before using remember."
}

func (t *recallMemoryTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Words to look for in the memory or the name of what it is about.",
			},
			"kind": map[string]any{
				"type": "string",
				"enum": []string{
					string(agent.MemoryKindInstruction),
					string(agent.MemoryKindFact),
					string(agent.MemoryKindCorrection),
				},
			},
			"subjectType": map[string]any{
				"type": "string",
				"enum": []string{
					string(agent.MemorySubjectCustomer),
					string(agent.MemorySubjectLocation),
					string(agent.MemorySubjectWorker),
					string(agent.MemorySubjectCarrier),
				},
			},
			"subjectId": map[string]any{"type": "string"},
			"toolName": map[string]any{
				"type":        "string",
				"description": "Only memories about one tool, such as corrections to assign_move.",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": fmt.Sprintf("How many to return, at most %d.", maxRecallRows),
			},
		},
		"additionalProperties": false,
	}
}

func (t *recallMemoryTool) PermissionResource() permission.Resource {
	return permission.ResourceAgentMemory
}

func (t *recallMemoryTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	limit := optionalInt(params.Params, "limit", defaultSearchLimit)
	if limit <= 0 || limit > maxRecallRows {
		limit = maxRecallRows
	}

	req := serviceports.RecallAgentMemoriesRequest{
		TenantInfo: tenantOf(params),
		Query:      optionalString(params.Params, "query"),
		Kind:       agent.MemoryKind(optionalString(params.Params, "kind")),
		ToolName:   optionalString(params.Params, "toolName"),
		Limit:      limit,
	}
	if req.Kind != "" && !req.Kind.IsValid() {
		return nil, fmt.Errorf("kind %q is not Instruction, Fact or Correction", req.Kind)
	}

	subjectType := agent.MemorySubjectType(optionalString(params.Params, "subjectType"))
	rawID := optionalString(params.Params, "subjectId")
	if (subjectType == "") != (rawID == "") {
		return nil, fmt.Errorf("subjectType and subjectId go together; give both or neither")
	}
	if subjectType != "" {
		if !subjectType.IsValid() {
			return nil, fmt.Errorf("subjectType %q is not Customer, Location, Worker or Carrier", subjectType)
		}
		subjectID, err := pulid.Parse(rawID)
		if err != nil {
			return nil, fmt.Errorf("subjectId %q is not a record id", rawID)
		}
		req.SubjectType = subjectType
		req.SubjectID = subjectID
	}

	criteria := filtercatalog.NewCriteria("memories").At(clockFor(params))
	criteria.Text(req.Query)
	criteria.Field("kind", string(req.Kind))
	if subjectType != "" {
		criteria.Field("about", strings.ToLower(string(subjectType))+" "+rawID)
	}
	criteria.Field("tool", req.ToolName)

	memories, err := t.memories.Recall(ctx, req)
	if err != nil {
		return nil, err
	}

	rows := make([]memoryRow, 0, len(memories))
	for _, memory := range memories {
		rows = append(rows, memoryRow{
			ID:         memory.ID.String(),
			Kind:       string(memory.Kind),
			Content:    memory.Content,
			About:      memory.Scope(),
			Tool:       memory.ToolName,
			RecordedBy: recordedBy(memory.Source),
			RecordedOn: recordedDate(memory.CreatedAt),
			ExpiresOn:  expectedDate(pointerSeconds(memory.ExpiresAt), "never"),
		})
	}

	return searchResult(criteria, rows, len(rows)), nil
}

func recordedBy(source agent.MemorySource) string {
	switch source {
	case agent.MemorySourceUser:
		return "a person"
	case agent.MemorySourceAgent:
		return "an agent"
	case agent.MemorySourceDecision:
		return "a decision on a proposal"
	default:
		return string(source)
	}
}

func pointerSeconds(seconds *int64) int64 {
	if seconds == nil {
		return 0
	}

	return *seconds
}
