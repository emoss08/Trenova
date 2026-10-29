package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingObserver struct {
	seen  []agentruntimeports.ToolObservation
	shown *agentruntimeports.ShownArtifact
	err   error
}

func (o *recordingObserver) observe(
	observation agentruntimeports.ToolObservation,
) (*agentruntimeports.ShownArtifact, error) {
	o.seen = append(o.seen, observation)

	return o.shown, o.err
}

func offeredNames(specs []kernelports.ToolSpec) []string {
	names := make([]string, 0, len(specs))
	for _, spec := range specs {
		names = append(names, spec.Name)
	}

	return names
}

func runWithObserver(
	t *testing.T,
	observer agentruntimeports.ToolObserver,
	tools []agenttoolports.AgentQueryTool,
	turns ...*llmports.ChatCompletionResult,
) (*agentruntimeports.RunResult, *scriptedCompletion) {
	t.Helper()

	completion := &scriptedCompletion{Turns: turns}
	rt := newRuntime(completion, &stubQueryRegistry{Tools: tools}, &stubActionRegistry{}, nil)
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name())
	}

	result, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition:   runtimetest.Definition(names...),
		Actor:        runtimetest.Actor(),
		Input:        "Tell me about SEED-SHP-007 and publish the details in an artifact.",
		ToolObserver: observer,
	})
	require.NoError(t, err)

	return result, completion
}

/*
A conversation with artifacts can publish one, and the agent is told how.

Asked to "publish the details in an artifact", the Dispatch desk said it had no
tool for creating artifacts and wrote the whole brief into the reply. A turn
that keeps artifacts now offers publish_artifact and says in the prompt when to
reach for it.
*/
func TestRun_OffersPublishingWhereArtifactsAreKept(t *testing.T) {
	t.Parallel()

	observer := &recordingObserver{}
	_, completion := runWithObserver(t, observer.observe, nil, runtimetest.TextTurn("Here it is."))

	assert.Contains(t, offeredNames(completion.LastReq.Tools), runtimetools.PublishArtifactName)
	assert.Contains(t, completion.LastReq.System, "## Artifacts")
	assert.Contains(t, completion.LastReq.System, "publish_artifact")
}

// A run with nowhere to put a document is not offered one, and is not told
// about artifacts it cannot see.
func TestRun_DoesNotOfferPublishingWithoutAPane(t *testing.T) {
	t.Parallel()

	_, completion := runWithObserver(
		t,
		nil,
		nil,
		runtimetest.ToolTurn(
			runtimetools.PublishArtifactName,
			map[string]any{"title": "Brief", "body": "# Brief"},
		),
		runtimetest.TextTurn("Here it is."),
	)

	assert.NotContains(t, offeredNames(completion.LastReq.Tools), runtimetools.PublishArtifactName)
	assert.NotContains(t, completion.LastReq.System, "## Artifacts")
	last := completion.LastReq.Messages[len(completion.LastReq.Messages)-1]
	assert.True(t, last.IsError)
	assert.Contains(t, last.Content, "nowhere to publish")
}

/*
A quick question in the command palette has nothing beside its answer.

Asked for the organization's shipment types, Nova in the palette replied "The
full table is in [Shipment types](artifact:…)", a link the palette cannot open.
An inline turn is offered no way to publish or ask for a decision on a card,
and is told to write its tables into the reply, even on a conversation that
would otherwise keep what it publishes.
*/
func TestRun_InlineTurnWritesItsTablesIntoTheReply(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.TextTurn("We have **10 shipment types**."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)

	_, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition(),
		Actor:      runtimetest.Actor(),
		Input:      "What shipment types do we have?",
		ThreadID:   pulid.MustNew("athr_"),
		Publishes:  true,
		Inline:     true,
	})
	require.NoError(t, err)

	offered := offeredNames(completion.LastReq.Tools)
	assert.NotContains(t, offered, runtimetools.PublishArtifactName)
	assert.NotContains(t, offered, runtimetools.RequestDecisionName)
	assert.NotContains(t, completion.LastReq.System, "## Artifacts")
	assert.Contains(t, completion.LastReq.System, "Nothing is kept beside this reply")
	assert.NotContains(t, completion.LastReq.System, "point to the table the tool result names")
}

