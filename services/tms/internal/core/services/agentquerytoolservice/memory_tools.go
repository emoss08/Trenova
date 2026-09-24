package agentquerytoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

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
	Match      string       `json:"match,omitempty"`

	FromOutsideContent bool `json:"fromOutsideContent,omitempty"`
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
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			paramQuery: map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: "Words to look for in the memory, the name of what it is " +
					"about, or its tool. Every word need not appear; the closest come first.",
			},
			"id": map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: "Optional: one memory's id, to read the whole of a memory " +
					"your instructions show cut short.",
			},
			"kind": map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: "Optional: Instruction for standing rules to follow, Fact " +
					"for things agents were told, or Correction for fixes people made to " +
					"earlier proposals.",
				"enum": []string{
					string(agent.MemoryKindInstruction),
					string(agent.MemoryKindFact),
					string(agent.MemoryKindCorrection),
				},
			},
			"subjectType": map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: "Optional: the kind of record subjectId names. Give both " +
					"or neither.",
				"enum": []string{
					string(agent.MemorySubjectCustomer),
					string(agent.MemorySubjectLocation),
					string(agent.MemorySubjectWorker),
					string(agent.MemorySubjectCarrier),
				},
			},
			"subjectId": map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: "Optional: the record's id, from list_customers, " +
					"list_locations, search_worker or list_carriers to match subjectType, " +
					"or the page you are on. Give it with subjectType.",
			},
			"toolName": map[string]any{
				toolschema.KeyType:        toolschema.TypeString,
				toolschema.KeyDescription: "Only memories about one tool, such as corrections to assign_move.",
			},
			paramLimit: map[string]any{
				toolschema.KeyType: toolschema.TypeInteger,
				toolschema.KeyDescription: fmt.Sprintf(
					"How many to return; %d when left out, at most %d.",
					agent.DefaultMemoryRecallLimit,
					agent.MaxMemoryRecallLimit,
				),
			},
		},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *recallMemoryTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceAgentMemory,
		reads:    agent.ExternalReadMarked,
		source:   agent.TaintSourceMemory,
		rationale: "Reads memories earlier runs saved, which carry the taint of the run that " +
			"wrote them.",
	})
}

func (t *recallMemoryTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	limit := optionalInt(params.Params, "limit", agent.DefaultMemoryRecallLimit)
	if limit <= 0 {
		limit = agent.DefaultMemoryRecallLimit
	}
	limit = min(limit, agent.MaxMemoryRecallLimit)

	req := serviceports.RecallAgentMemoriesRequest{
		TenantInfo:        tenantOf(params),
		AgentDefinitionID: params.AgentDefinitionID,
		Query:             optionalString(params.Params, "query"),
		Kind:              agent.MemoryKind(optionalString(params.Params, "kind")),
		ToolName:          optionalString(params.Params, "toolName"),
		Limit:             limit,
	}
	if req.Kind != "" && !req.Kind.IsValid() {
		return nil, fmt.Errorf("kind %q is not Instruction, Fact or Correction", req.Kind)
	}
	rawMemoryID := optionalString(params.Params, "id")
	if rawMemoryID != "" {
		memoryID, err := pulid.Parse(rawMemoryID)
		if err != nil {
			return nil, fmt.Errorf("id %q is not a memory id", rawMemoryID)
		}
		req.IDs = []pulid.ID{memoryID}
	}

	subjectType := agent.MemorySubjectType(optionalString(params.Params, "subjectType"))
	rawID := optionalString(params.Params, "subjectId")
	if (subjectType == "") != (rawID == "") {
		return nil, fmt.Errorf("subjectType and subjectId go together; give both or neither")
	}
	if subjectType != "" {
		if !subjectType.IsValid() {
			return nil, fmt.Errorf(
				"subjectType %q is not Customer, Location, Worker or Carrier",
				subjectType,
			)
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
	criteria.Field("id", rawMemoryID)
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
	tainted := make([]agent.RecordRef, 0, len(memories))
	for _, recalled := range memories {
		memory := recalled.Memory
		tainted = append(tainted, memory.TaintedRecords()...)
		rows = append(rows, memoryRow{
			ID:         memory.ID.String(),
			Kind:       string(memory.Kind),
			Content:    memory.Content,
			About:      memory.About(),
			Tool:       memory.ToolName,
			RecordedBy: recordedBy(memory.Source),
			RecordedOn: recordedDate(memory.CreatedAt),
			ExpiresOn:  expectedDate(pointerSeconds(memory.ExpiresAt), "never"),
			Match:      string(recalled.Match),

			FromOutsideContent: memory.DrawnFromOutside(),
		})
	}

	return recallOutcome{
		searchOutcome: searchResult(criteria, rows, len(rows)),
		tainted:       tainted,
	}, nil
}

// recallOutcome is a recall's answer, which names the memories in it that
// were written by a run that had read outside content. Only the runtime reads
// that; the model reads the rows.
type recallOutcome struct {
	searchOutcome

	tainted []agent.RecordRef
}

func (o recallOutcome) TaintedRecords() []agent.RecordRef { return o.tainted }

func recordedBy(source agent.MemorySource) string {
	switch source {
	case agent.MemorySourceUser:
		return "a person"
	case agent.MemorySourceAgent:
		return "an agent"
	case agent.MemorySourceDecision:
		return "a decision on a proposal"
	case agent.MemorySourceFeedback:
		return "people's ratings of an agent's work"
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
