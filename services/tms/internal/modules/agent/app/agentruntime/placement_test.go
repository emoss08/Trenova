package agentruntime

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func formulaCard() agentports.ShownArtifact {
	return agentports.ShownArtifact{
		ID:    pulid.MustNew("art_"),
		Kind:  "entity_card",
		Title: "Formula template Per CWT (Hundredweight)",
	}
}

func placementRun(
	t *testing.T,
	shown *agentports.ShownArtifact,
	delegation *agentports.Delegation,
	replies ...string,
) (*agentports.RunResult, *[]kernelports.StreamEvent, *scriptedCompletion) {
	t.Helper()

	turns := []*agentports.ChatCompletionResult{toolTurn("formula_get_template", map[string]any{})}
	for _, reply := range replies {
		turns = append(turns, textTurn(reply))
	}
	completion := &scriptedCompletion{Turns: turns}
	rt := newRuntime(completion, &stubQueryRegistry{Tools: []agentports.AgentQueryTool{
		queryTool("formula_get_template", map[string]any{"id": "ft_1"}, nil),
	}}, &stubActionRegistry{}, nil)
	req := &agentports.RunRequest{
		Definition: testDefinition("formula_get_template"),
		Actor:      testActor(),
		Input:      "Show me formula template Per CWT (Hundredweight)",
		Delegation: delegation,
		ToolObserver: func(agentports.ToolObservation) (*agentports.ShownArtifact, error) {
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
		if event.Event == agentports.AssistantEventRetrying {
			return true
		}
	}

	return false
}

func TestStrayArtifactLinks(t *testing.T) {
	t.Parallel()

	ref := ArtifactRef(&agentports.ShownArtifact{
		ID:    pulid.MustNew("art_"),
		Title: "Formula template Per CWT (Hundredweight)",
	})
	id := ArtifactRefIDs(ref)[0]

	stray := map[string]string{
		"after a sentence that has ended": "Its description says it multiplies a base rate " +
			"per CWT by the shipment weight divided by 100, rounded up. " + ref + "\n\n" +
			"You can view the template in Formula templates.",
		"on a line of its own":          "It is active.\n\n" + ref,
		"bold, on a line of its own":    "It is active.\n\n**" + ref + "**",
		"as a list item":                "It is active.\n\n- " + ref,
		"in a table":                    "| Template |\n| --- |\n| " + ref + " |",
		"in a heading":                  "## " + ref + "\n\nIt is active.",
		"after an arrow":                "It is active. → " + ref + ".",
		"as a sentence of its own":      ref + ". It is active.",
		"after a table":                 "| A |\n| - |\n| 1 |\n\n" + ref,
		"set off by a question mark":    "Want the rest? " + ref,
		"at the end of a list item":     "- It is active. " + ref,
		"after an exclamation mark":     "It is active! " + ref + ".",
		"between two sentences":         "It is active. " + ref + ". It rates by weight.",
		"inside a fenced code block":    "```\n" + ref + " is here\n```",
		"after a blockquote's sentence": "> It is active. " + ref,
	}
	for name, reply := range stray {
		assert.Equal(t, []string{id}, StrayArtifactLinks(reply), name)
	}

	inline := map[string]string{
		"leading a sentence":         ref + " has its full details.",
		"inside a sentence":          "See the " + ref + " for its full details.",
		"ending a sentence":          "Its full details are in " + ref + ".",
		"after a colon":              "Here is the template: " + ref + ".",
		"in a list item's sentence":  "- See " + ref + " for the rest.",
		"in a quoted sentence":       "> See " + ref + " for the rest.",
		"between two sentences":      "It is active. Read " + ref + " next. It rates by weight.",
		"with its words left out":    "Its details are in [](artifact:" + id + ").",
		"on a line after a sentence": "It is active.\nIts details are in " + ref + ".",
		"with a figure before it":    "It charges $2.50 per CWT, as " + ref + " shows.",
	}
	for name, reply := range inline {
		assert.Empty(t, StrayArtifactLinks(reply), name)
	}
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
		"100 lb, rounded up. " + ArtifactRef(&card)

	result, events, completion := placementRun(t, &card, nil, stray)

	assert.Equal(t, "**Per CWT (Hundredweight)** is active. It charges the base rate for each "+
		"100 lb, rounded up ("+ArtifactRef(&card)+").", result.Reply)
	assert.Empty(t, StrayArtifactLinks(result.Reply))
	assert.False(t, retried(events))
	assert.Len(t, completion.Requests, 2, "the model is not asked again")
	assert.Equal(t, []agentports.RegroundAction{agentports.RegroundPlaceArtifacts},
		regroundedActions(events))
}

// A link in a table cell sits where cutting it could leave the row saying
// nothing, so the model is asked once to name it in a sentence.
func TestDrive_AsksAgainForALinkInATable(t *testing.T) {
	t.Parallel()

	card := formulaCard()
	stray := "It is active.\n\n| Template | Basis |\n| --- | --- |\n| " + ArtifactRef(&card) +
		" | CWT |"
	fixed := "It is active; " + ArtifactRef(&card) + " charges by the hundredweight."

	result, events, completion := placementRun(t, &card, nil, stray, fixed)

	assert.Equal(t, fixed, result.Reply)
	assert.True(t, retried(events))
	assert.Equal(t, []agentports.RegroundAction{agentports.RegroundRelinkArtifacts},
		regroundedActions(events))

	last := completion.Requests[len(completion.Requests)-1].Messages
	correction := last[len(last)-1]
	assert.Equal(t, agentports.RoleSystem, correction.Role)
	assert.Contains(t, correction.Content, placementCorrectionLead)
	assert.Contains(t, correction.Content, ArtifactRef(&card)+" stands outside a sentence.")
}

// A model that does it again is not asked a third time: the link is taken out
// of where it stood and joined to the reply's sentence nearest it.
func TestDrive_PlacesALinkTheRewriteStillLeftOutsideASentence(t *testing.T) {
	t.Parallel()

	card := formulaCard()
	stray := "| Template |\n| --- |\n| " + ArtifactRef(&card) + " |"

	result, events, completion := placementRun(t, &card, nil, stray, "**"+ArtifactRef(&card)+
		"**\n\nIt is active. It charges the base rate for each 100 lb, rounded up.")

	assert.Equal(t, "It is active. It charges the base rate for each 100 lb, rounded up ("+
		ArtifactRef(&card)+").", result.Reply)
	assert.Empty(t, StrayArtifactLinks(result.Reply))
	assert.Len(t, completion.Requests, 3, "the model is asked once more, not twice")
	assert.Equal(t, []agentports.RegroundAction{
		agentports.RegroundRelinkArtifacts,
		agentports.RegroundPlaceArtifacts,
	}, regroundedActions(events))
}

func TestPlaced_JoinsAStrayLinkToTheSentenceNearestIt(t *testing.T) {
	t.Parallel()

	card := formulaCard()
	table := keptShipments(15)
	ref := ArtifactRef(&card)
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
			want: "Fifteen loads are late. The oldest is Acme's (" + ArtifactRef(&table) +
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
		turn := &Turn{artifacts: []agentports.ShownArtifact{card}}
		if tc.withTable {
			turn.artifacts = append(turn.artifacts, table)
		}
		problems := turn.placementProblems(tc.reply)
		assert.True(t, problems.mendable(), tc.name)

		got, ok := turn.placed(tc.reply, &problems)
		assert.Equal(t, tc.want, got, tc.name)
		assert.Equal(t, !tc.fail, ok, tc.name)
		if !tc.fail {
			assert.Empty(t, StrayArtifactLinks(got), tc.name)
		}
	}
}

