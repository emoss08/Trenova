package agentruntime

import (
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type groundingEffects struct {
	*localEffects

	supports bool
	// joined is an execution started before the figures were read apart.
	joined             bool
	askedFiguresAsRead bool
	// offersAreClaims is an execution started before an offer was read as
	// no claim.
	offersAreClaims bool
	// unworked is an execution started before a reply's working counted.
	unworked bool
	events   []kernelports.StreamEvent
}

func (fx *groundingEffects) Emit(event kernelports.StreamEvent) {
	fx.events = append(fx.events, event)
}

func (fx *groundingEffects) Supports(change string) bool {
	switch change {
	case changeGroundingGuard, changeGroundingReadAmounts:
		return fx.supports
	case changeGroundingFiguresAsRead:
		fx.askedFiguresAsRead = true
		return !fx.joined
	case changeGroundingOffers:
		return !fx.offersAreClaims
	case changeGroundingWorkedFigures:
		return !fx.unworked
	}

	return true
}

func (fx *groundingEffects) regrounded() []agentports.AssistantReplyRegroundedEvent {
	out := make([]agentports.AssistantReplyRegroundedEvent, 0, 2)
	for _, event := range fx.events {
		if event.Event != agentports.AssistantEventReplyRegrounded {
			continue
		}
		if data, ok := event.Data.(agentports.AssistantReplyRegroundedEvent); ok {
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
	result     *agentports.RunResult
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

	return runGroundedWith(t, &groundingEffects{supports: supports}, filed, replies...)
}

func runGroundedWith(
	t *testing.T,
	fx *groundingEffects,
	filed map[string]any,
	replies ...string,
) groundingRun {
	t.Helper()

	turns := []*agentports.ChatCompletionResult{toolTurn("create_shipment", filed)}
	for _, reply := range replies {
		turns = append(turns, textTurn(reply))
	}
	completion := &scriptedCompletion{Turns: turns}
	tool := &agenttest.StubActionTool{
		ToolName: "create_shipment",
		Tier:     agent.TierPropose,
		Schema:   groundedShipmentSchema(),
	}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{
		Tools: []agentports.AgentTool{tool},
	}, nil)
	req := &agentports.RunRequest{
		Definition: testDefinition("create_shipment"),
		Actor:      testActor(),
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

func lastUserMessage(req *agentports.ChatCompletionRequest) string {
	for idx := len(req.Messages) - 1; idx >= 0; idx-- {
		if req.Messages[idx].Role == agentports.RoleUser {
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
	assert.Equal(t, agentports.RegroundRewrite, regrounded[0].Action)
	assert.Equal(t, []string{"commodities"}, regrounded[0].Fields)
	assert.Contains(t, eventNames(run.fx.events), agentports.AssistantEventRetrying,
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
	assert.Equal(t, agentports.RegroundRewrite, regrounded[0].Action)
	assert.Equal(t, agentports.RegroundNote, regrounded[1].Action)
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

func TestGroundingGuard_AConversationThatReadNothingIsNotChecked(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		textTurn("Detention usually runs $50 to $75 an hour after two free hours."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fx := &groundingEffects{supports: true}
	fx.localEffects = &localEffects{s: rt, ctx: t.Context(), emit: func(kernelports.StreamEvent) {}}

	_, err := rt.Drive(rt.OpenTurn(t.Context(), &agentports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
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
	supports bool,
	tools []string,
	replies ...string,
) groundingRun {
	t.Helper()

	return readGroundedWith(t, &groundingEffects{supports: supports}, tools, replies...)
}

func readGroundedWith(
	t *testing.T,
	fx *groundingEffects,
	tools []string,
	replies ...string,
) groundingRun {
	t.Helper()

	turns := []*agentports.ChatCompletionResult{toolTurn("get_report_run", map[string]any{})}
	for _, reply := range replies {
		turns = append(turns, textTurn(reply))
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
	fx.localEffects = &localEffects{s: rt, ctx: t.Context(), emit: func(kernelports.StreamEvent) {}}

	result, err := rt.Drive(rt.OpenTurn(t.Context(), &agentports.RunRequest{
		Definition: testDefinition(tools...),
		Actor:      testActor(),
		Input:      "how much of what we billed in the last 30 days is still open?",
	}), fx)
	require.NoError(t, err)

	return groundingRun{result: result, completion: completion, fx: fx}
}

// DB-186: an answer that filed nothing divided one total by another in its head
// and stated a share no tool returned.
func TestGroundingGuard_RewritesAShareNothingRead(t *testing.T) {
	t.Parallel()

	run := readGrounded(t, true, []string{"get_report_run", "analyze_data"},
		"About 52.3% of the $87,310.40 billed is still open.",
		"Of the $87,310.40 billed, $45,662.16 is still open.",
	)

	require.Equal(t, 3, run.completion.CallCount, "the reply is asked for once more")
	correction := lastUserMessage(run.completion.Requests[2])
	assert.True(t, strings.HasPrefix(correction, amountsCorrectionLead))
	assert.Contains(t, correction, "52.3")
	assert.Contains(t, correction, "analyze_data")
	assert.Equal(t, "Of the $87,310.40 billed, $45,662.16 is still open.", run.result.Reply)
	assert.True(t, strings.HasSuffix(correction, unseenCorrection),
		"a rewrite answers the person, not the correction")

	regrounded := run.fx.regrounded()
	require.Len(t, regrounded, 1)
	assert.Equal(t, agentports.RegroundRewrite, regrounded[0].Action)
	assert.Equal(t, []string{"52.3"}, regrounded[0].Figures)
}

func TestGroundingGuard_WithoutAnalysisAShareIsLeftOut(t *testing.T) {
	t.Parallel()

	run := readGrounded(t, true, []string{"get_report_run"},
		"About 52.3% is still open.",
		"Of the $87,310.40 billed, $45,662.16 is still open.",
	)

	require.Equal(t, 3, run.completion.CallCount)
	correction := lastUserMessage(run.completion.Requests[2])
	assert.NotContains(t, correction, "analyze_data")
	assert.Contains(t, correction, "leave it out")
}

func TestGroundingGuard_NamesTheFiguresWhenTheRewriteStillStatesThem(t *testing.T) {
	t.Parallel()

	run := readGrounded(t, true, []string{"get_report_run"},
		"About 52.3% is still open.",
		"Roughly 52.3% is still open.",
	)

	require.Equal(t, 3, run.completion.CallCount, "the reply is asked for once more, only once")
	assert.True(t, strings.HasPrefix(run.result.Reply, "Roughly 52.3% is still open."))
	assert.Contains(t, run.result.Reply, "Check the figures in this reply")
	assert.Contains(t, run.result.Reply, "52.3 came from no record")

	regrounded := run.fx.regrounded()
	require.Len(t, regrounded, 2)
	assert.Equal(t, agentports.RegroundNote, regrounded[1].Action)
}

func TestGroundingGuard_AcceptsWhatTheConversationRead(t *testing.T) {
	t.Parallel()

	run := readGrounded(t, true, []string{"get_report_run"},
		"Of about $87.3k billed, $45,662.16 is still open, and 91.4% delivered on time "+
			"across 31 days.",
	)

	assert.Equal(t, 2, run.completion.CallCount)
	assert.Empty(t, run.fx.regrounded())
}

func TestGroundingGuard_LeavesAReadAnswerStartedBeforeItAlone(t *testing.T) {
	t.Parallel()

	run := readGrounded(t, false, []string{"get_report_run"}, "About 52.3% is still open.")

	assert.Equal(t, 2, run.completion.CallCount)
	assert.Empty(t, run.fx.regrounded())
	assert.Equal(t, "About 52.3% is still open.", run.result.Reply)
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

func TestFiledLabels_MarkANestedOneWordCollection(t *testing.T) {
	t.Parallel()

	labels := schemaLabels(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"requirements": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"codes": map[string]any{
							"type":  "array",
							"items": map[string]any{"type": "string"},
						},
						"allowedCodeLists": map[string]any{
							"type":  "array",
							"items": map[string]any{"type": "string"},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, labels, schemaLabel{path: "requirements", words: "requirements"})
	assert.Contains(t, labels,
		schemaLabel{path: "requirements[].allowedCodeLists", words: "allowed code lists"})
	assert.Contains(t, labels,
		schemaLabel{path: "requirements[].codes", words: "codes", nestedWord: true})
}

func TestGroundingFinding_DropsOnlyWordsNoTopLevelLabelClaims(t *testing.T) {
	t.Parallel()

	finding := groundingFinding{
		fields: []string{"codes", "commodities"},
		tools: map[string][]string{
			"codes":       {"apply_edi_connection_draft"},
			"commodities": {"create_shipment"},
		},
		nested: []string{"codes"},
	}
	finding.dropNestedWords()

	assert.Equal(t, []string{"commodities"}, finding.fields)
	assert.NotContains(t, finding.tools, "codes")
	assert.False(t, finding.empty())
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

	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		toolTurn("get_report_run", map[string]any{}),
		textTurn("The confirmation totals $3,610.00: $3,200.00 line haul and $410.00 fuel."),
	}}
	registry := &stubQueryRegistry{Tools: []agentports.AgentQueryTool{
		&agenttest.StubQueryTool{ToolName: "get_report_run", Result: map[string]any{}},
	}}
	rt := newRuntime(completion, registry, &stubActionRegistry{}, nil)
	fx := &groundingEffects{supports: true}
	fx.localEffects = &localEffects{s: rt, ctx: t.Context(), emit: func(kernelports.StreamEvent) {}}

	turn := rt.OpenTurn(t.Context(), &agentports.RunRequest{
		Definition: testDefinition("get_report_run"),
		Actor:      testActor(),
		Input:      "what's the total rate on it?",
	})
	turn.messages[len(turn.messages)-1].Files = []agentports.MessageFile{{FileName: "ratecon.pdf"}}
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

	run := runGroundedResult(t, &groundingEffects{supports: true},
		[]any{[]any{"Riverside", 39, 12937.5, 0.708}},
		"Proposed. Riverside cost $12,937.50, 70.8% of the total.",
	)

	assert.Equal(t, 2, run.completion.CallCount)
	assert.Empty(t, run.fx.regrounded())
	assert.True(t, run.fx.askedFiguresAsRead, "the two readings disagree, so the workflow is asked")
}

func TestGroundingGuard_AnExecutionThatReadFiguresJoinedKeepsDoingSo(t *testing.T) {
	t.Parallel()

	run := runGroundedResult(t, &groundingEffects{supports: true, joined: true},
		[]any{[]any{"Riverside", 39, 12937.5, 0.708}},
		"Proposed. Riverside cost $12,937.50.",
		"Proposed.",
	)

	assert.Equal(t, 3, run.completion.CallCount)
	require.Len(t, run.fx.regrounded(), 1)
}

func TestGroundingGuard_AgreeingReadingsDoNotAskTheWorkflow(t *testing.T) {
	t.Parallel()

	run := runGroundedResult(t, &groundingEffects{supports: true},
		[]any{"12937.5"},
		"Proposed. Riverside cost $12,937.50.",
	)

	assert.Empty(t, run.fx.regrounded())
	assert.False(t, run.fx.askedFiguresAsRead)
}

// runGroundedResult files a create_shipment whose result carries rows, then
// answers with the replies given.
func runGroundedResult(
	t *testing.T,
	fx *groundingEffects,
	rows []any,
	replies ...string,
) groundingRun {
	t.Helper()

	turns := []*agentports.ChatCompletionResult{
		toolTurn("create_shipment", map[string]any{"customerId": "cus_1"}),
	}
	for _, reply := range replies {
		turns = append(turns, textTurn(reply))
	}
	completion := &scriptedCompletion{Turns: turns}
	tool := &agenttest.StubActionTool{
		ToolName: "create_shipment",
		Tier:     agent.TierPropose,
		Schema:   groundedShipmentSchema(),
	}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{
		Tools: []agentports.AgentTool{tool},
	}, nil)
	fx.localEffects = &localEffects{s: rt, ctx: t.Context(), emit: func(kernelports.StreamEvent) {}}

	turn := rt.OpenTurn(t.Context(), &agentports.RunRequest{
		Definition: testDefinition("create_shipment"),
		Actor:      testActor(),
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

	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		toolTurn("get_invoices", map[string]any{}),
		textTurn("The $3,200 check leaves $500 open on the $3,700 invoice; a $517.40 fee would not."),
		textTurn("The $3,200 check leaves $500 open on the $3,700 invoice."),
	}}
	registry := &stubQueryRegistry{Tools: []agentports.AgentQueryTool{
		&agenttest.StubQueryTool{ToolName: "get_invoices", Result: map[string]any{"total": "3700"}},
	}}
	rt := newRuntime(completion, registry, &stubActionRegistry{}, nil)
	fx := &groundingEffects{supports: true}
	fx.localEffects = &localEffects{s: rt, ctx: t.Context(), emit: func(kernelports.StreamEvent) {}}

	_, err := rt.Drive(rt.OpenTurn(t.Context(), &agentports.RunRequest{
		Definition: testDefinition("get_invoices"),
		Actor:      testActor(),
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

	run := runGroundedWith(t, &groundingEffects{supports: true},
		map[string]any{"customerId": "cus_1"},
		"I proposed the copy of PRO-100. If you have the packing list, I can add the "+
			"commodities before you approve it.",
	)

	assert.Equal(t, 2, run.completion.CallCount)
	assert.Empty(t, run.fx.regrounded())
}

func TestGroundingGuard_AnOfferBesideAClaimStillChecksTheClaim(t *testing.T) {
	t.Parallel()

	run := runGroundedWith(t, &groundingEffects{supports: true},
		map[string]any{"customerId": "cus_1"},
		"I proposed the copy with its commodities. I can add more commodities if you need them.",
		"I proposed the copy of PRO-100.",
	)

	assert.Equal(t, 3, run.completion.CallCount)
	require.Len(t, run.fx.regrounded(), 1)
	assert.Equal(t, []string{"commodities"}, run.fx.regrounded()[0].Fields)
}

func TestGroundingGuard_AnExecutionThatReadOffersAsClaimsKeepsDoingSo(t *testing.T) {
	t.Parallel()

	run := runGroundedWith(t, &groundingEffects{supports: true, offersAreClaims: true},
		map[string]any{"customerId": "cus_1"},
		"I proposed the copy. I can add the commodities before you approve it.",
		"I proposed the copy of PRO-100.",
	)

	assert.Equal(t, 3, run.completion.CallCount)
}

// A share whose two sides the reply gives is shown working: the guard checks
// the division instead of sending the agent to analyze_data for it.
func TestGroundingGuard_AcceptsArithmeticOnFiguresTheReplyGives(t *testing.T) {
	t.Parallel()

	run := readGrounded(t, true, []string{"get_report_run"},
		"Of the $87,310.40 billed, $45,662.16 is still open: 52.3%, leaving $41,648.24 paid.",
	)

	assert.Equal(t, 2, run.completion.CallCount)
	assert.Empty(t, run.fx.regrounded())
}

func TestGroundingGuard_WrongArithmeticOnGivenFiguresIsStillRewritten(t *testing.T) {
	t.Parallel()

	run := readGrounded(t, true, []string{"get_report_run"},
		"Of the $87,310.40 billed, $45,662.16 is still open: 61.4%.",
		"Of the $87,310.40 billed, $45,662.16 is still open.",
	)

	require.Len(t, run.fx.regrounded(), 1)
	assert.Equal(t, []string{"61.4"}, run.fx.regrounded()[0].Figures)
}

func TestGroundingGuard_AnExecutionThatIgnoredWorkingKeepsDoingSo(t *testing.T) {
	t.Parallel()

	run := readGroundedWith(t, &groundingEffects{supports: true, unworked: true},
		[]string{"get_report_run"},
		"Of the $87,310.40 billed, $45,662.16 is still open: 52.3%.",
		"Of the $87,310.40 billed, $45,662.16 is still open.",
	)

	assert.Equal(t, 3, run.completion.CallCount)
}

// Told 112 gallons at $3.89, "$435.68" is the person's own two figures
// multiplied, whether or not the reply repeats them.
func TestGroundingGuard_AProductOfThePersonsFiguresIsStated(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		toolTurn("find_in_trenova", map[string]any{}),
		textTurn("Key it in at Fuel purchases; the total paid should be about $435.68."),
	}}
	registry := &stubQueryRegistry{Tools: []agentports.AgentQueryTool{
		&agenttest.StubQueryTool{ToolName: "find_in_trenova", Result: map[string]any{}},
	}}
	rt := newRuntime(completion, registry, &stubActionRegistry{}, nil)
	fx := &groundingEffects{supports: true}
	fx.localEffects = &localEffects{s: rt, ctx: t.Context(), emit: func(kernelports.StreamEvent) {}}

	_, err := rt.Drive(rt.OpenTurn(t.Context(), &agentports.RunRequest{
		Definition: testDefinition("find_in_trenova"),
		Actor:      testActor(),
		Input:      "log 112 gallons of diesel for TRC-001 bought in Texas yesterday at $3.89",
	}), fx)
	require.NoError(t, err)

	assert.Equal(t, 2, completion.CallCount)
	assert.Empty(t, fx.regrounded())
}

func TestGroundingGuard_AFiledReplysWorkingCounts(t *testing.T) {
	t.Parallel()

	run := runGroundedResult(t, &groundingEffects{supports: true},
		[]any{"1121.36", "898.80"},
		"Proposed at $1,121.36 in total: $898.80 linehaul and $222.56 in other charges.",
	)

	assert.Equal(t, 2, run.completion.CallCount)
	assert.Empty(t, run.fx.regrounded())
}
