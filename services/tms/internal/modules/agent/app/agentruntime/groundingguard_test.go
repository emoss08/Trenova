package agentruntime

import (
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/replycheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type groundingEffects struct {
	*localEffects

	events []kernelports.StreamEvent
}

func (fx *groundingEffects) Emit(event kernelports.StreamEvent) {
	fx.events = append(fx.events, event)
}

func (fx *groundingEffects) regrounded() []assistantports.AssistantReplyRegroundedEvent {
	out := make([]assistantports.AssistantReplyRegroundedEvent, 0, 2)
	for _, event := range fx.events {
		if event.Event != assistantports.AssistantEventReplyRegrounded {
			continue
		}
		if data, ok := event.Data.(assistantports.AssistantReplyRegroundedEvent); ok {
			out = append(out, data)
		}
	}

	return out
}

type groundingRun struct {
	result     *agentruntimeports.RunResult
	completion *scriptedCompletion
	fx         *groundingEffects
}

func runGrounded(
	t *testing.T,
	filed map[string]any,
	replies ...string,
) groundingRun {
	t.Helper()

	return runGroundedWith(t, &groundingEffects{}, filed, replies...)
}

func runGroundedWith(
	t *testing.T,
	fx *groundingEffects,
	filed map[string]any,
	replies ...string,
) groundingRun {
	t.Helper()

	turns := []*llmports.ChatCompletionResult{runtimetest.ToolTurn("shipment_create", filed)}
	for _, reply := range replies {
		turns = append(turns, runtimetest.TextTurn(reply))
	}
	completion := &scriptedCompletion{Turns: turns}
	tool := &agenttest.StubActionTool{
		ToolName: "shipment_create",
		Tier:     agent.TierPropose,
		Schema:   runtimetest.GroundedShipmentSchema(),
	}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{
		Tools: []agenttoolports.AgentTool{tool},
	}, nil)
	req := &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("shipment_create"),
		Actor:      runtimetest.Actor(),
		Input:      "Copy shipment PRO-100 for customer cus_1, same lane, for Tuesday.",
	}
	fx.localEffects = &localEffects{
		s:    rt,
		ctx:  t.Context(),
		emit: func(kernelports.StreamEvent) {},
	}

	result, err := rt.Drive(rt.OpenTurn(t.Context(), req), fx)
	require.NoError(t, err)

	return groundingRun{result: result, completion: completion, fx: fx}
}

func lastUserMessage(req *llmports.ChatCompletionRequest) string {
	for idx := len(req.Messages) - 1; idx >= 0; idx-- {
		if role := req.Messages[idx].Role; role == llmports.RoleUser ||
			role == llmports.RoleSystem {
			return req.Messages[idx].Content
		}
	}

	return ""
}

func TestGroundingGuard_RewritesAReplyThatClaimsWhatWasNotFiled(t *testing.T) {
	t.Parallel()

	run := runGrounded(t, map[string]any{"customerId": "cus_1"},
		"I proposed the copy of PRO-100 and copied its commodities across.",
		"I proposed a copy of PRO-100 for cus_1. Check the card before you approve it.",
	)

	require.Equal(t, 3, run.completion.CallCount, "the reply is asked for once more")
	correction := lastUserMessage(run.completion.Requests[2])
	assert.Equal(t, ReaskGrounding, run.completion.Requests[2].Reask)
	assert.Contains(t, correction, "commodities")
	assert.Contains(t, correction, "shipment_create")
	assert.Equal(t,
		"I proposed a copy of PRO-100 for cus_1. Check the card before you approve it.",
		run.result.Reply)

	regrounded := run.fx.regrounded()
	require.Len(t, regrounded, 1)
	assert.Equal(t, assistantports.RegroundRewrite, regrounded[0].Action)
	assert.Equal(t, []string{"commodities"}, regrounded[0].Fields)
	assert.Contains(t, eventNames(run.fx.events), assistantports.AssistantEventRetrying,
		"the reader drops the draft it was shown")
}