func TestPlacementProblems_OnlyALinkStandingApartIsMendable(t *testing.T) {
	t.Parallel()

	card := formulaCard()
	ref := ArtifactRef(&card)
	turn := &Turn{artifacts: []agentports.ShownArtifact{card}}

	for name, reply := range map[string]string{
		"in a table cell":   "It is active.\n\n| Template |\n| --- |\n| " + ref + " |",
		"in a heading":      "## Rates in " + ref + "\n\nIt is active.",
		"in a fenced block": "It is active.\n\n```\n" + ref + "\n```",
		"beside an invented id": "It is active.\n\n" + ref +
			" [x](artifact:art_01MADEUP0000000000000000000)",
	} {
		problems := turn.placementProblems(reply)
		assert.False(t, problems.empty(), name)
		assert.False(t, problems.mendable(), name)
	}
}

// A table the reply was told to point to, and did not, is asked for.
// A table the reply must name and left out is named at its end, without
// asking the model again: a finished answer restarting only to add a link was
// the restart people saw most.
func TestDrive_NamesATableTheReplyLeftOutWithoutAskingAgain(t *testing.T) {
	t.Parallel()

	table := keptShipments(15)

	result, events, completion := placementRun(t, &table, nil, "Fifteen loads are late.")

	assert.Equal(t, "Fifteen loads are late ("+ArtifactRef(&table)+").", result.Reply)
	assert.False(t, retried(events))
	assert.Len(t, completion.Requests, 2)
	assert.Equal(t, []agentports.RegroundAction{agentports.RegroundPlaceArtifacts},
		regroundedActions(events))
}

