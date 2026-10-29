package runtimetools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const (
	DelegateTaskName     = "delegate_task"
	MaxDelegateRecords   = 8
	maxDelegateRecordID  = 100
	MaxSharedResults     = 4
	DelegateAgentParam   = "agentId"
	DelegateTaskParam    = "task"
	DelegateRecordsParam = "records"
	DelegateSharedParam  = "shareResults"
)

// delegateTaskDescription is a constant for the same reason
// findToolsDescription is: it is addressed to a model, and the i18n extractor
// would otherwise harvest it.
const delegateTaskDescription = "Hand a lookup to another agent that holds read tools " +
	"you do not, or split independent lookups to run side by side: several calls in one " +
	"reply run at the same time. It works as the same person, only reads, changes nothing " +
	"and cannot see this conversation, so the task must say everything it needs and what " +
	"to hand back. The result is its answer and every result it read, word for word. Hand " +
	"over the records the task is about by their ids in records, and the results of this " +
	"turn's own calls it should work from in shareResults, rather than retyping them into " +
	"the task. Never ask it to change anything: make every change with your own tools."

// DelegateTaskSpec is the tool a turn hands a task to another agent with.
// agentId is limited to the agents the turn may ask, so a model cannot name
// one it was not offered.
func DelegateTaskSpec(delegates []agentturn.RuntimeDelegate) kernelports.ToolSpec {
	ids := make([]string, 0, len(delegates))
	for _, delegate := range delegates {
		ids = append(ids, delegate.ID.String())
	}

	return kernelports.ToolSpec{
		Name:        DelegateTaskName,
		Description: delegateTaskDescription,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				DelegateAgentParam: map[string]any{
					"type": "string",
					"enum": ids,
					"description": "The agentId of the agent to ask, from the agents you " +
						"can ask.",
				},
				DelegateTaskParam: map[string]any{
					"type": "string",
					"description": "What the agent should look up and what it should hand " +
						"back, with every name, id, date and figure it needs. \"List this " +
						"month's late deliveries for Acme Foods and hand back each load's PRO " +
						"number and how late it was.\"",
				},
				DelegateRecordsParam: map[string]any{
					toolschema.KeyType:     toolschema.TypeArray,
					toolschema.KeyMaxItems: MaxDelegateRecords,
					toolschema.KeyDescription: "The records the task is about, each by its " +
						"kind and id, so the agent opens them rather than reading a copy you " +
						"typed.",
					toolschema.KeyItems: map[string]any{
						toolschema.KeyType: toolschema.TypeObject,
						toolschema.KeyProperties: map[string]any{
							"entityType": agenttoolschema.Enum(
								"The kind of record.", agenttoolschema.RecordEntities,
							),
							"id": map[string]any{
								toolschema.KeyType:        toolschema.TypeString,
								toolschema.KeyMaxLength:   maxDelegateRecordID,
								toolschema.KeyDescription: "The record's id, from the tool that found it.",
							},
						},
						toolschema.KeyRequired:             []string{"entityType", "id"},
						toolschema.KeyAdditionalProperties: false,
					},
				},
				DelegateSharedParam: map[string]any{
					toolschema.KeyType:     toolschema.TypeArray,
					toolschema.KeyMaxItems: MaxSharedResults,
					toolschema.KeyDescription: "The ids of calls you made in this turn whose " +
						"results the agent should work from, handed over as they came back. " +
						"Each result may be at most 8 KiB.",
					toolschema.KeyItems: map[string]any{toolschema.KeyType: toolschema.TypeString},
				},
			},
			toolschema.KeyRequired: []string{DelegateAgentParam, DelegateTaskParam},
			"additionalProperties": false,
		},
	}
}