func TestGroundingGuard_NotesTheCardWhenTheRewriteStillDrifts(t *testing.T) {
	t.Parallel()

	run := runGrounded(t, map[string]any{"customerId": "cus_1"},
		"Proposed, with the commodities copied.",
		"Proposed, and the commodities come along too.",
	)

	require.Equal(t, 3, run.completion.CallCount, "the reply is asked for once more, only once")
	assert.True(t, strings.HasPrefix(run.result.Reply,
		"Proposed, and the commodities come along too."))
	assert.True(t, strings.HasSuffix(run.result.Reply, replycheck.GroundingNote))

	regrounded := run.fx.regrounded()
	require.Len(t, regrounded, 2)
	assert.Equal(t, assistantports.RegroundRewrite, regrounded[0].Action)
	assert.Equal(t, assistantports.RegroundNote, regrounded[1].Action)
}

func TestGroundingGuard_CatchesAFigureNothingReturnedOrFiled(t *testing.T) {
	t.Parallel()

	run := runGrounded(t, map[string]any{"customerId": "cus_1", "baseRate": 1250},
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

	run := runGrounded(t, map[string]any{"customerId": "cus_1"},
		"I proposed the copy. I did not copy the commodities; add them on the card if needed.",
	)

	assert.Equal(t, 2, run.completion.CallCount)
	assert.Empty(t, run.fx.regrounded())
}

func TestGroundingGuard_AcceptsWhatWasFiled(t *testing.T) {
	t.Parallel()

	run := runGrounded(t, map[string]any{
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

func TestGroundingGuard_AConversationThatReadNothingIsNotChecked(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.TextTurn("Detention usually runs $50 to $75 an hour after two free hours."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fx := &groundingEffects{}
	fx.localEffects = &localEffects{s: rt, ctx: t.Context(), emit: func(kernelports.StreamEvent) {}}

	_, err := rt.Drive(rt.OpenTurn(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition(),
		Actor:      runtimetest.Actor(),
		Input:      "what does detention usually cost?",
	}), fx)
	require.NoError(t, err)

	assert.Equal(t, 1, completion.CallCount)
	assert.Empty(t, fx.regrounded())
}

// readGrounded is a turn that reads two report totals, then answers with the
// replies given, filing nothing.
func readGrounded(
	t *testing.T,
	tools []string,
	replies ...string,
) groundingRun {
	t.Helper()

	return readGroundedWith(t, &groundingEffects{}, tools, replies...)
}

func readGroundedWith(
	t *testing.T,
	fx *groundingEffects,
	tools []string,
	replies ...string,
) groundingRun {
	t.Helper()

	turns := []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("report_get_run", map[string]any{}),
	}
	for _, reply := range replies {
		turns = append(turns, runtimetest.TextTurn(reply))
	}
	completion := &scriptedCompletion{Turns: turns}
	registry := &stubQueryRegistry{}
	for _, name := range tools {
		registry.Tools = append(registry.Tools, &agenttest.StubQueryTool{
			ToolName: name,
			Result: map[string]any{
				"billed":    "87310.40",
				"stillOpen": "45662.16",
				"onTime":    0.914,
			},
		})
	}
	rt := newRuntime(completion, registry, &stubActionRegistry{}, nil)
	rewire(rt, func(p *Params) { p.Scripts = scriptsOn(aiprovider.ScriptSettings{Scripts: true}) })
	fx.localEffects = &localEffects{s: rt, ctx: t.Context(), emit: func(kernelports.StreamEvent) {}}

	result, err := rt.Drive(rt.OpenTurn(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition(tools...),
		Actor:      runtimetest.Actor(),
		Input:      "how much of what we billed in the last 30 days is still open?",
	}), fx)
	require.NoError(t, err)

	return groundingRun{result: result, completion: completion, fx: fx}
}

// DB-186: an answer that filed nothing divided one total by another in its head
// and stated a share no tool returned.
func TestGroundingGuard_RewritesAShareNothingRead(t *testing.T) {
	t.Parallel()

	run := readGrounded(t, []string{"report_get_run", codemode.ToolRunScript},
		"About 52.3% of the $87,310.40 billed is still open.",
		"Of the $87,310.40 billed, $45,662.16 is still open.",
	)

	require.Equal(t, 3, run.completion.CallCount, "the reply is asked for once more")
	correction := lastUserMessage(run.completion.Requests[2])
	assert.True(t, strings.HasPrefix(correction, replycheck.AmountsCorrectionLead))
	assert.Contains(t, correction, "52.3")
	assert.Contains(t, correction, codemode.ToolRunScript)
	assert.Equal(t, "Of the $87,310.40 billed, $45,662.16 is still open.", run.result.Reply)
	assert.True(t, strings.HasSuffix(correction, replycheck.UnseenCorrection),
		"a rewrite answers the person, not the correction")

	regrounded := run.fx.regrounded()
	require.Len(t, regrounded, 1)
	assert.Equal(t, assistantports.RegroundRewrite, regrounded[0].Action)
	assert.Equal(t, []string{"52.3"}, regrounded[0].Figures)
}

func TestGroundingGuard_WithoutAnalysisAShareIsLeftOut(t *testing.T) {
	t.Parallel()

	run := readGrounded(t, []string{"report_get_run"},
		"About 52.3% is still open.",
		"Of the $87,310.40 billed, $45,662.16 is still open.",
	)

	require.Equal(t, 3, run.completion.CallCount)
	correction := lastUserMessage(run.completion.Requests[2])
	assert.NotContains(t, correction, codemode.ToolRunScript)
	assert.Contains(t, correction, "leave it out")
}

func TestGroundingGuard_NamesTheFiguresWhenTheRewriteStillStatesThem(t *testing.T) {
	t.Parallel()

	run := readGrounded(t, []string{"report_get_run"},
		"About 52.3% is still open.",
		"Roughly 52.3% is still open.",
	)

	require.Equal(t, 3, run.completion.CallCount, "the reply is asked for once more, only once")
	assert.True(t, strings.HasPrefix(run.result.Reply, "Roughly 52.3% is still open."))
	assert.Contains(t, run.result.Reply, "Check the figures in this reply")
	assert.Contains(t, run.result.Reply, "52.3 came from no record")

	regrounded := run.fx.regrounded()
	require.Len(t, regrounded, 2)
	assert.Equal(t, assistantports.RegroundNote, regrounded[1].Action)
}

func TestGroundingGuard_AcceptsWhatTheConversationRead(t *testing.T) {
	t.Parallel()

	run := readGrounded(t, []string{"report_get_run"},
		"Of about $87.3k billed, $45,662.16 is still open, and 91.4% delivered on time "+
			"across 31 days.",
	)

	assert.Equal(t, 2, run.completion.CallCount)
	assert.Empty(t, run.fx.regrounded())
}

func eventNames(events []kernelports.StreamEvent) []string {
	names := make([]string, 0, len(events))
	for _, event := range events {
		names = append(names, event.Event)
	}

	return names
}

// A rate confirmation read as a PDF gives the model figures the guard never
// sees. Holding the reply to them rewrote a correct summary into "please
// disregard the figures from your attachment".
func TestGroundingGuard_AFileOnlyTheModelReadsIsNotHeldAgainstIt(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("report_get_run", map[string]any{}),
		runtimetest.TextTurn(
			"The confirmation totals $3,610.00: $3,200.00 line haul and $410.00 fuel.",
		),
	}}
	registry := &stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{
		&agenttest.StubQueryTool{ToolName: "report_get_run", Result: map[string]any{}},
	}}
	rt := newRuntime(completion, registry, &stubActionRegistry{}, nil)
	fx := &groundingEffects{}
	fx.localEffects = &localEffects{s: rt, ctx: t.Context(), emit: func(kernelports.StreamEvent) {}}

	turn := rt.OpenTurn(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("report_get_run"),
		Actor:      runtimetest.Actor(),
		Input:      "what's the total rate on it?",
	})
	turn.messages[len(turn.messages)-1].Files = []llmports.MessageFile{{FileName: "ratecon.pdf"}}
	_, err := rt.Drive(turn, fx)
	require.NoError(t, err)

	assert.Equal(t, 2, completion.CallCount)
	assert.Empty(t, fx.regrounded())
}

