package agentruntime

import (
	"testing"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingObserver struct {
	seen  []serviceports.ToolObservation
	shown *serviceports.ShownArtifact
	err   error
}

func (o *recordingObserver) observe(
	observation serviceports.ToolObservation,
) (*serviceports.ShownArtifact, error) {
	o.seen = append(o.seen, observation)

	return o.shown, o.err
}

func offeredNames(specs []serviceports.ToolSpec) []string {
	names := make([]string, 0, len(specs))
	for _, spec := range specs {
		names = append(names, spec.Name)
	}

	return names
}

func runWithObserver(
	t *testing.T,
	observer serviceports.ToolObserver,
	tools []serviceports.AgentQueryTool,
	turns ...*serviceports.ChatCompletionResult,
) (*serviceports.RunResult, *scriptedCompletion) {
	t.Helper()

	completion := &scriptedCompletion{Turns: turns}
	rt := newRuntime(completion, &stubQueryRegistry{Tools: tools}, &stubActionRegistry{}, nil)
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name())
	}

	result, err := rt.Run(t.Context(), &serviceports.RunRequest{
		Definition:   testDefinition(names...),
		Actor:        testActor(),
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
	_, completion := runWithObserver(t, observer.observe, nil, textTurn("Here it is."))

	assert.Contains(t, offeredNames(completion.LastReq.Tools), publishArtifactName)
	assert.Contains(t, completion.LastReq.System, "## Artifacts")
	assert.Contains(t, completion.LastReq.System, "publish_artifact")
}

// A run with nowhere to put a document is not offered one, and is not told
// about artifacts it cannot see.
func TestRun_DoesNotOfferPublishingWithoutAPane(t *testing.T) {
	t.Parallel()

	_, completion := runWithObserver(t, nil, nil,
		toolTurn(publishArtifactName, map[string]any{"title": "Brief", "body": "# Brief"}),
		textTurn("Here it is."),
	)

	assert.NotContains(t, offeredNames(completion.LastReq.Tools), publishArtifactName)
	assert.NotContains(t, completion.LastReq.System, "## Artifacts")
	last := completion.LastReq.Messages[len(completion.LastReq.Messages)-1]
	assert.True(t, last.IsError)
	assert.Contains(t, last.Content, "nowhere to publish")
}

// A published document reaches the observer whole, and the model is told the
// id it can revise it by.
func TestRun_PublishesADocumentAndReturnsItsID(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("art_")
	observer := &recordingObserver{shown: &serviceports.ShownArtifact{
		ID: id, Kind: "document", Title: "SEED-SHP-007 brief",
	}}
	result, _ := runWithObserver(t, observer.observe, nil,
		toolTurn(publishArtifactName, map[string]any{
			"title": "SEED-SHP-007 brief",
			"body":  "# SEED-SHP-007\n\nRange Logistics, Chicago to Denver.",
		}),
		textTurn("The brief is open beside the conversation."),
	)

	require.Len(t, observer.seen, 1)
	document, ok := observer.seen[0].Data.(serviceports.PublishedDocument)
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
	result, _ := runWithObserver(t, observer.observe, nil,
		toolTurn(publishArtifactName, map[string]any{"title": "Brief", "body": "# Brief"}),
		textTurn("Here it is."),
	)

	assert.True(t, result.Messages[2].ToolFailed)
	assert.Contains(t, result.Messages[2].Content, "Put the text in your reply")
}

// A publish with nothing in it is refused before anything is kept.
func TestRun_RefusesAnEmptyDocument(t *testing.T) {
	t.Parallel()

	observer := &recordingObserver{}
	result, _ := runWithObserver(t, observer.observe, nil,
		toolTurn(publishArtifactName, map[string]any{"title": "", "body": ""}),
		textTurn("Sorry."),
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

	observer := &recordingObserver{shown: &serviceports.ShownArtifact{
		ID: pulid.MustNew("art_"), Kind: "table_view", Title: "Shipments",
	}}
	tool := queryTool("search_shipments", map[string]any{"items": []any{}}, nil)
	result, _ := runWithObserver(t, observer.observe, []serviceports.AgentQueryTool{tool},
		toolTurn("search_shipments", map[string]any{"query": ""}),
		textTurn("Twenty-five shipments; two are delayed."),
	)

	content := result.Messages[2].Content
	assert.Contains(t, content, `Shown to the person as a table titled "Shipments"`)
	_, payload, fenced := UnfenceToolResult(content)
	require.True(t, fenced, "the note sits after the fence and does not break reading it back")
	assert.Contains(t, payload, `"items"`)
}
