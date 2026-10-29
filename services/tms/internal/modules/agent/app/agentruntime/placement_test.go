package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/replycheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func formulaCard() agentruntimeports.ShownArtifact {
	return agentruntimeports.ShownArtifact{
		ID:    pulid.MustNew("art_"),
		Kind:  "entity_card",
		Title: "Formula template Per CWT (Hundredweight)",
	}
}

func placementRun(
	t *testing.T,
	shown *agentruntimeports.ShownArtifact,
	delegation *agentruntimeports.Delegation,
	replies ...string,
) (*agentruntimeports.RunResult, *[]kernelports.StreamEvent, *scriptedCompletion) {
	t.Helper()

	turns := []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("formula_get_template", map[string]any{}),
	}
	for _, reply := range replies {
		turns = append(turns, runtimetest.TextTurn(reply))
	}
	completion := &scriptedCompletion{Turns: turns}
	rt := newRuntime(completion, &stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{
		queryTool("formula_get_template", map[string]any{"id": "ft_1"}, nil),
	}}, &stubActionRegistry{}, nil)
	req := &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("formula_get_template"),
		Actor:      runtimetest.Actor(),
		Input:      "Show me formula template Per CWT (Hundredweight)",
		Delegation: delegation,
		ToolObserver: func(agentruntimeports.ToolObservation) (*agentruntimeports.ShownArtifact, error) {
			return shown, nil
		},
	}
	events := recordEvents(req)

	result, err := rt.Run(t.Context(), req)
	require.NoError(t, err)

	return result, events, completion
}

func retried(events *[]kernelports.StreamEvent) bool {
	for _, event := range *events {
		if event.Event == assistantstreamports.AssistantEventRetrying {
			return true
		}
	}

	return false
}

// The Formula Desk ended its reply with the card as a sentence of its own,
// and Haiku ended one with the shipment card on a line of its own (DB-148).
// Each was asked again, and the person watched a finished answer restart to
// move a link. The link is now cut and joined to the reply's own sentence,
// with nothing asked again.
func TestDrive_MovesALinkStandingApartIntoTheRepliesSentence(t *testing.T) {
	t.Parallel()

	card := formulaCard()
	stray := "**Per CWT (Hundredweight)** is active. It charges the base rate for each " +
		"100 lb, rounded up. " + replycheck.ArtifactRef(&card)

	result, events, completion := placementRun(t, &card, nil, stray)

	assert.Equal(t, "**Per CWT (Hundredweight)** is active. It charges the base rate for each "+
		"100 lb, rounded up ("+replycheck.ArtifactRef(&card)+").", result.Reply)
	assert.Empty(t, replycheck.StrayArtifactLinks(result.Reply))
	assert.False(t, retried(events))
	assert.Len(t, completion.Requests, 2, "the model is not asked again")
	assert.Equal(
		t,
		[]assistantstreamports.RegroundAction{assistantstreamports.RegroundPlaceArtifacts},
		regroundedActions(events),
	)
}

// A link in a table cell sits where cutting it could leave the row saying
// nothing, so the model is asked once to name it in a sentence.
func TestDrive_AsksAgainForALinkInATable(t *testing.T) {
	t.Parallel()

	card := formulaCard()
	stray := "It is active.\n\n| Template | Basis |\n| --- | --- |\n| " + replycheck.ArtifactRef(
		&card,
	) +
		" | CWT |"
	fixed := "It is active; " + replycheck.ArtifactRef(&card) + " charges by the hundredweight."

	result, events, completion := placementRun(t, &card, nil, stray, fixed)

	assert.Equal(t, fixed, result.Reply)
	assert.True(t, retried(events))
	assert.Equal(
		t,
		[]assistantstreamports.RegroundAction{assistantstreamports.RegroundRelinkArtifacts},
		regroundedActions(events),
	)

	last := completion.Requests[len(completion.Requests)-1].Messages
	correction := last[len(last)-1]
	assert.Equal(t, llmports.RoleSystem, correction.Role)
	assert.Contains(t, correction.Content, replycheck.PlacementCorrectionLead)
	assert.Contains(
		t,
		correction.Content,
		replycheck.ArtifactRef(&card)+" stands outside a sentence.",
	)
}