// A published document reaches the observer whole, and the model is told the
// id it can revise it by.
func TestRun_PublishesADocumentAndReturnsItsID(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("art_")
	observer := &recordingObserver{shown: &agentruntimeports.ShownArtifact{
		ID: id, Kind: "document", Title: "SEED-SHP-007 brief",
	}}
	result, _ := runWithObserver(t, observer.observe, nil,
		runtimetest.ToolTurn(runtimetools.PublishArtifactName, map[string]any{
			"title": "SEED-SHP-007 brief",
			"body":  "# SEED-SHP-007\n\nRange Logistics, Chicago to Denver.",
		}),
		runtimetest.TextTurn("The brief is open beside the conversation."),
	)

	require.Len(t, observer.seen, 1)
	document, ok := observer.seen[0].Data.(agentruntimeports.PublishedDocument)
	require.True(t, ok)
	assert.Equal(t, "SEED-SHP-007 brief", document.Title)
	assert.Contains(t, document.Body, "Range Logistics")

	toolMessage := result.Messages[2]
	assert.False(t, toolMessage.ToolFailed)
	assert.Contains(t, toolMessage.Content, id.String())
	assert.Contains(t, toolMessage.Content, "do not repeat its text")
}

// A document that could not be kept is a failure the model hears about, so it
// puts the text in the reply rather than pointing at nothing.
func TestRun_SaysWhenADocumentCouldNotBeKept(t *testing.T) {
	t.Parallel()

	observer := &recordingObserver{}
	result, _ := runWithObserver(
		t,
		observer.observe,
		nil,
		runtimetest.ToolTurn(
			runtimetools.PublishArtifactName,
			map[string]any{"title": "Brief", "body": "# Brief"},
		),
		runtimetest.TextTurn("Here it is."),
	)

	assert.True(t, result.Messages[2].ToolFailed)
	assert.Contains(t, result.Messages[2].Content, "Put the text in your reply")
}

// A document that publishes an analysis names it, and the recorder adds it
// from what the analysis kept; the same analysis named twice is added once.
func TestRun_PublishesTheAnalysesADocumentNames(t *testing.T) {
	t.Parallel()

	analysis := pulid.MustNew("art_")
	observer := &recordingObserver{shown: &agentruntimeports.ShownArtifact{
		ID: pulid.MustNew("art_"), Kind: "document", Title: "Lane brief",
	}}
	runWithObserver(t, observer.observe, nil,
		runtimetest.ToolTurn(runtimetools.PublishArtifactName, map[string]any{
			"title":    "Lane brief",
			"body":     "Revenue is spread thin.",
			"analyses": []any{analysis.String(), analysis.String()},
		}),
		runtimetest.TextTurn("The brief is beside the conversation."),
	)

	require.Len(t, observer.seen, 1)
	document, ok := observer.seen[0].Data.(agentruntimeports.PublishedDocument)
	require.True(t, ok)
	assert.Equal(t, []pulid.ID{analysis}, document.Analyses)
}

// An analysis named by something that is not an id is refused before
// anything is kept.
func TestRun_RefusesAnAnalysisThatIsNotAnID(t *testing.T) {
	t.Parallel()

	observer := &recordingObserver{}
	result, _ := runWithObserver(t, observer.observe, nil,
		runtimetest.ToolTurn(runtimetools.PublishArtifactName, map[string]any{
			"title": "Brief", "body": "# Brief", "analyses": []any{"lanes"},
		}),
		runtimetest.TextTurn("Sorry."),
	)

	for _, seen := range observer.seen {
		_, kept := seen.Data.(agentruntimeports.PublishedDocument)
		assert.False(t, kept, "no document is handed over to keep")
	}
	assert.True(t, result.Messages[2].ToolFailed)
	assert.Contains(t, result.Messages[2].Content, "artifactId an analysis_run result kept")
}

