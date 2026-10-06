package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestArtifactRef(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("aart_")
	shown := &services.ShownArtifact{ID: id, Kind: "table", Title: "Missing a [biller]"}

	ref := ArtifactRef(shown)
	assert.Equal(t, "[Missing a biller](artifact:"+id.String()+")", ref)
	assert.Contains(t, shownNote(shown), ref)
	assert.Contains(t, publishedContent(shown), ref)
}

func TestStripArtifactLinks(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		"Eleven have no biller: Missing a biller. See the queue.",
		StripArtifactLinks("Eleven have no biller: [Missing a biller](artifact:aart_01ABC). See the queue."),
	)
	assert.Equal(t,
		"Open [the docs](https://example.com) for more.",
		StripArtifactLinks("Open [the docs](https://example.com) for more."),
		"other links are left alone",
	)
}