// A model that does it again is not asked a third time: the link is taken out
// of where it stood and joined to the reply's sentence nearest it.
func TestDrive_PlacesALinkTheRewriteStillLeftOutsideASentence(t *testing.T) {
	t.Parallel()

	card := formulaCard()
	stray := "| Template |\n| --- |\n| " + replycheck.ArtifactRef(&card) + " |"

	result, events, completion := placementRun(
		t,
		&card,
		nil,
		stray,
		"**"+replycheck.ArtifactRef(&card)+
			"**\n\nIt is active. It charges the base rate for each 100 lb, rounded up.",
	)

	assert.Equal(t, "It is active. It charges the base rate for each 100 lb, rounded up ("+
		replycheck.ArtifactRef(&card)+").", result.Reply)
	assert.Empty(t, replycheck.StrayArtifactLinks(result.Reply))
	assert.Len(t, completion.Requests, 3, "the model is asked once more, not twice")
	assert.Equal(t, []assistantstreamports.RegroundAction{
		assistantstreamports.RegroundRelinkArtifacts,
		assistantstreamports.RegroundPlaceArtifacts,
	}, regroundedActions(events))
}

func TestPlaced_JoinsAStrayLinkToTheSentenceNearestIt(t *testing.T) {
	t.Parallel()

	card := formulaCard()
	table := runtimetest.KeptShipments(15)
	ref := replycheck.ArtifactRef(&card)
	cases := []struct {
		name      string
		reply     string
		want      string
		withTable bool
		fail      bool
	}{
		{
			name: "on a line of its own at the end (DB-148, delayed-reefer-detail)",
			reply: "The late reefer is **SEED-SHP-010**.\n\n- **Pickup:** reached 22h late.\n" +
				"- **No ETA:** no truck is reporting.\n\nI can rank the drivers who could " +
				"take the LA leg.\n\n" + ref,
			want: "The late reefer is **SEED-SHP-010**.\n\n- **Pickup:** reached 22h late.\n" +
				"- **No ETA:** no truck is reporting.\n\nI can rank the drivers who could " +
				"take the LA leg (" + ref + ").",
		},
		{
			name:  "after a weight answer (DB-148, codemode-control-weight-question)",
			reply: "**SEED-SHP-004** weighs **38,000**. It is still unassigned.\n\n" + ref,
			want:  "**SEED-SHP-004** weighs **38,000**. It is still unassigned (" + ref + ").",
		},
		{
			name:  "trailing after a list",
			reply: "Two loads are late:\n\n- PRO-1\n- PRO-2\n\n**" + ref + "**",
			want:  "Two loads are late:\n\n- PRO-1\n- PRO-2 (" + ref + ")",
		},
		{
			name:  "after a sentence on its own line",
			reply: "It is active. " + ref + "\n\nWant the rest?",
			want:  "It is active (" + ref + ").\n\nWant the rest?",
		},
		{
			name:  "leading the reply, joined to the first sentence below",
			reply: ref + "\n\nIt is active.",
			want:  "It is active (" + ref + ").",
		},
		{
			name:  "as a list item of its own",
			reply: "It is active.\n\n- " + ref,
			want:  "It is active (" + ref + ").",
		},
		{
			name:  "already named in a sentence, cut without naming it twice",
			reply: "Its details are in " + ref + ".\n\n" + ref,
			want:  "Its details are in " + ref + ".",
		},
		{
			name: "beside a left-out table, each where it belongs",
			reply: "Fifteen loads are late. The oldest is Acme's.\n\nI can call the " +
				"carriers.\n\n" + ref,
			want: "Fifteen loads are late. The oldest is Acme's (" + replycheck.ArtifactRef(
				&table,
			) +
				").\n\nI can call the carriers (" + ref + ").",
			withTable: true,
		},
		{
			name:  "with no prose to join",
			reply: "| A |\n| - |\n| 1 |\n\n" + ref,
			want:  "| A |\n| - |\n| 1 |\n\n" + ref,
			fail:  true,
		},
	}
	for _, tc := range cases {
		turn := &Turn{artifacts: []agentruntimeports.ShownArtifact{card}}
		if tc.withTable {
			turn.artifacts = append(turn.artifacts, table)
		}
		problems := replycheck.FindPlacementProblems(turn, tc.reply)
		assert.True(t, problems.Mendable(), tc.name)

		got, ok := replycheck.PlaceLinks(turn, tc.reply, &problems)
		assert.Equal(t, tc.want, got, tc.name)
		assert.Equal(t, !tc.fail, ok, tc.name)
		if !tc.fail {
			assert.Empty(t, replycheck.StrayArtifactLinks(got), tc.name)
		}
	}
}