// An analysis table's rows reach the model as JSON arrays. Read with every
// comma as a thousands separator, the whole row was one figure that never
// parsed, and a reply quoting the table was rewritten.
func TestGroundingGuard_ReadsEachFigureOfAJSONRow(t *testing.T) {
	t.Parallel()

	run := runGroundedResult(t, &groundingEffects{},
		[]any{[]any{"Riverside", 39, 12937.5, 0.708}},
		"Proposed. Riverside cost $12,937.50, 70.8% of the total.",
	)

	assert.Equal(t, 2, run.completion.CallCount)
	assert.Empty(t, run.fx.regrounded())
}

func TestGroundingGuard_AcceptsAFigureTheResultCarries(t *testing.T) {
	t.Parallel()

	run := runGroundedResult(t, &groundingEffects{},
		[]any{"12937.5"},
		"Proposed. Riverside cost $12,937.50.",
	)

	assert.Empty(t, run.fx.regrounded())
}

// runGroundedResult files a shipment_create whose result carries rows, then
// answers with the replies given.
func runGroundedResult(
	t *testing.T,
	fx *groundingEffects,
	rows []any,
	replies ...string,
) groundingRun {
	t.Helper()

	turns := []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("shipment_create", map[string]any{"customerId": "cus_1"}),
	}
	for _, reply := range replies {
		turns = append(turns, runtimetest.TextTurn(reply))
	}
	completion := &scriptedCompletion{Turns: turns}
	tool := &agenttest.StubActionTool{
		ToolName: "shipment_create",
		Tier:     agent.TierPropose,
		Schema:   runtimetest.GroundedShipmentSchema(),
	}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{
		Tools: []agenttoolports.AgentTool{tool},
	}, nil)
	fx.localEffects = &localEffects{s: rt, ctx: t.Context(), emit: func(kernelports.StreamEvent) {}}

	turn := rt.OpenTurn(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("shipment_create"),
		Actor:      runtimetest.Actor(),
		Input:      "Copy shipment PRO-100 for customer cus_1.",
	})
	encoded, err := sonic.MarshalString(map[string]any{"rows": rows})
	require.NoError(t, err)
	turn.system += "\n" + encoded
	result, err := rt.Drive(turn, fx)
	require.NoError(t, err)

	return groundingRun{result: result, completion: completion, fx: fx}
}

