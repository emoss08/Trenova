package assistantservice

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const conversationTranscriptGolden = "testdata/conversation_transcript.golden.md"

func parityDelegateID() pulid.ID {
	return pulid.ID("agdef_01J0000000000000000000DELE")
}

func parityMessages() []conversation.Message {
	messages := transcriptMessages()
	messages = append(messages,
		conversation.Message{
			Sequence:  7,
			Role:      conversation.RoleAssistant,
			Content:   "Handing the lane review to the pricing desk.",
			CreatedAt: 1_790_000_051,
			ToolCalls: []conversation.ToolCallRecord{{
				ID:        "call_d",
				Name:      "delegate_task",
				Arguments: map[string]any{"task": "Review lane rates"},
			}},
		},
		conversation.Message{
			Sequence:          8,
			Role:              conversation.RoleUser,
			Kind:              conversation.MessageKindDelegated,
			AgentDefinitionID: parityDelegateID(),
			DelegateCallID:    "call_d",
			Content:           "Review lane rates",
			CreatedAt:         1_790_000_052,
		},
		conversation.Message{
			Sequence:          9,
			Role:              conversation.RoleAssistant,
			Kind:              conversation.MessageKindDelegated,
			AgentDefinitionID: parityDelegateID(),
			DelegateCallID:    "call_d",
			Content:           "Rates look current.",
			Model:             "nvidia/nemotron-3-super",
			CreatedAt:         1_790_000_053,
			ToolCalls: []conversation.ToolCallRecord{{
				ID:        "call_r",
				Name:      "list_rates",
				Arguments: map[string]any{"lane": "CHI-DAL"},
			}},
		},
		conversation.Message{
			Sequence:          10,
			Role:              conversation.RoleTool,
			Kind:              conversation.MessageKindDelegated,
			AgentDefinitionID: parityDelegateID(),
			DelegateCallID:    "call_d",
			ToolCallID:        "call_r",
			ToolName:          "list_rates",
			Content:           agentruntime.FenceToolResult("list_rates", "plain text result"),
			CreatedAt:         1_790_000_054,
		},
		conversation.Message{
			Sequence:  11,
			Role:      conversation.RoleUser,
			Kind:      conversation.MessageKindDecisionNote,
			Content:   "Approved create_report, and it ran.\nDecision on proposal ap_01.",
			CreatedAt: 1_790_000_055,
		},
		conversation.Message{
			Sequence:      12,
			Role:          conversation.RoleAssistant,
			Content:       "Withheld.",
			Refused:       true,
			ScopeCategory: "Unsafe",
			CreatedAt:     1_790_000_056,
		},
		conversation.Message{
			Sequence:    13,
			Role:        conversation.RoleTool,
			ToolFailed:  true,
			ToolVerdict: aitrace.OutcomeOverBudget,
			Content:     "budget spent",
			CreatedAt:   1_790_000_057,
		},
		conversation.Message{
			Sequence:  14,
			Role:      conversation.Role("System"),
			Content:   "a system note",
			CreatedAt: 1_790_000_058,
		},
	)

	return messages
}

func parityInput() *transcriptInput {
	thread := transcriptThread()
	thread.ID = pulid.ID("athr_01J00000000000000000PARITY")
	executedAt := int64(1_790_000_070)

	return &transcriptInput{
		Thread:     thread,
		AgentName:  "Dispatch desk",
		Delegates:  map[pulid.ID]string{parityDelegateID(): "Pricing desk"},
		Messages:   parityMessages(),
		ExportedAt: 1_790_000_100,
		Proposals: []*agent.AgentProposal{
			{
				ToolName:     "create_report",
				ToolParams:   map[string]any{"name": "Revenue by customer"},
				Confidence:   decimal.NewFromFloat(0.9),
				Rationale:    "Asked for a saved report.",
				AutonomyTier: agent.TierActWithApproval,
				Status:       agent.ProposalStatusExecuted,
				ExecutedAt:   &executedAt,
				ExecutionResult: &agent.ToolExecutionResult{
					Action: "created",
					Kind:   "report",
					Name:   "Revenue by customer",
					IDs:    map[string]string{"definitionId": "rd_01"},
				},
			},
			{
				ToolName:       "assign_move",
				ToolParams:     map[string]any{"moveId": "smv_1"},
				AutonomyTier:   agent.TierActWithApproval,
				Status:         agent.ProposalStatusExecutionFailed,
				ExecutionError: "the move is already assigned",
			},
		},
	}
}

func TestTranscript_ConversationDocumentIsUnchanged(t *testing.T) {
	t.Parallel()

	want, err := os.ReadFile(filepath.FromSlash(conversationTranscriptGolden))
	require.NoError(t, err)

	assert.Equal(t, string(want), renderTranscript(parityInput()))
}
