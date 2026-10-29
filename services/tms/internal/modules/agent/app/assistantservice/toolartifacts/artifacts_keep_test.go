package toolartifacts

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/stretchr/testify/assert"
)

// A table the reply was told to point to rather than reprint stays whether
// or not the reply managed the link: the reader draws a badge for it. A
// short table and a card not pointed to are still dropped.
func TestMustBePointedTo_KeepsLongRankedAndActionableTables(t *testing.T) {
	t.Parallel()

	table := func(payload map[string]any) *assistantartifact.Artifact {
		return &assistantartifact.Artifact{Kind: assistantartifact.KindTableView, Payload: payload}
	}

	assert.True(t, MustBePointedTo(table(map[string]any{"rowCount": 40})), "a long table")
	assert.True(
		t,
		MustBePointedTo(table(map[string]any{"rowCount": 5, payloadRanked: true})),
		"a ranking",
	)
	assert.True(
		t,
		MustBePointedTo(table(map[string]any{"rowCount": 3, "path": "/billing/queue"})),
		"a live view",
	)
	assert.False(t, MustBePointedTo(table(map[string]any{"rowCount": 5})), "a short table")
	assert.False(t, MustBePointedTo(&assistantartifact.Artifact{
		Kind: assistantartifact.KindEntityCard, Payload: map[string]any{},
	}), "a card")
}
