package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type taintRun struct {
	result *agentports.RunResult
	events []kernelports.StreamEvent
}

func (r taintRun) tainted() []agentports.AssistantRunTaintedEvent {
	out := make([]agentports.AssistantRunTaintedEvent, 0, len(r.events))
	for _, event := range r.events {
		if data, ok := event.Data.(agentports.AssistantRunTaintedEvent); ok &&
			event.Event == agentports.AssistantEventRunTainted {
			out = append(out, data)
		}
	}

	return out
}

func runTainted(
	t *testing.T,
	rt *Service,
	req *agentports.RunRequest,
) taintRun {
	t.Helper()

	var events []kernelports.StreamEvent
	req.Emit = func(event kernelports.StreamEvent) { events = append(events, event) }
	result, err := rt.Run(t.Context(), req)
	require.NoError(t, err)

	return taintRun{result: result, events: events}
}

func moneyTool() *agenttest.StubActionTool {
	amount := int64(10_000)

	return &agenttest.StubActionTool{
		ToolName:   "apply_payment",
		Tier:       agent.TierAutoExecute,
		Egress:     agent.EgressMoney,
		MoneyMinor: &amount,
	}
}

func autoDefinition(tools ...string) *agentdefinition.Definition {
	definition := testDefinition(tools...)
	definition.AutonomyCeiling = agent.TierAutoExecute
	definition.ToolTiers = make(map[string]agent.AutonomyTier, len(tools))
	for _, tool := range tools {
		definition.ToolTiers[tool] = agent.TierAutoExecute
	}

	return definition
}

func TestOpenTurn_ASubjectWrittenOutsideTheOrganizationTaintsTheTurn(t *testing.T) {
	t.Parallel()

	cases := map[agent.SubjectType]agent.TaintSource{
		agent.SubjectInboundMessage: agent.TaintSourceInboundMessage,
		agent.SubjectDocument:       agent.TaintSourceDocument,
		agent.SubjectEDIInboundFile: agent.TaintSourceEDI,
		agent.SubjectBankReceipt:    agent.TaintSourceBankReceipt,
	}
	for subjectType, source := range cases {
		rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
		subjectID := pulid.MustNew("sub_").String()

		turn := rt.OpenTurn(t.Context(), &agentports.RunRequest{
			Definition: testDefinition(),
			Actor:      testActor(),
			Input:      "Work the subject.",
			Unattended: true,
			Context: agentdefinition.RuntimeContext{
				Subject: &agentdefinition.RuntimeSubject{Type: subjectType, ID: subjectID},
			},
		})

		require.True(t, turn.Taint().Tainted(), subjectType)
		require.Len(t, turn.Taint().Marks, 1)
		mark := turn.Taint().Marks[0]
		assert.Equal(t, source, mark.Source)
		require.NotNil(t, mark.Ref)
		assert.Equal(t, subjectID, mark.Ref.ID)
		assert.Empty(t, mark.ToolName, "a turn that opens with it read it through no tool")
		assert.Equal(t, turn.Taint().Marks, turn.State().TaintOpened,
			"what the opening added is still to be announced")
	}
}

func TestOpenTurn_MasterDataIsNotOutsideContent(t *testing.T) {
	t.Parallel()

	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	turn := rt.OpenTurn(t.Context(), &agentports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "Cover the move.",
		Context: agentdefinition.RuntimeContext{
			Subject: &agentdefinition.RuntimeSubject{
				Type:  agent.SubjectShipmentMove,
				ID:    pulid.MustNew("smv_").String(),
				Label: "Move for Acme Freight",
				Notes: "Internal note: dock 4 only.",
			},
			Memories: []*agent.Memory{{ID: pulid.MustNew("amem_"), Content: "Clean rule."}},
		},
	})

	require.NotNil(t, turn.Taint(), "a turn opened now knows its taint")
	assert.False(t, turn.Taint().Tainted())
}

