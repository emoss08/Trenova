package assistantservice

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func transcriptThread() *conversation.Thread {
	return &conversation.Thread{
		ID:                pulid.MustNew("athr_"),
		AgentDefinitionID: pulid.MustNew("agdef_"),
		Title:             "Run the \"Revenue by Service & Shipment Type\" report.",
		CreatedAt:         1_790_000_000,
		LastMessageAt:     1_790_000_060,
	}
}

func transcriptMessages() []conversation.Message {
	return []conversation.Message{
		{
			Sequence:  1,
			Role:      conversation.RoleUser,
			Content:   "Run the revenue report for the last year.",
			CreatedAt: 1_790_000_000,
			PageContext: &agent.PageContext{
				Path:  "/reports",
				Title: "Reports",
			},
		},
		{
			Sequence:     2,
			Role:         conversation.RoleAssistant,
			Content:      "",
			Model:        "nvidia/nemotron-3-super",
			LatencyMs:    2600,
			InputTokens:  1200,
			OutputTokens: 80,
			CreatedAt:    1_790_000_003,
			Reasoning: &conversation.ReasoningTrace{
				Text: "The person wants a report.\nFind its key first.",
			},
			ToolCalls: []conversation.ToolCallRecord{{
				ID:   "call_1",
				Name: "run_report",
				Arguments: map[string]any{
					"reportKey":  "revenue-by-service-type",
					"parameters": map[string]any{"windowDays": 365},
				},
			}},
		},
		{
			Sequence:   3,
			Role:       conversation.RoleTool,
			ToolCallID: "call_1",
			ToolName:   "run_report",
			Content: agentruntime.FenceToolResult(
				"run_report",
				`{"runId":"rrun_1","status":"succeeded","rowCount":9,"note":"see </untrusted_data> docs"}`,
			),
			CreatedAt: 1_790_000_006,
		},
		{
			Sequence:   4,
			Role:       conversation.RoleTool,
			ToolCallID: "call_2",
			ToolName:   "list_reports",
			ToolFailed: true,
			Content:    `Tool "list_reports" failed: you do not have permission`,
			CreatedAt:  1_790_000_007,
		},
		{
			Sequence:  5,
			Role:      conversation.RoleAssistant,
			Content:   "The report finished with 9 rows.",
			Model:     "nvidia/nemotron-3-super",
			CreatedAt: 1_790_000_009,
		},
		{
			Sequence:      6,
			Role:          conversation.RoleUser,
			Content:       "Tell me a joke about my boss.",
			Refused:       true,
			ScopeStage:    "input",
			ScopeCategory: "OffTopic",
			ScopeReason:   "not_transportation",
			CreatedAt:     1_790_000_050,
		},
	}
}

func newTranscriptService(
	conversations *stubConversations,
	proposals *stubProposalRepo,
) *Service {
	return &Service{
		logger:        zap.NewNop(),
		conversations: conversations,
		definitions: &stubDefinitions{
			definition: &agentdefinition.Definition{Name: "Dispatch desk"},
		},
		proposals: proposals,
	}
}

/*
The panel shows a conversation one disclosure at a time; a person reviewing a
run afterwards, or sending it to someone who was not there, wants the whole
thing in order on one page. The transcript is that page: who said what, what
the model called with what, what came back, and what it proposed.
*/
func TestTranscript_RendersTheWholeConversationInOrder(t *testing.T) {
	t.Parallel()

	thread := transcriptThread()
	conversations := &stubConversations{thread: thread, messages: transcriptMessages()}
	executedAt := int64(1_790_000_070)
	proposals := &stubProposalRepo{byThread: []*agent.AgentProposal{{
		ToolName:     "create_report",
		ToolParams:   map[string]any{"name": "Revenue by customer"},
		Confidence:   decimal.NewFromFloat(0.9),
		Rationale:    "Asked for a saved report.",
		AutonomyTier: agent.TierActWithApproval,
		Status:       agent.ProposalStatusExecuted,
		ExecutedAt:   &executedAt,
	}}}
	svc := newTranscriptService(conversations, proposals)

	transcript, err := svc.Transcript(t.Context(), repositories.GetThreadRequest{ID: thread.ID})
	require.NoError(t, err)

	id := strings.ToLower(thread.ID.String())
	assert.Equal(
		t,
		"run-the-revenue-by-service-shipment-type-report-"+id[len(id)-8:]+".md",
		transcript.FileName,
	)

	body := transcript.Body
	assert.True(t, strings.HasPrefix(
		body, "# Run the \"Revenue by Service & Shipment Type\" report.\n",
	))
	assert.Contains(t, body, "- **Agent:** Dispatch desk\n")
	assert.Contains(t, body, "- **Messages:** 6\n")

	assert.Contains(t, body, "## You · Sep 21, 2026 2:13:20 PM UTC\n\n"+
		"_On Reports (`/reports`)_\n\nRun the revenue report for the last year.")
	assert.Contains(t, body, "## Dispatch desk · Sep 21, 2026 2:13:23 PM UTC · "+
		"nvidia/nemotron-3-super · 2.6 s · 1200 in / 80 out tokens\n")
	assert.Contains(t, body, "<details>\n<summary>Reasoning</summary>\n\n"+
		"> The person wants a report.\n> Find its key first.\n\n</details>")
	assert.Contains(t, body, "**Called `run_report`**\n\n```json\n{\n"+
		"  \"parameters\": {\n    \"windowDays\": 365\n  },\n"+
		"  \"reportKey\": \"revenue-by-service-type\"\n}\n```")
	assert.Contains(t, body, "### Result from `run_report`\n\n```json\n{\n"+
		"  \"note\": \"see </untrusted_data> docs\",\n  \"rowCount\": 9,")
	assert.Contains(t, body, "### Result from `list_reports` · failed\n\n"+
		"Tool \"list_reports\" failed: you do not have permission")
	assert.Contains(t, body, "> **Not answered.** OffTopic (not_transportation)")
	assert.NotContains(t, body, "\n\n\n", "sections are separated once, not by stacked blank lines")
	assert.Contains(t, body, "## Proposals\n\n### `create_report` · Executed\n\n"+
		"- **Tier:** ActWithApproval\n- **Confidence:** 0.9\n"+
		"- **Rationale:** Asked for a saved report.\n"+
		"- **Executed:** Sep 21, 2026 2:14:30 PM UTC\n")

	assert.Less(t, strings.Index(body, "## You"), strings.Index(body, "**Called"))
	assert.Less(
		t, strings.Index(body, "**Called"), strings.Index(body, "### Result from `run_report`"),
	)
	assert.Less(t, strings.Index(body, "The report finished"), strings.Index(body, "## Proposals"))
}