func TestPlacementProblems_OnlyALinkStandingApartIsMendable(t *testing.T) {
	t.Parallel()

	card := formulaCard()
	ref := replycheck.ArtifactRef(&card)
	turn := &Turn{artifacts: []agentruntimeports.ShownArtifact{card}}

	for name, reply := range map[string]string{
		"in a table cell":   "It is active.\n\n| Template |\n| --- |\n| " + ref + " |",
		"in a heading":      "## Rates in " + ref + "\n\nIt is active.",
		"in a fenced block": "It is active.\n\n```\n" + ref + "\n```",
		"beside an invented id": "It is active.\n\n" + ref +
			" [x](artifact:art_01MADEUP0000000000000000000)",
	} {
		problems := replycheck.FindPlacementProblems(turn, reply)
		assert.False(t, problems.Empty(), name)
		assert.False(t, problems.Mendable(), name)
	}
}

// A table the reply was told to point to, and did not, is asked for.
// A table the reply must name and left out is named at its end, without
// asking the model again: a finished answer restarting only to add a link was
// the restart people saw most.
func TestDrive_NamesATableTheReplyLeftOutWithoutAskingAgain(t *testing.T) {
	t.Parallel()

	table := runtimetest.KeptShipments(15)

	result, events, completion := placementRun(t, &table, nil, "Fifteen loads are late.")

	assert.Equal(t, "Fifteen loads are late ("+replycheck.ArtifactRef(&table)+").", result.Reply)
	assert.False(t, retried(events))
	assert.Len(t, completion.Requests, 2)
	assert.Equal(
		t,
		[]assistantstreamports.RegroundAction{assistantstreamports.RegroundPlaceArtifacts},
		regroundedActions(events),
	)
}

// An id the conversation never held is not left as a link that opens nothing.
func TestDrive_UnlinksAnArtifactTheConversationNeverHeld(t *testing.T) {
	t.Parallel()

	card := formulaCard()
	invented := "See [the template](artifact:art_01MADEUP0000000000000000000) for more; " +
		replycheck.ArtifactRef(&card) + " has the rest."

	result, _, _ := placementRun(t, &card, nil, invented, invented)

	assert.Equal(t, "See the template for more; "+replycheck.ArtifactRef(&card)+" has the rest.",
		result.Reply)
}

// A delegate's reply is read by the agent that handed it the task, which
// names what it needs in its own reply; nobody is asked again for it.
func TestDrive_LeavesADelegatesLinksToTheAgentThatAsked(t *testing.T) {
	t.Parallel()

	card := formulaCard()
	stray := "It is active. " + replycheck.ArtifactRef(&card)

	result, events, completion := placementRun(t, &card, &agentruntimeports.Delegation{
		ParentAgentID:   pulid.MustNew("agdef_"),
		ParentAgentName: "Formula Desk",
		CallID:          "call_parent",
	}, stray)

	assert.Equal(t, stray, result.Reply)
	assert.False(t, retried(events))
	assert.Len(t, completion.Requests, 2)
}

// A reply that names its artifacts in sentences costs nothing more.
func TestDrive_LeavesAReplyThatNamesItsArtifactsInSentences(t *testing.T) {
	t.Parallel()

	card := formulaCard()
	reply := "**Per CWT (Hundredweight)** is active. " + replycheck.ArtifactRef(&card) +
		" charges the base rate for each 100 lb, rounded up."

	result, events, completion := placementRun(t, &card, nil, reply)

	assert.Equal(t, reply, result.Reply)
	assert.False(t, retried(events))
	assert.Empty(t, regroundedActions(events))
	assert.Len(t, completion.Requests, 2)
}