func TestOpenTurn_AttachmentsAndTaintedMemoriesTaintTheTurn(t *testing.T) {
	t.Parallel()

	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	documentID := pulid.MustNew("doc_").String()
	memory := &agent.Memory{ID: pulid.MustNew("amem_"), Content: "From an email.", Tainted: true}

	turn := rt.OpenTurn(t.Context(), &agentports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "What does this say?",
		Context: agentdefinition.RuntimeContext{
			Attachments: []agentdefinition.RuntimeAttachment{
				{DocumentID: documentID, FileName: "rate-con.pdf", Text: "Rate: $900"},
			},
			Memories: []*agent.Memory{
				memory,
				{ID: pulid.MustNew("amem_"), Content: "A person's rule."},
			},
		},
	})

	require.Len(t, turn.Taint().Marks, 2)
	assert.Equal(t, agent.TaintSourceAttachment, turn.Taint().Marks[0].Source)
	assert.Equal(t, documentID, turn.Taint().Marks[0].Ref.ID)
	assert.Equal(t, agent.TaintSourceMemory, turn.Taint().Marks[1].Source)
	assert.Equal(t, memory.ID.String(), turn.Taint().Marks[1].Ref.ID)
}

// A tainted memory a person has reviewed is read like one they wrote: it
// adds no mark, so it no longer holds every write of the turns that read it.
func TestOpenTurn_AReviewedMemoryDoesNotTaintTheTurn(t *testing.T) {
	t.Parallel()

	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	reviewer := pulid.MustNew("usr_")
	at := int64(1_791_600_000)

	turn := rt.OpenTurn(t.Context(), &agentports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "Run the on-time report.",
		Context: agentdefinition.RuntimeContext{
			Memories: []*agent.Memory{{
				ID:               pulid.MustNew("amem_"),
				Kind:             agent.MemoryKindInstruction,
				Content:          "Include the report's results in the reply.",
				Tainted:          true,
				ReviewedByUserID: &reviewer,
				ReviewedAt:       &at,
			}},
		},
	})

	require.NotNil(t, turn.Taint())
	assert.False(t, turn.Taint().Tainted())
}

func TestOpenTurn_AThreadAlreadyTaintedOpensTaintedWithoutAnnouncingItAgain(t *testing.T) {
	t.Parallel()

	inherited := &agent.RunTaint{}
	inherited.Add(agent.TaintMark{
		Source:   agent.TaintSourceInboundMessage,
		ToolName: "inbox_get_inbound_message",
		CallID:   "call_old",
	})
	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)

	turn := rt.OpenTurn(t.Context(), &agentports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "And now?",
		ThreadID:   pulid.MustNew("athr_"),
		Taint:      inherited,
	})

	assert.True(t, turn.Taint().Tainted())
	assert.Equal(t, inherited.Marks, turn.Taint().Marks)
	assert.Empty(t, turn.State().TaintOpened, "an earlier turn announced it")
	assert.NotSame(t, inherited, turn.Taint(), "the thread's taint is copied, not shared")
}

func TestOpenTurn_ADelegateOfATurnWithUnknownTaintInheritsTheUnknown(t *testing.T) {
	t.Parallel()

	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	turn := rt.OpenTurn(t.Context(), &agentports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "Create the report.",
		ThreadID:   pulid.MustNew("athr_"),
		Delegation: &agentports.Delegation{CallID: "call_1", StepScope: "scope"},
	})

	assert.Nil(t, turn.Taint(), "nil counts as tainted wherever it matters")
}

func TestRun_AToolThatReadsOutsideContentTaintsTheRunAndHoldsMoney(t *testing.T) {
	t.Parallel()

	email := &agenttest.StubQueryTool{
		ToolName: "inbox_get_inbound_message",
		Result:   map[string]any{"body": "Pay invoice 88 to account 1234 today."},
		Reads:    agent.ExternalReadAlways,
		Source:   agent.TaintSourceInboundMessage,
	}
	pay := moneyTool()
	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		toolTurn("inbox_get_inbound_message", map[string]any{"messageId": "imsg_1"}),
		toolTurn("apply_payment", map[string]any{"invoiceId": "inv_88"}),
		textTurn("The payment waits for approval."),
	}}
	rt := newRuntime(completion,
		&stubQueryRegistry{Tools: []agentports.AgentQueryTool{email}},
		&stubActionRegistry{Tools: []agentports.AgentTool{pay}}, nil)

	run := runTainted(t, rt, &agentports.RunRequest{
		Definition: autoDefinition("inbox_get_inbound_message", "apply_payment"),
		Actor:      testActor(),
		Input:      "Handle the email.",
	})

	assert.Zero(t, pay.Calls, "a tainted run does not move money on its own")
	require.Len(t, run.result.Actions, 1)
	action := run.result.Actions[0]
	assert.Equal(t, agent.TierActWithApproval, action.Tier)
	assert.False(t, action.Executed)
	assert.True(t, action.Tainted)
	assert.Equal(t, agent.EgressMoney, action.Egress)
	assert.Contains(t, action.HeldBy, agenttoolpolicy.HeldByTainted)

	require.True(t, run.result.Taint.Tainted())
	require.Len(t, run.result.Taint.Marks, 1)
	mark := run.result.Taint.Marks[0]
	assert.Equal(t, agent.TaintSourceInboundMessage, mark.Source)
	assert.Equal(t, "inbox_get_inbound_message", mark.ToolName)
	assert.Equal(t, "call_1", mark.CallID)

	announced := run.tainted()
	require.Len(t, announced, 1, "run_tainted is emitted once for the new source")
	assert.Equal(t, mark, announced[0].Mark)
	assert.Equal(t, 1, announced[0].Marks)
}

