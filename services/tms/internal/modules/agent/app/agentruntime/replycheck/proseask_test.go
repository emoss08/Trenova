package replycheck

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type askTurn struct {
	boardTurn
	noAsk bool
}

func (a *askTurn) Offers(name string) bool {
	return name == askUserTool && !a.noAsk
}

func plannedTurn(req agentruntimeports.RunRequest) *askTurn {
	return &askTurn{boardTurn: boardTurn{
		req:      req,
		messages: []llmports.Message{{Role: llmports.RoleUser, Content: req.Input}},
	}}
}

func TestAskForQuestion_SendsBackAReplyThatEndsOnAQuestion(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		reply    string
		question string
	}{
		"want me to go ahead": {
			reply: "There are 274 shipments with status New or Delayed.\n\nThe next step is a " +
				"coverage check on the Delayed shipments. Want me to go ahead?",
			question: "Want me to go ahead?",
		},
		"would you like, in bold": {
			reply: "Batch 1 is planned: 8 picks at $12,105.\n\n**Would you like me to move on " +
				"to batch 2 next?**",
			question: "Would you like me to move on to batch 2 next?",
		},
		"either or": {
			reply: "Where things stand: batch 1 is not planned.\n\nWould you like me to keep " +
				"going with batch 1 once I've counted its ids, or stop here?",
			question: "Would you like me to keep going with batch 1 once I've counted its ids, " +
				"or stop here?",
		},
		"let me know which": {
			reply: "Each batch is a proposal.\n\nLet me know which batches to approve, or " +
				"which lines to drop.",
			question: "Let me know which batches to approve, or which lines to drop.",
		},
		"a question then a closing line": {
			reply: "Nothing is assigned yet.\n\nShould I send the 8 assignments to dispatch? " +
				"Nothing goes out until you approve.",
			question: "Should I send the 8 assignments to dispatch?",
		},
	} {
		state := &BoardAskState{}
		back, ask := AskAgain(
			plannedTurn(agentruntimeports.RunRequest{Input: "plan the shipments"}),
			draftOf(tc.reply),
			state,
		)

		require.True(t, ask, name)
		assert.Equal(t, GuardQuestion, back.Guard, name)
		assert.True(t, state.Questioned, name)
		regrounded, ok := back.Events[1].Data.(assistantstreamports.AssistantReplyRegroundedEvent)
		require.True(t, ok, name)
		assert.Equal(t, assistantstreamports.RegroundAskUser, regrounded.Action, name)
		correction := back.Messages[1].Content
		assert.Contains(t, correction, `("`+tc.question+`")`, name)
		assert.Contains(t, correction, "call ask_user", name)
		assert.Contains(t, correction, "questions", name)

		_, again := AskAgain(
			plannedTurn(agentruntimeports.RunRequest{Input: "plan the shipments"}),
			draftOf(tc.reply),
			state,
		)
		assert.False(t, again, "%s: a question is sent back once a turn", name)
	}
}

func TestAskForQuestion_LeavesAReplyThatAsksNothing(t *testing.T) {
	t.Parallel()

	for name, reply := range map[string]string{
		"a plain answer": "No, S26110001005205 is not in the list of new shipments. Its " +
			"status is Delayed.",
		"a question answered earlier": "Is it in the list? No: it is Delayed, not New, so the " +
			"planner never saw it.",
		"a question quoted in code": "Run this in the console:\n\n```\nwhy is this late?\n```",
		"a question mid reply": "Why did the calls fail?\n\nEach one sent a shipment id where " +
			"a move id goes. They were fixed and run again.",
	} {
		_, ask := AskAgain(
			plannedTurn(agentruntimeports.RunRequest{Input: "is it in your list"}),
			draftOf(reply),
			&BoardAskState{},
		)
		assert.False(t, ask, name)
	}
}

func TestAskForQuestion_HoldsOnlyAWatchedTurnThatCanAsk(t *testing.T) {
	t.Parallel()

	reply := "Batch 3 is planned.\n\nShould I move on to batch 4?"
	for name, turn := range map[string]*askTurn{
		"unattended": plannedTurn(agentruntimeports.RunRequest{Input: "x", Unattended: true}),
		"a decision follow-up": plannedTurn(
			agentruntimeports.RunRequest{Input: "x", TrenovaWritten: true},
		),
		"ask_user withheld": func() *askTurn {
			turn := plannedTurn(agentruntimeports.RunRequest{Input: "x"})
			turn.noAsk = true
			return turn
		}(),
	} {
		_, ask := AskAgain(turn, draftOf(reply), &BoardAskState{})
		assert.False(t, ask, name)
	}
}