// A publish the model can fix is answered as one: it is told to send the call
// again, never to put the text in its reply, which for an analysis would be
// the retyping that publishing it replaces.
func TestRun_AFixablePublishIsSentAgainNotRetyped(t *testing.T) {
	t.Parallel()

	observer := &recordingObserver{err: &agentruntimeports.PublishMistake{
		Reason: "an analysis it names is not one this conversation kept",
	}}
	result, _ := runWithObserver(
		t,
		observer.observe,
		nil,
		runtimetest.ToolTurn(
			runtimetools.PublishArtifactName,
			map[string]any{"title": "Brief", "body": "# Brief"},
		),
		runtimetest.TextTurn("Sorry."),
	)

	content := result.Messages[2].Content
	assert.True(t, result.Messages[2].ToolFailed)
	assert.Contains(t, content, "an analysis it names is not one this conversation kept")
	assert.Contains(t, content, "Fix the call and send it again")
	assert.NotContains(t, content, "Put the text in your reply")
}

// A publish with nothing in it is refused before anything is kept.
func TestRun_RefusesAnEmptyDocument(t *testing.T) {
	t.Parallel()

	observer := &recordingObserver{}
	result, _ := runWithObserver(
		t,
		observer.observe,
		nil,
		runtimetest.ToolTurn(
			runtimetools.PublishArtifactName,
			map[string]any{"title": "", "body": ""},
		),
		runtimetest.TextTurn("Sorry."),
	)

	assert.True(t, result.Messages[2].ToolFailed)
	assert.Contains(t, result.Messages[2].Content, "title is required")
	assert.Contains(t, result.Messages[2].Content, "body is required")
}

/*
A result the person can already see says so.

The Dispatch desk listed twenty-five shipments and then reprinted all of them
as markdown tables under the table the pane was showing. The tool result now
carries a note that it is on screen and what to do instead.
*/
func TestRun_TellsTheModelWhatThePersonAlreadySees(t *testing.T) {
	t.Parallel()

	observer := &recordingObserver{shown: &agentruntimeports.ShownArtifact{
		ID: pulid.MustNew("art_"), Kind: "table_view", Title: "Shipments", Rows: 25,
	}}
	tool := queryTool("shipment_list", map[string]any{"items": []any{}}, nil)
	result, _ := runWithObserver(t, observer.observe, []agenttoolports.AgentQueryTool{tool},
		runtimetest.ToolTurn("shipment_list", map[string]any{"query": ""}),
		runtimetest.TextTurn("Twenty-five shipments; two are delayed."),
	)

	content := result.Messages[2].Content
	assert.Contains(t, content, `kept as a table titled "Shipments"`)
	assert.Contains(t, content, "do not write its rows out as a markdown table")
	_, payload, fenced := toolresult.Unfence(content)
	require.True(t, fenced, "the note sits after the fence and does not break reading it back")
	assert.Contains(t, payload, `"items"`)
}

/*
A short list is answered in the reply, not kept beside it.

A billing queue of nine items came back both as a markdown table and as a
table artifact under it: the same rows twice. A result of a dozen rows or
fewer is now answered with a table in the reply, and the model is told not
to point to the artifact, so it is not kept — unless it would rank or
compare the rows, when it points to the table instead, and so is given the
pointer to write.
*/
func TestRun_ShortListsAreAnsweredInTheReply(t *testing.T) {
	t.Parallel()

	observer := &recordingObserver{shown: &agentruntimeports.ShownArtifact{
		ID: pulid.MustNew("art_"), Kind: "table_view", Title: "Billing queue items", Rows: 9,
	}}
	tool := queryTool("billing_list_queue_items", map[string]any{"items": []any{}}, nil)
	result, _ := runWithObserver(t, observer.observe, []agenttoolports.AgentQueryTool{tool},
		runtimetest.ToolTurn("billing_list_queue_items", map[string]any{"query": ""}),
		runtimetest.TextTurn("Nine items."),
	)

	content := result.Messages[2].Content
	assert.Contains(t, content, "9 rows, few enough to answer in your reply")
	assert.Contains(t, content, "do not point to the table")
	assert.Contains(t, content, "When you would rank or compare its rows")
	assert.Contains(t, content, "artifact:", "a ranking from a short list is pointed to")
}

