package board_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/stretchr/testify/assert"
)

func TestSegmentsSplitsAnArtifactLinkFromTheWords(t *testing.T) {
	t.Parallel()

	segments := board.Segments("Ridgeline's cover ends in {left}; I drafted " +
		"[a request for its renewed certificate](artifact:art_01JDRAFT0000000000000001).")

	assert.Equal(t, []board.Segment{
		{Text: "Ridgeline's cover ends in "},
		{Reading: "left"},
		{Text: "; I drafted "},
		{Text: "a request for its renewed certificate", Artifact: "art_01JDRAFT0000000000000001"},
		{Text: "."},
	}, segments)
}

func TestArtifactLinksAreTheBoardsLinkedArtifactsOnce(t *testing.T) {
	t.Parallel()

	layout := board.Board{
		Verdict: "See [the draft](artifact:art_A) and [it again](artifact:art_A).",
		Blocks: []board.Block{
			{Type: board.BlockProse, Text: "Also [the letter](artifact:art_B)."},
			{Type: board.BlockRecords, Caption: "Sent with [the draft](artifact:art_A)."},
		},
	}

	assert.Equal(t, []string{"art_A", "art_B"}, layout.ArtifactLinks())
}

func TestFiguresDoNotReadAnArtifactLinksID(t *testing.T) {
	t.Parallel()

	assert.Empty(t, board.Figures("I drafted [a request](artifact:art_01JDRAFT0000000000000001)."))
	assert.Equal(t, "Wait the draft.", board.Unlinked("Wait [the draft](artifact:art_1)."))
}