// Arithmetic on the person's own two figures restates what they said.
func TestGroundingGuard_ADifferenceOfThePersonsOwnFiguresIsStated(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("invoice_get", map[string]any{}),
		runtimetest.TextTurn(
			"The $3,200 check leaves $500 open on the $3,700 invoice; a $517.40 fee would not.",
		),
		runtimetest.TextTurn("The $3,200 check leaves $500 open on the $3,700 invoice."),
	}}
	registry := &stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{
		&agenttest.StubQueryTool{ToolName: "invoice_get", Result: map[string]any{"total": "3700"}},
	}}
	rt := newRuntime(completion, registry, &stubActionRegistry{}, nil)
	fx := &groundingEffects{}
	fx.localEffects = &localEffects{s: rt, ctx: t.Context(), emit: func(kernelports.StreamEvent) {}}

	_, err := rt.Drive(rt.OpenTurn(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("invoice_get"),
		Actor:      runtimetest.Actor(),
		Input:      "peak sent a check for 3,200 but their invoice was 3,700. what do we do with it",
	}), fx)
	require.NoError(t, err)

	regrounded := fx.regrounded()
	require.Len(t, regrounded, 1)
	assert.Equal(t, []string{"517.40"}, regrounded[0].Figures, "only the figure nobody stated")
}

// DB-189: "If you have the receipt, I can add the vendor and transaction
// reference" offers to add a field; it does not say the change holds one.
func TestGroundingGuard_AnOfferToAddAFieldIsNoClaim(t *testing.T) {
	t.Parallel()

	run := runGroundedWith(t, &groundingEffects{},
		map[string]any{"customerId": "cus_1"},
		"I proposed the copy of PRO-100. If you have the packing list, I can add the "+
			"commodities before you approve it.",
	)

	assert.Equal(t, 2, run.completion.CallCount)
	assert.Empty(t, run.fx.regrounded())
}