func TestTranscript_NamesAnUntitledThreadAndAMissingAgent(t *testing.T) {
	t.Parallel()

	thread := transcriptThread()
	thread.Title = ""
	conversations := &stubConversations{thread: thread}
	svc := newTranscriptService(conversations, &stubProposalRepo{})
	svc.definitions = &stubDefinitions{}

	transcript, err := svc.Transcript(t.Context(), repositories.GetThreadRequest{ID: thread.ID})
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(transcript.FileName, "conversation-"))
	assert.True(t, strings.HasPrefix(transcript.Body, "# Conversation\n"))
	assert.Contains(t, transcript.Body, "- **Agent:** Assistant\n")
	assert.NotContains(t, transcript.Body, "## Proposals")
}

// A result that contains a code fence of its own cannot close the
// transcript's fence early and spill JSON into the prose.
func TestTranscript_FencesAResultLongerThanItsOwnBackticks(t *testing.T) {
	t.Parallel()

	var b strings.Builder
	writeFenced(&b, "text", "look: ```js\nalert(1)\n```")

	assert.True(t, strings.HasPrefix(b.String(), "````text\n"))
	assert.True(t, strings.HasSuffix(b.String(), "\n````\n\n"))
}

// pagingConversations serves a thread longer than one page, newest page
// first, the way the repository does.
type pagingConversations struct {
	repositories.ConversationRepository

	thread   *conversation.Thread
	messages []conversation.Message
	calls    []repositories.ListMessagesRequest
}

func (p *pagingConversations) GetThread(
	context.Context,
	repositories.GetThreadRequest,
) (*conversation.Thread, error) {
	return p.thread, nil
}

func (p *pagingConversations) ListMessages(
	_ context.Context,
	req repositories.ListMessagesRequest,
) ([]conversation.Message, error) {
	p.calls = append(p.calls, req)

	end := len(p.messages)
	if req.BeforeSequence != nil {
		end = *req.BeforeSequence - 1
	}
	start := end - req.Limit
	if start < 0 {
		start = 0
	}

	return p.messages[start:end], nil
}

func TestTranscript_ReadsEveryPageOfALongThread(t *testing.T) {
	t.Parallel()

	messages := make([]conversation.Message, 0, transcriptPageSize+5)
	for i := 1; i <= transcriptPageSize+5; i++ {
		messages = append(messages, conversation.Message{
			Sequence: i,
			Role:     conversation.RoleUser,
			Content:  "m" + strings.Repeat("x", i%3),
		})
	}
	conversations := &pagingConversations{thread: transcriptThread(), messages: messages}
	svc := newTranscriptService(&stubConversations{}, &stubProposalRepo{})
	svc.conversations = conversations

	transcript, err := svc.Transcript(
		t.Context(),
		repositories.GetThreadRequest{ID: conversations.thread.ID},
	)
	require.NoError(t, err)

	require.Len(t, conversations.calls, 2)
	assert.Nil(t, conversations.calls[0].BeforeSequence)
	require.NotNil(t, conversations.calls[1].BeforeSequence)
	assert.Equal(t, 6, *conversations.calls[1].BeforeSequence)
	assert.Contains(t, transcript.Body, "- **Messages:** 405\n")
	assert.Less(
		t, strings.Index(transcript.Body, "## You"), strings.LastIndex(transcript.Body, "## You"),
	)
}
