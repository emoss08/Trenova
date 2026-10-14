package aianalyticsrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/stretchr/testify/assert"
)

func TestOffsetSQLPicksTheOffsetInForceWhenTheRunFinished(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "0", offsetSQL(nil))
	assert.Equal(t, "-18000", offsetSQL([]aianalyticsports.OffsetSpan{{From: 1, Offset: -18000}}))
	assert.Equal(
		t,
		"CASE WHEN w.finished_at >= 300 THEN -18000 WHEN w.finished_at >= 200 THEN -21600 ELSE -18000 END",
		offsetSQL([]aianalyticsports.OffsetSpan{
			{From: 100, Offset: -18000},
			{From: 200, Offset: -21600},
			{From: 300, Offset: -18000},
		}),
	)
}