func TestGroundingGuard_AnOfferBesideAClaimStillChecksTheClaim(t *testing.T) {
	t.Parallel()

	run := runGroundedWith(t, &groundingEffects{},
		map[string]any{"customerId": "cus_1"},
		"I proposed the copy with its commodities. I can add more commodities if you need them.",
		"I proposed the copy of PRO-100.",
	)

	assert.Equal(t, 3, run.completion.CallCount)
	require.Len(t, run.fx.regrounded(), 1)
	assert.Equal(t, []string{"commodities"}, run.fx.regrounded()[0].Fields)
}

// A share whose two sides the reply gives is shown working: the guard checks
// the division instead of sending the agent to analysis_run for it.
func TestGroundingGuard_AcceptsArithmeticOnFiguresTheReplyGives(t *testing.T) {
	t.Parallel()

	run := readGrounded(t, []string{"report_get_run"},
		"Of the $87,310.40 billed, $45,662.16 is still open: 52.3%, leaving $41,648.24 paid.",
	)

	assert.Equal(t, 2, run.completion.CallCount)
	assert.Empty(t, run.fx.regrounded())
}

func TestGroundingGuard_WrongArithmeticOnGivenFiguresIsStillRewritten(t *testing.T) {
	t.Parallel()

	run := readGrounded(t, []string{"report_get_run"},
		"Of the $87,310.40 billed, $45,662.16 is still open: 61.4%.",
		"Of the $87,310.40 billed, $45,662.16 is still open.",
	)

	require.Len(t, run.fx.regrounded(), 1)
	assert.Equal(t, []string{"61.4"}, run.fx.regrounded()[0].Figures)
}

// Told 112 gallons at $3.89, "$435.68" is the person's own two figures
// multiplied, whether or not the reply repeats them.
func TestGroundingGuard_AProductOfThePersonsFiguresIsStated(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("agent_find_page", map[string]any{}),
		runtimetest.TextTurn(
			"Key it in at Fuel purchases; the total paid should be about $435.68.",
		),
	}}
	registry := &stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{
		&agenttest.StubQueryTool{ToolName: "agent_find_page", Result: map[string]any{}},
	}}
	rt := newRuntime(completion, registry, &stubActionRegistry{}, nil)
	fx := &groundingEffects{}
	fx.localEffects = &localEffects{s: rt, ctx: t.Context(), emit: func(kernelports.StreamEvent) {}}

	_, err := rt.Drive(rt.OpenTurn(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("agent_find_page"),
		Actor:      runtimetest.Actor(),
		Input:      "log 112 gallons of diesel for TRC-001 bought in Texas yesterday at $3.89",
	}), fx)
	require.NoError(t, err)

	assert.Equal(t, 2, completion.CallCount)
	assert.Empty(t, fx.regrounded())
}

func TestGroundingGuard_AFiledReplysWorkingCounts(t *testing.T) {
	t.Parallel()

	run := runGroundedResult(t, &groundingEffects{},
		[]any{"1121.36", "898.80"},
		"Proposed at $1,121.36 in total: $898.80 linehaul and $222.56 in other charges.",
	)

	assert.Equal(t, 2, run.completion.CallCount)
	assert.Empty(t, run.fx.regrounded())
}

func TestGroundingGuard_AReadQuantityThatLooksLikeAYearStaysRead(t *testing.T) {
	t.Parallel()

	run := runGroundedResult(t, &groundingEffects{},
		[]any{map[string]any{"pieces": 20, "weightPerUnit": 2000}},
		"Proposed. Each unit weighs 2,000 lb, so 20 of them come to 40,000 lb.",
		"Proposed.",
	)

	assert.Empty(t, run.fx.regrounded(),
		"2,000 was read, and 40,000 is 20 of it, worked in the reply")
}

func TestGroundingGuard_AFigureNoReadingSupportsIsStillCaught(t *testing.T) {
	t.Parallel()

	run := runGroundedResult(t, &groundingEffects{},
		[]any{map[string]any{"pieces": 20, "weightPerUnit": 2000}},
		"Proposed. The load weighs 41,500 lb.",
		"Proposed.",
	)

	require.Len(t, run.fx.regrounded(), 1)
	assert.Equal(t, []string{"41,500"}, run.fx.regrounded()[0].Figures)
}

func transferSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentIds": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
			},
			"deliveredFrom":               map[string]any{"type": "string"},
			"deliveredTo":                 map[string]any{"type": "string"},
			"markCompletedReadyToInvoice": map[string]any{"type": "boolean"},
		},
	}
}

