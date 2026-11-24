package boardmarkdown_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardmarkdown"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/stretchr/testify/assert"
)

func TestAnswerKeepsALinkToAnArtifactTheTurnMade(t *testing.T) {
	t.Parallel()

	days := boardcatalog.Days(-8)
	filled := &board.Filled{
		Verdict: []board.FilledSegment{
			{Text: "Jane Doe's card lapsed "},
			{Reading: "lapsed"},
			{Text: " ago; I wrote "},
			{Text: "a reminder", Artifact: "art_01JDRAFT0000000000000001"},
			{Text: "."},
		},
		Readings: []board.FilledReading{{Key: "lapsed", Value: &days}},
	}

	assert.Equal(t, "Jane Doe's card lapsed 8 days ago; I wrote "+
		"[a reminder](artifact:art_01JDRAFT0000000000000001).", boardmarkdown.Answer(filled))
}
