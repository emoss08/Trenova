package replycheck

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type draftTurn struct {
	boardTurn
}

func (d *draftTurn) Offers(name string) bool {
	return name == showAnswerTool || name == publishArtifactTool
}

const lapsedCredentials = `{"items":[` +
	`{"id":"wcred_01JCRED00000000000000001","type":"DOT Medical Card","status":"Expired",` +
	`"worker":{"id":"wrk_01JJANE000000000000000001","name":"Jane Doe"}},` +
	`{"id":"wcred_01JCRED00000000000000002","type":"DOT Physical","status":"Expired",` +
	`"worker":{"id":"wrk_01JJANE000000000000000001","name":"Jane Doe"}}],"total":2}`

func reminderTurn(asked string) *draftTurn {
	return &draftTurn{boardTurn{
		req: agentruntimeports.RunRequest{Input: asked},
		messages: []llmports.Message{
			{Role: llmports.RoleUser, Content: asked},
			{
				Role: llmports.RoleTool, ToolName: "worker_list_credentials", ToolCallID: "call_1",
				Content: toolresult.Fence("worker_list_credentials", lapsedCredentials),
			},
		},
	}}
}

const stepTwo = "Draft a short renewal reminder to each person whose credential on this board " +
	"has lapsed or runs out soon, as a document I can read and send myself."

func TestAskForDraft_SendsBackAReplyThatWritesOutTheMessageToSend(t *testing.T) {
	t.Parallel()

	for name, reply := range map[string]string{
		"the action's turn, a drafted letter": "The board I showed lists only Jane Doe's two " +
			"credentials, so there is one person to draft for.\n\n**Draft: renewal reminder to " +
			"Jane Doe**\n\nSubject: Action needed: DOT Medical Card and DOT Physical expired\n\n" +
			"Hi Jane,\n\nYour DOT Medical Card and DOT Physical both expired on November 15, " +
			"2026. Please book your medical exam as soon as you can.\n\nThanks,\n[Your name]\n" +
			"Trenova Logistics\n\nNothing has been sent.",
		"one quoted line": "Jane Doe's DOT Medical Card expired on 2026-11-15. Here is the text " +
			"you can send her:\n\n> Hi Jane, your DOT Medical Card expired on November 15, 2026, " +
			"and you can't be dispatched until it is renewed. Thanks, Trenova Logistics",
	} {
		state := &BoardAskState{}
		back, ask := AskAgain(reminderTurn(stepTwo), draftOf(reply), state)

		require.True(t, ask, name)
		assert.Equal(t, GuardDraft, back.Guard, name)
		assert.True(t, state.Drafted, name)
		regrounded, ok := back.Events[1].Data.(assistantstreamports.AssistantReplyRegroundedEvent)
		require.True(t, ok)
		assert.Equal(t, assistantstreamports.RegroundPublishDraft, regrounded.Action)
		correction := back.Messages[1].Content
		assert.Contains(t, correction, `(it opens "`, name)
		assert.Contains(t, correction, "call publish_artifact with the message as its body")
		assert.Contains(t, correction, "Nothing is sent: the person sends it themselves.")

		_, again := AskAgain(reminderTurn(stepTwo), draftOf(reply), state)
		assert.False(t, again, "%s: the draft is asked for once a turn", name)
	}
}

func TestAskForDraft_LeavesAReplyThatHoldsNoDraftedMessage(t *testing.T) {
	t.Parallel()

	quoted := reminderTurn("what did acme say about the POD?")
	quoted.messages = append(quoted.messages, llmports.Message{
		Role: llmports.RoleTool, ToolName: "email_get", ToolCallID: "call_2",
		Content: toolresult.Fence("email_get", `{"body":"Hi team, we need the POD for `+
			`SEED-SHP-008 by Friday or we hold payment. Thanks, Dana"}`),
	})
	published := reminderTurn(stepTwo)
	published.artifacts = []agentruntimeports.ShownArtifact{{
		ID: pulid.ID("art_01JDRAFT0000000000000001"), Kind: string(assistantartifact.KindDocument),
	}}

	for name, test := range map[string]struct {
		turn  *draftTurn
		reply string
	}{
		"a message the turn read, quoted": {quoted, "Acme wrote:\n\n> Hi team, we need the POD " +
			"for SEED-SHP-008 by Friday or we hold payment. Thanks, Dana"},
		"a greeting to the person": {reminderTurn("which credentials lapsed?"), "Hi Eric, two " +
			"credentials have lapsed, both Jane Doe's, on Nov 15."},
		"after the draft was published": {published, "I drafted it.\n\n> Hi Jane, your card " +
			"expired. Thanks"},
		"a reply about a message": {reminderTurn(stepTwo), "The reminder should ask Jane to " +
			"book her exam and send the new card."},
	} {
		_, ask := AskForDraft(test.turn, draftOf(test.reply), &BoardAskState{})
		assert.False(t, ask, name)
	}
}

func TestAskAgain_AsksForTheDraftBeforeABoard(t *testing.T) {
	t.Parallel()

	turn := reminderTurn(stepTwo)
	turn.messages[1].Content = fencedResult("shipment_list", twoLateLoads)
	reply := "SEED-SHP-008 and SEED-SHP-010 are late. Here is the note to send:\n\n> Hi Leon, " +
		"SEED-SHP-008 is running behind.\n> Thanks"
	state := &BoardAskState{}

	first, ask := AskAgain(turn, draftOf(reply), state)
	require.True(t, ask)
	assert.Equal(t, GuardDraft, first.Guard)

	second, ask := AskAgain(turn, draftOf(reply), state)
	require.True(t, ask, "the board is still asked for once")
	assert.Equal(t, GuardBoard, second.Guard)
}

func TestAskForDraft_AsksForTheBoardTooWhenThePersonAskedAQuestion(t *testing.T) {
	t.Parallel()

	letter := "Here is the reminder:\n\n> Hi Jane, your DOT Medical Card expired on November " +
		"15, 2026. Thanks, Trenova Logistics"
	asked := reminderTurn("whose credentials have already lapsed? write a short reminder I can " +
		"send them to renew, and show me what lapsed")

	back, ask := AskForDraft(asked, draftOf(letter), &BoardAskState{})
	require.True(t, ask)
	assert.Contains(t, back.Messages[1].Content, "answer that once the draft's result is back, "+
		"with show_answer, linking the draft in its words")

	back, ask = AskForDraft(reminderTurn(stepTwo), draftOf(letter), &BoardAskState{})
	require.True(t, ask)
	assert.NotContains(t, back.Messages[1].Content, "show_answer",
		"a request that asks no question is only drafted")
}
