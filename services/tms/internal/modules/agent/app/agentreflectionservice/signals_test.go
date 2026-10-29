package agentreflectionservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentreflection"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/stretchr/testify/assert"
)

func person(text string) conversation.Message {
	return conversation.Message{
		Role:    conversation.RoleUser,
		Kind:    conversation.MessageKindMessage,
		Content: text,
	}
}

func reply(text string) conversation.Message {
	return conversation.Message{Role: conversation.RoleAssistant, Content: text}
}

func toolResult(name string, failed bool, verdict string) conversation.Message {
	return conversation.Message{
		Role:        conversation.RoleTool,
		ToolName:    name,
		ToolFailed:  failed,
		ToolVerdict: verdict,
	}
}

func signalKinds(signals agentreflection.ReflectionSignals) []agentreflection.ReflectionSignalKind {
	kinds := make([]agentreflection.ReflectionSignalKind, 0, len(signals))
	for _, signal := range signals {
		kinds = append(kinds, signal.Kind)
	}

	return kinds
}

func TestReadSignals_AQuietSuccessfulTurnHasNothingToLearn(t *testing.T) {
	t.Parallel()

	signals := ReadSignals(&Window{
		Messages: []conversation.Message{
			person("How many loads are late today?"),
			toolResult("shipment_list", false, aitrace.OutcomeRan),
			reply("Three are late."),
		},
		HasPerson: true,
	})

	assert.Empty(t, signals)
}

func TestReadSignals_AToolThatWorkedAfterFailingIsRecoveredNotFailed(t *testing.T) {
	t.Parallel()

	signals := ReadSignals(&Window{
		Messages: []conversation.Message{
			toolResult("dispatch_assign_move", true, aitrace.OutcomeFailed),
			toolResult("dispatch_assign_move", false, aitrace.OutcomeInvalid),
			toolResult("dispatch_assign_move", false, aitrace.OutcomeRan),
			toolResult("tender_load", true, aitrace.OutcomeFailed),
		},
	})

	assert.Equal(t, []agentreflection.ReflectionSignalKind{
		agentreflection.ReflectionSignalToolRecovered,
		agentreflection.ReflectionSignalToolFailed,
	}, signalKinds(signals))
	assert.Equal(t, "dispatch_assign_move", signals[0].Detail)
	assert.Equal(t, "tender_load", signals[1].Detail)
}

func TestReadSignals_RefusalsAreNotLessons(t *testing.T) {
	t.Parallel()

	signals := ReadSignals(&Window{
		Messages: []conversation.Message{
			toolResult("invoice_send", true, aitrace.OutcomeDenied),
			toolResult("invoice_send", true, aitrace.OutcomeOverBudget),
			toolResult("invoice_send", true, aitrace.OutcomeDuplicate),
		},
	})

	assert.Empty(t, signals)
}

func TestReadSignals_ACorrectionCountsOnlyAfterTheAgentAnswered(t *testing.T) {
	t.Parallel()

	opening := ReadSignals(&Window{
		Messages:  []conversation.Message{person("No, the other customer")},
		HasPerson: true,
	})
	assert.Empty(t, opening)

	afterReply := ReadSignals(&Window{
		Messages: []conversation.Message{
			person("Show me Acme's invoices"),
			reply("Here are Acme Freight's invoices."),
			person("No, I meant Acme Foods"),
		},
		HasPerson: true,
	})
	assert.Equal(
		t,
		[]agentreflection.ReflectionSignalKind{agentreflection.ReflectionSignalPersonCorrected},
		signalKinds(afterReply),
	)

	continued := ReadSignals(&Window{
		Messages:  []conversation.Message{person("That's not what I asked for")},
		HasPerson: true,
		Continues: true,
	})
	assert.Equal(
		t,
		[]agentreflection.ReflectionSignalKind{agentreflection.ReflectionSignalPersonCorrected},
		signalKinds(continued),
	)
}

func TestReadSignals_AStandingRequestIsHeardWithoutACorrection(t *testing.T) {
	t.Parallel()

	signals := ReadSignals(&Window{
		Messages:  []conversation.Message{person("From now on, copy dispatch on these emails")},
		HasPerson: true,
	})

	assert.Equal(
		t,
		[]agentreflection.ReflectionSignalKind{agentreflection.ReflectionSignalStandingRequest},
		signalKinds(signals),
	)
}

func TestReadSignals_ARunWithNobodyInItReadsNoPersonSignals(t *testing.T) {
	t.Parallel()

	signals := ReadSignals(&Window{
		Messages: []conversation.Message{
			reply("Checked the load."),
			person("No, from now on always do it differently"),
		},
	})

	assert.Empty(t, signals)
}

func TestReadSignals_DecisionsFeedbackAndLongTasks(t *testing.T) {
	t.Parallel()

	messages := make([]conversation.Message, 0, LongTaskToolCalls)
	for range LongTaskToolCalls {
		messages = append(messages, toolResult("shipment_get", false, aitrace.OutcomeRan))
	}

	signals := ReadSignals(&Window{
		Messages: messages,
		Proposals: []DecidedProposal{
			{ToolName: "dispatch_assign_move", Outcome: agenttooltrust.TrustOutcomeModified},
			{ToolName: "dispatch_assign_move", Outcome: agenttooltrust.TrustOutcomeModified},
			{
				ToolName: "invoice_send",
				Outcome:  agenttooltrust.TrustOutcomeRejected,
				Reason:   "Wrong bill-to",
			},
			{ToolName: "update_stop", Outcome: agenttooltrust.TrustOutcomeApproved},
		},
		NegativeRatings: 2,
		HasPerson:       true,
	})

	assert.Equal(t, []agentreflection.ReflectionSignalKind{
		agentreflection.ReflectionSignalProposalModified,
		agentreflection.ReflectionSignalProposalRejected,
		agentreflection.ReflectionSignalNegativeFeedback,
		agentreflection.ReflectionSignalLongTask,
	}, signalKinds(signals))
	assert.Equal(t, 1, signals[0].Count)
	assert.Equal(t, "dispatch_assign_move", signals[0].Detail)
	assert.Equal(t, 2, signals[2].Count)
}

func TestReadSignals_ALongRunWithNobodyInItIsNoSignal(t *testing.T) {
	t.Parallel()

	messages := make([]conversation.Message, 0, LongTaskToolCalls)
	for range LongTaskToolCalls {
		messages = append(messages, toolResult("shipment_get", false, aitrace.OutcomeRan))
	}

	assert.Empty(t, ReadSignals(&Window{Messages: messages}),
		"a run that only worked has nothing a person showed it to learn from")
}
