package replycheck

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripArtifactLinks(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		"Eleven have no biller: Missing a biller. See the queue.",
		StripArtifactLinks(
			"Eleven have no biller: [Missing a biller](artifact:aart_01ABC). See the queue.",
		),
	)
	assert.Equal(t,
		"Open [the docs](https://example.com) for more.",
		StripArtifactLinks("Open [the docs](https://example.com) for more."),
		"other links are left alone",
	)
}