// An id the conversation never held is not left as a link that opens nothing.
func TestDrive_UnlinksAnArtifactTheConversationNeverHeld(t *testing.T) {
	t.Parallel()

	card := formulaCard()
	invented := "See [the template](artifact:art_01MADEUP0000000000000000000) for more; " +
		ArtifactRef(&card) + " has the rest."

	result, _, _ := placementRun(t, &card, nil, invented, invented)

	assert.Equal(t, "See the template for more; "+ArtifactRef(&card)+" has the rest.",
		result.Reply)
}

// A delegate's reply is read by the agent that handed it the task, which
// names what it needs in its own reply; nobody is asked again for it.
func TestDrive_LeavesADelegatesLinksToTheAgentThatAsked(t *testing.T) {
	t.Parallel()

	card := formulaCard()
	stray := "It is active. " + ArtifactRef(&card)

	result, events, completion := placementRun(t, &card, &agentports.Delegation{
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
	reply := "**Per CWT (Hundredweight)** is active. " + ArtifactRef(&card) +
		" charges the base rate for each 100 lb, rounded up."

	result, events, completion := placementRun(t, &card, nil, reply)

	assert.Equal(t, reply, result.Reply)
	assert.False(t, retried(events))
	assert.Empty(t, regroundedActions(events))
	assert.Len(t, completion.Requests, 2)
}

func TestNameArtifactLinks(t *testing.T) {
	t.Parallel()

	titles := map[string]string{"art_1": "Shipment S26100090038417", "art_2": "Shipments"}
	titleOf := func(id, _ string) (string, bool) {
		title, ok := titles[id]
		return title, ok
	}

	assert.Equal(t,
		"Its [Shipment S26100090038417](artifact:art_1) has the details; "+
			"[Shipments](artifact:art_2) lists the rest, and the brief has more.",
		NameArtifactLinks("Its [shipment card](artifact:art_1) has the details; "+
			"[Shipments table](artifact:art_2) lists the rest, and the "+
			"[brief](artifact:art_9) has more.", titleOf))
	assert.Equal(t, "No links here.", NameArtifactLinks("No links here.", titleOf))
	assert.Equal(t, "[Shipments](artifact:art_2)", NameArtifactLinks("[](artifact:art_2)", titleOf))
}

// DB-237: asked in Spanish, the agent answered in Spanish and the reply then
// ended in English, "The full table is in [Shipments](artifact:…).", because
// the fallback wrote its own sentence. It now writes no words: the link joins
// the reply's own sentence, in whatever language the model wrote it.
func TestDrive_NamesALeftOutTableInTheRepliesOwnLanguage(t *testing.T) {
	t.Parallel()

	table := keptShipments(15)
	reply := "Hay quince cargas retrasadas. La más antigua es de Acme.\n\n" +
		"¿Quieres que llame a los transportistas?"

	result, events, completion := placementRun(t, &table, nil, reply)

	assert.Equal(t, "Hay quince cargas retrasadas. La más antigua es de Acme ("+
		ArtifactRef(&table)+").\n\n¿Quieres que llame a los transportistas?", result.Reply)
	assert.NotContains(t, result.Reply, "The full table")
	assert.Empty(t, StrayArtifactLinks(result.Reply))
	assert.False(t, retried(events))
	assert.Len(t, completion.Requests, 2)
	assert.Equal(t, []agentports.RegroundAction{agentports.RegroundPlaceArtifacts},
		regroundedActions(events))
}

func TestLinkRefs_JoinsRefsToTheFirstSentence(t *testing.T) {
	t.Parallel()

	a := "[Shipments](artifact:art_1)"
	b := "[Invoices](artifact:art_2)"
	cases := []struct {
		name string
		text string
		refs []string
		want string
	}{
		{
			name: "before the closing punctuation of the first sentence line",
			text: "Fifteen loads are late.\n\nWant me to call the carriers?",
			refs: []string{a},
			want: "Fifteen loads are late (" + a + ").\n\nWant me to call the carriers?",
		},
		{
			name: "after a colon that introduces it",
			text: "Aquí están las cargas:\n\n- PRO-1\n- PRO-2",
			refs: []string{a},
			want: "Aquí están las cargas: " + a + "\n\n- PRO-1\n- PRO-2",
		},
		{
			name: "after the markup round a colon",
			text: "**Late loads:**",
			refs: []string{a},
			want: "**Late loads:** " + a,
		},
		{
			name: "inside the markup round a sentence",
			text: "**Fifteen loads are late.**",
			refs: []string{a},
			want: "**Fifteen loads are late (" + a + ").**",
		},
		{
			name: "before a full-width stop",
			text: "有十五批货物延误。",
			refs: []string{a},
			want: "有十五批货物延误 (" + a + ")。",
		},
		{
			name: "several, in one parenthesis",
			text: "Fifteen loads are late and three invoices are open!",
			refs: []string{a, b},
			want: "Fifteen loads are late and three invoices are open (" + a + ", " + b + ")!",
		},
		{
			name: "past a label, a table, a heading and a fence",
			text: "## Late\n\n**Summary**\n\n| A |\n| - |\n| 1 |\n\n```\nnot this.\n```\n" +
				"Fifteen loads are late.",
			refs: []string{a},
			want: "## Late\n\n**Summary**\n\n| A |\n| - |\n| 1 |\n\n```\nnot this.\n```\n" +
				"Fifteen loads are late (" + a + ").",
		},
		{
			name: "to a line with no closing punctuation when none has one",
			text: "**Summary**",
			refs: []string{a},
			want: "**Summary** (" + a + ")",
		},
		{
			name: "on a line of its own when the reply has no prose",
			text: "| A |\n| - |\n| 1 |",
			refs: []string{a},
			want: "| A |\n| - |\n| 1 |\n\n" + a,
		},
		{name: "as the reply when it is empty", text: "", refs: []string{a}, want: a},
		{name: "unchanged with nothing to name", text: "It is active.", want: "It is active."},
	}
	for _, tc := range cases {
		got, _ := linkRefs(strings.Split(tc.text, "\n"), nil, tc.refs)
		assert.Equal(t, tc.want, got, tc.name)
		assert.NotContains(t, got, "The full", tc.name)
	}
}

func draftVersions(lineage bool) (first, second agentports.ShownArtifact) {
	first = agentports.ShownArtifact{
		ID:    pulid.MustNew("art_"),
		Kind:  "edi_connection_draft",
		Title: "EDI connection for Acme Retail Group",
	}
	second = agentports.ShownArtifact{
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
	first, second *agentports.ShownArtifact,
	replies ...string,
) (*agentports.RunResult, *[]kernelports.StreamEvent) {
	t.Helper()

	turns := []*agentports.ChatCompletionResult{
		toolTurn("edi_draft_connection", map[string]any{}),
		toolTurn("edi_draft_connection", map[string]any{"values": []any{"again"}}),
	}
	for _, reply := range replies {
		turns = append(turns, textTurn(reply))
	}
	rt := newRuntime(&scriptedCompletion{Turns: turns}, &stubQueryRegistry{
		Tools: []agentports.AgentQueryTool{
			queryTool("edi_draft_connection", map[string]any{"ready": true}, nil),
		},
	}, &stubActionRegistry{}, nil)
	shown := []*agentports.ShownArtifact{first, second}
	req := &agentports.RunRequest{
		Definition: testDefinition("edi_draft_connection"),
		Actor:      testActor(),
		Input:      "acme retail sent over their edi guide, can you get them set up?",
		ToolObserver: func(agentports.ToolObservation) (*agentports.ShownArtifact, error) {
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
	reply := "The connection is drafted in " + ArtifactRef(&second) + ", waiting for approval."

	result, events := redraftRun(t, &first, &second, reply)

	assert.Equal(t, reply, result.Reply)
	assert.False(t, retried(events))
	assert.Empty(t, regroundedActions(events))
}

func TestDrive_AsksForEachArtifactOutsideALineage(t *testing.T) {
	t.Parallel()

	first, second := draftVersions(false)
	reply := "The connection is drafted in " + ArtifactRef(&second) + ", waiting for approval."

	result, events := redraftRun(t, &first, &second, reply)

	assert.Equal(t, "The connection is drafted in "+ArtifactRef(&second)+
		", waiting for approval ("+ArtifactRef(&first)+").", result.Reply,
		"a draft of another partner is not replaced by this one, so it is named too")
	assert.False(t, retried(events))
}
