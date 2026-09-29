package agentruntime

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type groundingEffects struct {
	*localEffects

	supports bool
	events   []serviceports.StreamEvent
}

func (fx *groundingEffects) Emit(event serviceports.StreamEvent) {
	fx.events = append(fx.events, event)
}

func (fx *groundingEffects) Supports(change string) bool {
	if change == changeGroundingGuard {
		return fx.supports
	}

	return true
}

func (fx *groundingEffects) regrounded() []serviceports.AssistantReplyRegroundedEvent {
	out := make([]serviceports.AssistantReplyRegroundedEvent, 0, 2)
	for _, event := range fx.events {
		if event.Event != serviceports.AssistantEventReplyRegrounded {
			continue
		}
		if data, ok := event.Data.(serviceports.AssistantReplyRegroundedEvent); ok {
			out = append(out, data)
		}
	}

	return out
}

func groundedShipmentSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"customerId": map[string]any{"type": "string"},
			"baseRate":   map[string]any{"type": "number"},
			"commodities": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"commodityId": map[string]any{"type": "string"},
						"pieces":      map[string]any{"type": "integer"},
					},
				},
			},
			"additionalCharges": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "object"},
			},
		},
		"required": []string{"customerId"},
	}
}

type groundingRun struct {
	result     *serviceports.RunResult
	completion *scriptedCompletion
	fx         *groundingEffects
}

func runGrounded(
	t *testing.T,
	supports bool,
	filed map[string]any,
	replies ...string,
) groundingRun {
	t.Helper()

	turns := []*serviceports.ChatCompletionResult{toolTurn("create_shipment", filed)}
	for _, reply := range replies {
		turns = append(turns, textTurn(reply))
	}
	completion := &scriptedCompletion{Turns: turns}
	tool := &agentruntimetest.StubActionTool{
		ToolName: "create_shipment",
		Tier:     agent.TierPropose,
		Schema:   groundedShipmentSchema(),
	}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{
		Tools: []serviceports.AgentTool{tool},
	}, nil)
	req := &serviceports.RunRequest{
		Definition: testDefinition("create_shipment"),
		Actor:      testActor(),
		Input:      "Copy shipment PRO-100 for customer cus_1, same lane, for Tuesday.",
	}
	fx := &groundingEffects{supports: supports}
	fx.localEffects = &localEffects{
		s:    rt,
		ctx:  t.Context(),
		emit: func(serviceports.StreamEvent) {},
	}

	result, err := rt.Drive(rt.OpenTurn(t.Context(), req), fx)
	require.NoError(t, err)

	return groundingRun{result: result, completion: completion, fx: fx}
}

func lastUserMessage(req *serviceports.ChatCompletionRequest) string {
	for idx := len(req.Messages) - 1; idx >= 0; idx-- {
		if req.Messages[idx].Role == serviceports.RoleUser {
			return req.Messages[idx].Content
		}
	}

	return ""
}

func TestGroundingGuard_RewritesAReplyThatClaimsWhatWasNotFiled(t *testing.T) {
	t.Parallel()

	run := runGrounded(t, true, map[string]any{"customerId": "cus_1"},
		"I proposed the copy of PRO-100 and copied its commodities across.",
		"I proposed a copy of PRO-100 for cus_1. Check the card before you approve it.",
	)

	require.Equal(t, 3, run.completion.CallCount, "the reply is asked for once more")
	correction := lastUserMessage(run.completion.Requests[2])
	assert.Contains(t, correction, "commodities")
	assert.Contains(t, correction, "create_shipment")
	assert.Equal(t,
		"I proposed a copy of PRO-100 for cus_1. Check the card before you approve it.",
		run.result.Reply)

	regrounded := run.fx.regrounded()
	require.Len(t, regrounded, 1)
	assert.Equal(t, serviceports.RegroundRewrite, regrounded[0].Action)
	assert.Equal(t, []string{"commodities"}, regrounded[0].Fields)
	assert.Contains(t, eventNames(run.fx.events), serviceports.AssistantEventRetrying,
		"the reader drops the draft it was shown")
}