// runTransfer reads a delivered shipment, files a billing_transfer of it
// holding only the shipment, then answers with the replies given.
func runTransfer(t *testing.T, replies ...string) groundingRun {
	t.Helper()

	turns := []*llmports.ChatCompletionResult{
		{
			ToolCalls: []llmports.ToolCall{{
				ID:        "call_read",
				Name:      "shipment_get",
				Arguments: map[string]any{"id": "shp_1"},
			}},
			ModelIdentifier: "test-model",
		},
		{
			ToolCalls: []llmports.ToolCall{{
				ID:   "call_file",
				Name: "billing_transfer",
				Arguments: map[string]any{
					"shipmentIds":                 []any{"shp_1"},
					"markCompletedReadyToInvoice": true,
				},
			}},
			ModelIdentifier: "test-model",
		},
	}
	for _, reply := range replies {
		turns = append(turns, runtimetest.TextTurn(reply))
	}
	completion := &scriptedCompletion{Turns: turns}
	rt := newRuntime(completion, &stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{
		queryTool("shipment_get", map[string]any{
			"id":         "shp_1",
			"proNumber":  "SEED-DET-010",
			"status":     "Completed",
			"pickup":     "Los Angeles Terminal, Los Angeles CA",
			"delivery":   "Chicago Distribution Center, Chicago IL",
			"drivers":    "John Smith on TRC-001",
			"customer":   "Acme Manufacturing",
			"deliveryAt": "2026-10-23",
		}, nil),
	}}, &stubActionRegistry{Tools: []agenttoolports.AgentTool{&agenttest.StubActionTool{
		ToolName: "billing_transfer",
		Tier:     agent.TierPropose,
		Schema:   transferSchema(),
	}}}, nil)
	fx := &groundingEffects{}
	fx.localEffects = &localEffects{s: rt, ctx: t.Context(), emit: func(kernelports.StreamEvent) {}}

	result, err := rt.Drive(rt.OpenTurn(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("shipment_get", "billing_transfer"),
		Actor:      runtimetest.Actor(),
		Input:      "acme's last delivered load that hasn't gone to billing yet, send it over",
	}), fx)
	require.NoError(t, err)

	return groundingRun{result: result, completion: completion, fx: fx}
}

// DB-244: the reply said where the load was delivered, read from the
// shipment, and the guard took "delivered to" for the transfer's deliveredTo
// filter, which the filed call does not hold, and rewrote a correct reply. A
// claim a tool result the conversation read holds is grounded.
func TestGroundingGuard_AcceptsAFactTheConversationReadBesideAFiledChange(t *testing.T) {
	t.Parallel()

	reply := "I've filed a proposal to send **SEED-DET-010** to billing for Acme " +
		"Manufacturing. It was picked up in Los Angeles on 2026-10-22 and delivered to the " +
		"**Chicago Distribution Center** on 2026-10-23, driven by John Smith on TRC-001."

	run := runTransfer(t, reply)

	assert.Equal(t, 3, run.completion.CallCount, "the reply is not asked for again")
	assert.Empty(t, run.fx.regrounded())
	assert.Equal(t, reply, run.result.Reply)
}

// What the proposal will do is still held to what was filed, even where a
// read holds the value, and a value nothing read gives is still caught.
func TestGroundingGuard_HoldsClaimsAboutTheChangeToWhatWasFiled(t *testing.T) {
	t.Parallel()

	drifted := map[string]string{
		"a read value said of the proposal": "I've filed a proposal that will send the " +
			"shipment delivered to the Chicago Distribution Center.",
		"a value nothing read gives": "Sent for review. It was delivered to the Denver " +
			"Drop Point on 2026-10-23.",
		"no value at all": "It was delivered to, and now waits on the card.",
	}
	for name, reply := range drifted {
		run := runTransfer(t, reply, "I've filed a proposal to send SEED-DET-010 to billing.")

		assert.Equal(t, 4, run.completion.CallCount, name)
		regrounded := run.fx.regrounded()
		require.Len(t, regrounded, 1, name)
		assert.Equal(t, []string{"delivered to"}, regrounded[0].Fields, name)
	}
}