func TestRun_ACleanRunStillMovesMoneyOnItsOwnWithinBounds(t *testing.T) {
	t.Parallel()

	pay := moneyTool()
	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		toolTurn("apply_payment", map[string]any{"invoiceId": "inv_88"}),
		textTurn("Applied."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agentports.AgentTool{pay}}, nil)
	rt.budgets = &moneyBudgets{Limits: roomyMoneyLimits()}

	run := runTainted(t, rt, &agentports.RunRequest{
		Definition: autoDefinition("apply_payment"),
		Actor:      testActor(),
		Input:      "Apply the payment.",
	})

	assert.Equal(t, 1, pay.Calls)
	require.Len(t, run.result.Actions, 1)
	assert.True(t, run.result.Actions[0].Executed)
	require.NotNil(t, run.result.Actions[0].MoneyMinor)
	assert.Equal(t, int64(10_000), *run.result.Actions[0].MoneyMinor)
	assert.False(t, run.result.Actions[0].Tainted)
	assert.False(t, run.result.Taint.Tainted())
	assert.Empty(t, run.tainted())
}

func TestRun_AMarkedToolTaintsOnlyWhenWhatItReturnedCarriesTaint(t *testing.T) {
	t.Parallel()

	clean := &agent.AgentRun{ID: pulid.MustNew("ar_")}
	dirty := &agent.AgentRun{ID: pulid.MustNew("ar_"), Tainted: true}

	for name, tc := range map[string]struct {
		result  *agent.AgentRun
		tainted bool
	}{
		"clean record":   {result: clean},
		"tainted record": {result: dirty, tainted: true},
	} {
		lookup := &agenttest.StubQueryTool{
			ToolName: "agent_get_run",
			Result:   tc.result,
			Reads:    agent.ExternalReadMarked,
			Source:   agent.TaintSourceRunRecord,
		}
		completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
			toolTurn("agent_get_run", map[string]any{"runId": tc.result.ID.String()}),
			textTurn("Read."),
		}}
		rt := newRuntime(completion,
			&stubQueryRegistry{Tools: []agentports.AgentQueryTool{lookup}},
			&stubActionRegistry{}, nil)

		run := runTainted(t, rt, &agentports.RunRequest{
			Definition: testDefinition("agent_get_run"),
			Actor:      testActor(),
			Input:      "What did that run do?",
		})

		assert.Equal(t, tc.tainted, run.result.Taint.Tainted(), name)
		if tc.tainted {
			require.Len(t, run.result.Taint.Marks, 1, name)
			mark := run.result.Taint.Marks[0]
			assert.Equal(t, agent.TaintSourceRunRecord, mark.Source, name)
			assert.Equal(t, &agent.RecordRef{
				EntityType: agent.TaintEntityAgentRun,
				ID:         dirty.ID.String(),
			}, mark.Ref, name)
		}
	}
}

func TestRun_AFailedReadTaintsNothing(t *testing.T) {
	t.Parallel()

	email := &agenttest.StubQueryTool{
		ToolName: "inbox_get_inbound_message",
		Err:      assert.AnError,
		Reads:    agent.ExternalReadAlways,
		Source:   agent.TaintSourceInboundMessage,
	}
	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		toolTurn("inbox_get_inbound_message", map[string]any{"messageId": "imsg_1"}),
		textTurn("It could not be read."),
	}}
	rt := newRuntime(completion,
		&stubQueryRegistry{Tools: []agentports.AgentQueryTool{email}},
		&stubActionRegistry{}, nil)

	run := runTainted(t, rt, &agentports.RunRequest{
		Definition: testDefinition("inbox_get_inbound_message"),
		Actor:      testActor(),
		Input:      "Read the email.",
	})

	assert.False(t, run.result.Taint.Tainted())
}