func TestGroundingGuard_NotesTheCardWhenTheRewriteStillDrifts(t *testing.T) {
	t.Parallel()

	run := runGrounded(t, true, map[string]any{"customerId": "cus_1"},
		"Proposed, with the commodities copied.",
		"Proposed, and the commodities come along too.",
	)

	require.Equal(t, 3, run.completion.CallCount, "the reply is asked for once more, only once")
	assert.True(t, strings.HasPrefix(run.result.Reply,
		"Proposed, and the commodities come along too."))
	assert.True(t, strings.HasSuffix(run.result.Reply, groundingNote))

	regrounded := run.fx.regrounded()
	require.Len(t, regrounded, 2)
	assert.Equal(t, serviceports.RegroundRewrite, regrounded[0].Action)
	assert.Equal(t, serviceports.RegroundNote, regrounded[1].Action)
}

func TestGroundingGuard_CatchesAFigureNothingReturnedOrFiled(t *testing.T) {
	t.Parallel()

	run := runGrounded(t, true, map[string]any{"customerId": "cus_1", "baseRate": 1250},
		"Proposed at a base rate of $4,800.",
		"Proposed at a base rate of $1,250.",
	)

	require.Equal(t, 3, run.completion.CallCount)
	regrounded := run.fx.regrounded()
	require.Len(t, regrounded, 1)
	assert.Equal(t, []string{"4,800"}, regrounded[0].Figures)
	assert.Contains(t, lastUserMessage(run.completion.Requests[2]), "4,800")
	assert.Equal(t, "Proposed at a base rate of $1,250.", run.result.Reply)
}

func TestGroundingGuard_LetsAReplySayWhatWasLeftOut(t *testing.T) {
	t.Parallel()

	run := runGrounded(t, true, map[string]any{"customerId": "cus_1"},
		"I proposed the copy. I did not copy the commodities; add them on the card if needed.",
	)

	assert.Equal(t, 2, run.completion.CallCount)
	assert.Empty(t, run.fx.regrounded())
}

func TestGroundingGuard_AcceptsWhatWasFiled(t *testing.T) {
	t.Parallel()

	run := runGrounded(t, true, map[string]any{
		"customerId": "cus_1",
		"commodities": []any{
			map[string]any{"commodityId": "com_1", "pieces": 40},
		},
	},
		"I proposed the copy with its commodities: 40 pieces of com_1.",
	)

	assert.Equal(t, 2, run.completion.CallCount)
	assert.Empty(t, run.fx.regrounded())
}

func TestGroundingGuard_LeavesAnExecutionStartedBeforeItAlone(t *testing.T) {
	t.Parallel()

	run := runGrounded(t, false, map[string]any{"customerId": "cus_1"},
		"Proposed, with the commodities copied.",
	)

	assert.Equal(t, 2, run.completion.CallCount)
	assert.Empty(t, run.fx.regrounded())
	assert.Equal(t, "Proposed, with the commodities copied.", run.result.Reply)
}

func TestGroundingGuard_ATurnThatFiledNothingIsNotChecked(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		textTurn("Your fleet ran 4,812 loads last year and every one had commodities."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fx := &groundingEffects{supports: true}
	fx.localEffects = &localEffects{s: rt, ctx: t.Context(), emit: func(serviceports.StreamEvent) {}}

	_, err := rt.Drive(rt.OpenTurn(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "how busy were we?",
	}), fx)
	require.NoError(t, err)

	assert.Equal(t, 1, completion.CallCount)
	assert.Empty(t, fx.regrounded())
}

func TestFiledLabels_ComeFromTheToolsSchema(t *testing.T) {
	t.Parallel()

	labels := schemaLabels(groundedShipmentSchema())

	assert.Contains(t, labels, schemaLabel{path: "commodities", words: "commodities"})
	assert.Contains(t, labels, schemaLabel{path: "additionalCharges", words: "additional charges"})
	assert.Contains(t, labels, schemaLabel{path: "baseRate", words: "base rate"})
	for _, label := range labels {
		assert.NotEqual(t, "customerId", label.path, "an id is not something a reply claims")
		assert.NotEqual(t, "commodities[].pieces", label.path,
			"a single word scalar is too common to hold a reply to")
	}
}

func eventNames(events []serviceports.StreamEvent) []string {
	names := make([]string, 0, len(events))
	for _, event := range events {
		names = append(names, event.Event)
	}

	return names
}