// readFigures is a turn that reads one record and answers the person with the
// replies given, filing nothing.
func readFigures(
	t *testing.T,
	input string,
	read map[string]any,
	replies ...string,
) groundingRun {
	t.Helper()

	turns := []*llmports.ChatCompletionResult{runtimetest.ToolTurn("record_get", map[string]any{})}
	for _, reply := range replies {
		turns = append(turns, runtimetest.TextTurn(reply))
	}
	completion := &scriptedCompletion{Turns: turns}
	registry := &stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{
		&agenttest.StubQueryTool{ToolName: "record_get", Result: read},
	}}
	rt := newRuntime(completion, registry, &stubActionRegistry{}, nil)
	fx := &groundingEffects{}
	fx.localEffects = &localEffects{s: rt, ctx: t.Context(), emit: func(kernelports.StreamEvent) {}}

	result, err := rt.Drive(rt.OpenTurn(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("record_get"),
		Actor:      runtimetest.Actor(),
		Input:      input,
	}), fx)
	require.NoError(t, err)

	return groundingRun{result: result, completion: completion, fx: fx}
}

// DB-145, DB-230, DB-244: a figure the reply works out from what the
// conversation read, in steps the reply shows, is checked by the guard rather
// than sent back to the model; a figure nothing read or shown gives is still
// rewritten.
func TestGroundingGuard_ChecksArithmeticOnWhatTheConversationRead(t *testing.T) {
	t.Parallel()

	loads := map[string]any{"shipments": []any{
		map[string]any{"pro": "SEED-SHP-002", "charge": "1900"},
		map[string]any{"pro": "SEED-SHP-009", "charge": "3700"},
		map[string]any{"pro": "SEED-DET-009", "charge": "2700"},
		map[string]any{"pro": "SEED-DET-003", "charge": "3150"},
		map[string]any{"pro": "SEED-PAY-009", "charge": "2600"},
		map[string]any{"pro": "SEED-PAY-003", "charge": "2950"},
	}}
	loadList := "**Loads**\n\n" +
		"- **SEED-SHP-002**, $1,900, status New.\n" +
		"- **SEED-SHP-009**, $3,700, delivered 10-21.\n" +
		"- **SEED-DET-009**, $2,700, delivered 10-19.\n" +
		"- **SEED-DET-003**, $3,150, delivered 10-22.\n" +
		"- **SEED-PAY-009**, $2,600, delivered 10-23.\n" +
		"- **SEED-PAY-003**, $2,950, delivered 10-20.\n\n"
	invoices := map[string]any{"invoices": []any{
		map[string]any{"number": "SEED-INV-0007", "open": "1562.50"},
		map[string]any{"number": "SEED-INV-0006", "open": "2531.25"},
		map[string]any{"number": "SEED-INV-0005", "open": "2081.25"},
	}}
	const shortPaid = "peak sent a check for 3,200 but their invoice was 3,700. what do we " +
		"do with it"
	const formula = "make me a new pricing formula for flatbed: 3.10 a mile plus a 150 tarp fee"
	template := map[string]any{"name": "Per Mile", "exampleDistance": 400}
	billing := map[string]any{"billed": "87310.40", "stillOpen": "45662.16"}

	cases := []struct {
		name    string
		input   string
		read    map[string]any
		reply   string
		figures []string
	}{
		{
			name:  "log-a-fuel-purchase: the product of the person's own figures",
			input: "log fuel for trc-001: 112 gallons diesel at 3.89 in amarillo tx yesterday",
			read:  map[string]any{"tractor": "TRC-001", "fuelType": "Diesel"},
			reply: "TRC-001 took 112 gallons of diesel in Amarillo yesterday, $435.68 in all.",
		},
		{
			name:  "formula-asked-for-in-the-desk: 400 mi x $3.10 + $150, in two steps",
			input: formula,
			read:  template,
			reply: "For a 400-mile load, that prices at $1,240 for the miles plus $150 for " +
				"the tarp, for **$1,390** total. The $3.10 rate and $150 fee are defaults.",
		},
		{
			name:  "customer-short-paid: a payment applied oldest first",
			input: shortPaid,
			read:  invoices,
			reply: "Applied oldest first, the $3,200 pays $1,562.50 to SEED-INV-0007 and " +
				"$1,637.50 of the $2,531.25 on SEED-INV-0006, leaving $893.75 open on it.",
		},
		{
			name:  "palette-customer-by-name: the total of the reply's own list",
			input: "how is peak distributing doing",
			read:  loads,
			reply: loadList + "Total on these loads is $17,000.",
		},
		{
			name:  "the total of amounts one sentence lists",
			input: "what does peak owe",
			read:  invoices,
			reply: "Peak's three invoices, $1,562.50, $2,531.25 and $2,081.25, come to $6,175.00.",
		},
		{
			name:  "a 5.6x ratio of two figures the reply gives",
			input: "how does garza compare to vanguard",
			read:  map[string]any{"garza": "1958616.70", "vanguard": "349753.00"},
			reply: "Garza & Coleman's $1,958,616.70 open is 5.6 times Vanguard's $349,753.00, " +
				"560% of it.",
		},
		{
			name:    "an invented price",
			input:   "what does acme pay on the dallas lane",
			read:    map[string]any{"ratePerMile": "2.15", "miles": "412"},
			reply:   "Acme pays $2.15 a mile over 412 miles, a $1,450.00 linehaul.",
			figures: []string{"1,450.00"},
		},
		{
			name:    "a percentage nothing supports",
			input:   "how much of what we billed is still open",
			read:    billing,
			reply:   "Of the $87,310.40 billed, $45,662.16 is still open: about 61.4%.",
			figures: []string{"61.4"},
		},
		{
			name:    "a share whose other side the reply leaves out (DB-186)",
			input:   "how much of what we billed is still open",
			read:    billing,
			reply:   "About 52.3% of the $87,310.40 billed is still open.",
			figures: []string{"52.3"},
		},
		{
			name:    "a wrong second step",
			input:   formula,
			read:    template,
			reply:   "A 400-mile load prices at $1,240 for the miles at $3.10, plus $150, $1,490.",
			figures: []string{"1,490"},
		},
		{
			name:    "a wrong list total",
			input:   "how is peak distributing doing",
			read:    loads,
			reply:   loadList + "Total on these loads is $17,900.",
			figures: []string{"17,900"},
		},
	}
	for _, tc := range cases {
		run := readFigures(t, tc.input, tc.read, tc.reply, "I checked the record.")

		regrounded := run.fx.regrounded()
		if len(tc.figures) == 0 {
			assert.Equal(t, 2, run.completion.CallCount, tc.name)
			assert.Empty(t, regrounded, tc.name)
			assert.Equal(t, tc.reply, run.result.Reply, tc.name)
			continue
		}
		assert.Equal(t, 3, run.completion.CallCount, tc.name)
		require.Len(t, regrounded, 1, tc.name)
		assert.Equal(t, assistantports.RegroundRewrite, regrounded[0].Action, tc.name)
		assert.Equal(t, tc.figures, regrounded[0].Figures, tc.name)
	}
}

// A figure the reply works out about what it read may take two steps beside a
// filed change; one it says the change holds is held to what was read and
// filed and one step of the reply's working.
func TestGroundingGuard_HoldsAFigureSaidOfTheChangeToOneStep(t *testing.T) {
	t.Parallel()

	rows := []any{map[string]any{"ratePerMile": 3.1, "miles": 400, "tarpFee": 150}}

	read := runGroundedResult(t, &groundingEffects{}, rows,
		"Proposed the copy. At $3.10 a mile, its 400 miles cost $1,240, and with the $150 "+
			"tarp it comes to $1,390.",
	)
	assert.Equal(t, 2, read.completion.CallCount)
	assert.Empty(t, read.fx.regrounded())

	claimed := runGroundedResult(t, &groundingEffects{}, rows,
		"The proposal bills $3.10 a mile for 400 miles, $1,240, and the $150 tarp, $1,390 in "+
			"all.",
		"Proposed the copy.",
	)
	assert.Equal(t, 3, claimed.completion.CallCount)
	regrounded := claimed.fx.regrounded()
	require.Len(t, regrounded, 1)
	assert.Equal(t, []string{"1,390"}, regrounded[0].Figures)
}