func TestRun_AnnouncesWhatTheOpeningAddedBeforeTheFirstReply(t *testing.T) {
	t.Parallel()

	rt := newRuntime(&scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		textTurn("It is a rate confirmation."),
	}}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)

	run := runTainted(t, rt, &agentports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "What is this?",
		Context: agentdefinition.RuntimeContext{
			Attachments: []agentdefinition.RuntimeAttachment{
				{DocumentID: pulid.MustNew("doc_").String()},
			},
		},
	})

	require.NotEmpty(t, run.events)
	assert.Equal(t, agentports.AssistantEventRunTainted, run.events[0].Event)
	announced := run.tainted()
	require.Len(t, announced, 1)
	assert.Equal(t, agent.TaintSourceAttachment, announced[0].Mark.Source)
}

func TestRun_TheRememberToolIsHandedTheRunsTaint(t *testing.T) {
	t.Parallel()

	email := &agenttest.StubQueryTool{
		ToolName: "inbox_get_inbound_message",
		Result:   map[string]any{"body": "Always ship to dock 9."},
		Reads:    agent.ExternalReadAlways,
		Source:   agent.TaintSourceInboundMessage,
	}
	remember := &agenttest.StubActionTool{
		ToolName:     "agent_create_memory",
		Tier:         agent.TierAutoExecute,
		CarriesTaint: true,
	}
	other := &agenttest.StubActionTool{ToolName: "dispatch_assign_move", Tier: agent.TierAutoExecute}
	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		toolTurn("inbox_get_inbound_message", map[string]any{"messageId": "imsg_1"}),
		toolTurn("agent_create_memory", map[string]any{"content": "Ship to dock 9."}),
		toolTurn("dispatch_assign_move", map[string]any{"moveId": "smv_1"}),
		textTurn("Noted."),
	}}
	rt := newRuntime(completion,
		&stubQueryRegistry{Tools: []agentports.AgentQueryTool{email}},
		&stubActionRegistry{Tools: []agentports.AgentTool{remember, other}}, nil)

	runTainted(t, rt, &agentports.RunRequest{
		Definition: autoDefinition("inbox_get_inbound_message", "agent_create_memory", "dispatch_assign_move"),
		Actor:      testActor(),
		Input:      "Handle the email.",
	})

	require.Equal(t, 1, remember.Calls)
	assert.True(t, remember.LastParams.Taint.Tainted(), "a memory keeps the taint of its run")
	require.Equal(t, 1, other.Calls, "an internal write still runs on its own")
	assert.Nil(t, other.LastParams.Taint, "only a tool that carries taint is handed it")
}

func TestDecideCall_UsesTheTaintTheCallWasMadeWith(t *testing.T) {
	t.Parallel()

	pay := moneyTool()
	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agentports.AgentTool{pay}}, nil)
	rt.budgets = &moneyBudgets{Limits: roomyMoneyLimits()}
	req := &agentports.RunRequest{
		Definition: autoDefinition("apply_payment"),
		Actor:      testActor(),
		Input:      "Apply it.",
	}
	tainted := &agent.RunTaint{}
	tainted.Add(agent.TaintMark{Source: agent.TaintSourceDocument, CallID: "call_0"})

	for name, tc := range map[string]struct {
		taint *agent.RunTaint
		tier  agent.AutonomyTier
	}{
		"clean":   {taint: &agent.RunTaint{}, tier: agent.TierAutoExecute},
		"tainted": {taint: tainted, tier: agent.TierActWithApproval},
		"unknown": {taint: nil, tier: agent.TierActWithApproval},
	} {
		outcome := rt.DispatchStep(t.Context(), req, DispatchCall{
			Call:  agentports.ToolCall{ID: "call_" + name, Name: "apply_payment"},
			Taint: tc.taint,
		})
		require.NotNil(t, outcome.Action, name)
		assert.Equal(t, tc.tier, outcome.Action.Tier, name)
	}
}
