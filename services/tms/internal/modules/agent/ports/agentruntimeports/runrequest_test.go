package agentruntimeports_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmessage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/stretchr/testify/assert"
)

/*
A dispatch proposal raised by an event-started run read "Asked to assign
dispatch move in reply to: An event started this run: …", quoting the
runtime's own instruction as if a person had asked for the assignment. Only
a turn a person typed into carries a request worth quoting.
*/
func TestAsk_IsOnlyWhatAPersonTyped(t *testing.T) {
	t.Parallel()

	background := agentmessage.BackgroundRunInput(
		agent.RunTriggerEvent,
		agenttrigger.EventKind("shipment_move.coverage_at_risk"),
		nil,
	)
	cases := map[string]struct {
		req  agentruntimeports.RunRequest
		want string
	}{
		"a chat turn": {
			req:  agentruntimeports.RunRequest{Input: "Cover S26110032638290 with Ana"},
			want: "Cover S26110032638290 with Ana",
		},
		"a run an event started": {
			req:  agentruntimeports.RunRequest{Input: background, Unattended: true},
			want: "",
		},
		"a task another agent handed over": {
			req: agentruntimeports.RunRequest{
				Input:      "Find a carrier for the Dallas load",
				Delegation: &agentruntimeports.Delegation{},
			},
			want: "",
		},
	}
	cases["a turn Trenova opened after a decision"] = struct {
		req  agentruntimeports.RunRequest
		want string
	}{
		req: agentruntimeports.RunRequest{
			Input:          "Approved billing_approve_queue_item, and it ran.\nDecision on proposal ap_1",
			TrenovaWritten: true,
			Asked:          "Watch this and post it once I'm done",
		},
		want: "Watch this and post it once I'm done",
	}
	cases["a turn Trenova opened before the person typed anything"] = struct {
		req  agentruntimeports.RunRequest
		want string
	}{
		req:  agentruntimeports.RunRequest{Input: "A wait ended", TrenovaWritten: true},
		want: "",
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, tc.req.Ask())
		})
	}
}

func TestLastAsked_IsTheLastMessageThePersonTyped(t *testing.T) {
	t.Parallel()

	history := []conversation.Message{
		{Role: conversation.RoleUser, Kind: conversation.MessageKindMessage,
			Content: "Watch this and post it once I'm done"},
		{Role: conversation.RoleAssistant, Content: "I'm watching it."},
		{Role: conversation.RoleUser, Kind: conversation.MessageKindDecisionNote,
			Content: "Approved billing_approve_queue_item"},
		{Role: conversation.RoleUser, Kind: conversation.MessageKindWaitNote,
			Content: "The wait ended"},
	}

	assert.Equal(t, "Watch this and post it once I'm done", agentruntimeports.LastAsked(history))
	assert.Empty(t, agentruntimeports.LastAsked(nil))
}
