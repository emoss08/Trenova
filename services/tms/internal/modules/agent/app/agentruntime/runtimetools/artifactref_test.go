package runtimetools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/replycheck"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArtifactRef(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("aart_")
	shown := &agentruntimeports.ShownArtifact{ID: id, Kind: "table", Title: "Missing a [biller]"}

	ref := replycheck.ArtifactRef(shown)
	assert.Equal(t, "[Missing a biller](artifact:"+id.String()+")", ref)
	assert.Contains(t, ShownNote(shown), ref)
	assert.Contains(t, PublishedContent(shown), ref)
}

func TestShownNote_NeverCallsPointingOptionalForAnArtifactThatMustBeLinked(t *testing.T) {
	t.Parallel()

	shown := &agentruntimeports.ShownArtifact{
		ID:    pulid.MustNew("aart_"),
		Kind:  string(assistantartifact.KindEDIConnectionDraft),
		Title: "EDI connection for Acme Retail Group",
	}
	require.True(t, shown.MustBeLinked())

	note := ShownNote(shown)

	assert.NotContains(t, note, "only if your answer rests on it")
	assert.NotContains(t, note, "not shown")
	assert.Contains(t, note, "a connection draft titled")
	assert.Contains(t, note, "Point to it in the sentence")
	assert.Contains(t, note, replycheck.ArtifactRef(shown))
}

func TestShownNote_APagedTableIsPointedToHoweverShortItsPage(t *testing.T) {
	t.Parallel()

	shown := &agentruntimeports.ShownArtifact{
		ID:    pulid.MustNew("aart_"),
		Kind:  string(assistantartifact.KindTableView),
		Title: "Shipments",
		Rows:  5,
		Paged: true,
	}
	require.True(t, shown.MustBeLinked())

	note := ShownNote(shown)

	assert.NotContains(t, note, "few enough to answer in your reply")
	assert.NotContains(t, note, "a table you do not point to is not kept")
	assert.Contains(t, note, "the person pages through all of them there")
	assert.Contains(t, note, "a page you read joins this table rather than making a new one")
	assert.Contains(t, note, "Point to it in the sentence that mentions it")
	assert.Contains(t, note, replycheck.ArtifactRef(shown))
}

func TestShownNote_AShortTableThatIsTheWholeListMayBeAnsweredInline(t *testing.T) {
	t.Parallel()

	shown := &agentruntimeports.ShownArtifact{
		ID:    pulid.MustNew("aart_"),
		Kind:  string(assistantartifact.KindTableView),
		Title: "Shipments",
		Rows:  5,
	}
	require.False(t, shown.MustBeLinked())

	assert.Contains(t, ShownNote(shown), "few enough to answer in your reply")
}