/*
A table the person works from is pointed to however short it is.

The billing queue opens with a checkbox on each row and a bar to approve the
ones selected. Two items answered as a markdown table leave nothing to select.
*/
func TestRun_AnActionableTableIsPointedToHoweverShort(t *testing.T) {
	t.Parallel()

	observer := &recordingObserver{shown: &agentruntimeports.ShownArtifact{
		ID: pulid.MustNew("art_"), Kind: "table_view", Title: "Billing queue items", Rows: 2,
		Actionable: true,
	}}
	tool := queryTool("billing_list_queue_items", map[string]any{"items": []any{}}, nil)
	result, _ := runWithObserver(t, observer.observe, []agenttoolports.AgentQueryTool{tool},
		runtimetest.ToolTurn("billing_list_queue_items", map[string]any{"query": ""}),
		runtimetest.TextTurn("Two items."),
	)

	content := result.Messages[2].Content
	assert.Contains(t, content, "works from beside the conversation")
	assert.Contains(t, content, "artifact:", "the reply is given the pointer to write")
	assert.NotContains(t, content, "few enough to answer in your reply")
}

/*
A table whose rows open as the record itself is pointed to however short it
is. Three formula templates answered in the reply leave no formula to read
and nothing to price.
*/
func TestRun_ATableOfRecordsToOpenIsPointedToHoweverShort(t *testing.T) {
	t.Parallel()

	observer := &recordingObserver{shown: &agentruntimeports.ShownArtifact{
		ID: pulid.MustNew("art_"), Kind: "table_view", Title: "Formula templates", Rows: 3,
		OpensRecords: true,
	}}
	tool := queryTool("formula_list_templates", map[string]any{"items": []any{}}, nil)
	result, _ := runWithObserver(t, observer.observe, []agenttoolports.AgentQueryTool{tool},
		runtimetest.ToolTurn("formula_list_templates", map[string]any{"query": ""}),
		runtimetest.TextTurn("Three templates."),
	)

	content := result.Messages[2].Content
	assert.Contains(t, content, "each row opens there as the record itself")
	assert.Contains(t, content, "artifact:", "the reply is given the pointer to write")
	assert.NotContains(t, content, "few enough to answer in your reply")
	assert.NotContains(t, content, "select rows there")
}

// A composed view is pointed to as a view: what it was narrowed to, not rows
// to work from where it is drawn.
func TestRun_AViewIsPointedToAsAView(t *testing.T) {
	t.Parallel()

	observer := &recordingObserver{shown: &agentruntimeports.ShownArtifact{
		ID: pulid.MustNew("art_"), Kind: "table_view", Title: "Late loads", Rows: 3,
		Actionable: true, Opens: true,
	}}
	tool := queryTool("report_compose_table_view", map[string]any{"path": "/shipments"}, nil)
	result, _ := runWithObserver(
		t,
		observer.observe,
		[]agenttoolports.AgentQueryTool{tool},
		runtimetest.ToolTurn(
			"report_compose_table_view",
			map[string]any{"entity": "shipments", "description": "late"},
		),
		runtimetest.TextTurn("Three loads are late."),
	)

	content := result.Messages[2].Content
	assert.Contains(t, content, "the person opens it as the live table")
	assert.Contains(t, content, "artifact:")
	assert.NotContains(t, content, "select rows there")
}

/*
A ranking is pointed to however short it is, and never rewritten.

A dispatcher asked for drivers ranked for a load. The Desk read five ranked
candidates and answered with its own markdown table of them, linking nothing:
the order survived, the findings behind each verdict and the link to each
driver did not. A ranked table is now one the model points to by its pick.
*/
func TestRun_ARankingIsPointedToHoweverShort(t *testing.T) {
	t.Parallel()

	observer := &recordingObserver{shown: &agentruntimeports.ShownArtifact{
		ID: pulid.MustNew("art_"), Kind: "table_view", Title: "Drivers ranked for the move",
		Rows: 5, Ranked: true,
	}}
	tool := queryTool("dispatch_rank_candidates", map[string]any{"candidates": []any{}}, nil)
	result, _ := runWithObserver(t, observer.observe, []agenttoolports.AgentQueryTool{tool},
		runtimetest.ToolTurn("dispatch_rank_candidates", map[string]any{"shipmentMoveId": "smv_1"}),
		runtimetest.TextTurn("Emily Chen is the best fit."),
	)

	content := result.Messages[2].Content
	assert.Contains(t, content, `kept as a ranked table titled "Drivers ranked for the move"`)
	assert.Contains(t, content, "however few rows it has")
	assert.Contains(t, content, "do not write its rows out as a markdown table")
	assert.Contains(t, content, "artifact:", "the reply is given the pointer to write")
	assert.NotContains(t, content, "few enough to answer in your reply")
}