// DB-237: asked in Spanish, the agent answered in Spanish and the reply then
// ended in English, "The full table is in [Shipments](artifact:…).", because
// the fallback wrote its own sentence. It now writes no words: the link joins
// the reply's own sentence, in whatever language the model wrote it.
func TestDrive_NamesALeftOutTableInTheRepliesOwnLanguage(t *testing.T) {
	t.Parallel()

	table := runtimetest.KeptShipments(15)
	reply := "Hay quince cargas retrasadas. La más antigua es de Acme.\n\n" +
		"¿Quieres que llame a los transportistas?"

	result, events, completion := placementRun(t, &table, nil, reply)

	assert.Equal(t, "Hay quince cargas retrasadas. La más antigua es de Acme ("+
		replycheck.ArtifactRef(
			&table,
		)+").\n\n¿Quieres que llame a los transportistas?", result.Reply)
	assert.NotContains(t, result.Reply, "The full table")
	assert.Empty(t, replycheck.StrayArtifactLinks(result.Reply))
	assert.False(t, retried(events))
	assert.Len(t, completion.Requests, 2)
	assert.Equal(
		t,
		[]assistantstreamports.RegroundAction{assistantstreamports.RegroundPlaceArtifacts},
		regroundedActions(events),
	)
}

func draftVersions(lineage bool) (first, second agentruntimeports.ShownArtifact) {
	first = agentruntimeports.ShownArtifact{
		ID:    pulid.MustNew("art_"),
		Kind:  "edi_connection_draft",
		Title: "EDI connection for Acme Retail Group",
	}
	second = agentruntimeports.ShownArtifact{
		ID:    pulid.MustNew("art_"),
		Kind:  first.Kind,
		Title: first.Title,
	}
	if lineage {
		first.Lineage = first.ID
		second.Lineage = first.ID
	}

	return first, second
}

func redraftRun(
	t *testing.T,
	first, second *agentruntimeports.ShownArtifact,
	replies ...string,
) (*agentruntimeports.RunResult, *[]kernelports.StreamEvent) {
	t.Helper()

	turns := []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("edi_draft_connection", map[string]any{}),
		runtimetest.ToolTurn("edi_draft_connection", map[string]any{"values": []any{"again"}}),
	}
	for _, reply := range replies {
		turns = append(turns, runtimetest.TextTurn(reply))
	}
	rt := newRuntime(&scriptedCompletion{Turns: turns}, &stubQueryRegistry{
		Tools: []agenttoolports.AgentQueryTool{
			queryTool("edi_draft_connection", map[string]any{"ready": true}, nil),
		},
	}, &stubActionRegistry{}, nil)
	shown := []*agentruntimeports.ShownArtifact{first, second}
	req := &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("edi_draft_connection"),
		Actor:      runtimetest.Actor(),
		Input:      "acme retail sent over their edi guide, can you get them set up?",
		ToolObserver: func(agentruntimeports.ToolObservation) (*agentruntimeports.ShownArtifact, error) {
			next := shown[0]
			shown = shown[1:]

			return next, nil
		},
	}
	events := recordEvents(req)

	result, err := rt.Run(t.Context(), req)
	require.NoError(t, err)

	return result, events
}

func TestDrive_AsksOnlyForTheLatestVersionOfAnArtifact(t *testing.T) {
	t.Parallel()

	first, second := draftVersions(true)
	reply := "The connection is drafted in " + replycheck.ArtifactRef(
		&second,
	) + ", waiting for approval."

	result, events := redraftRun(t, &first, &second, reply)

	assert.Equal(t, reply, result.Reply)
	assert.False(t, retried(events))
	assert.Empty(t, regroundedActions(events))
}

func TestDrive_AsksForEachArtifactOutsideALineage(t *testing.T) {
	t.Parallel()

	first, second := draftVersions(false)
	reply := "The connection is drafted in " + replycheck.ArtifactRef(
		&second,
	) + ", waiting for approval."

	result, events := redraftRun(t, &first, &second, reply)

	assert.Equal(t, "The connection is drafted in "+replycheck.ArtifactRef(&second)+
		", waiting for approval ("+replycheck.ArtifactRef(&first)+").", result.Reply,
		"a draft of another partner is not replaced by this one, so it is named too")
	assert.False(t, retried(events))
}
