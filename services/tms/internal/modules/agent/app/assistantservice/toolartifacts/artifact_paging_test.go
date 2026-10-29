package toolartifacts

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactpayload"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTableArtifact_KeepsWhereTheListGoesOn(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		result     map[string]any
		hasMore    bool
		nextCursor any
		nextOffset any
		total      any
		estimated  any
	}{
		"a cursor list with more": {
			result: assistanttest.QueuePage(
				0,
				2,
				map[string]any{"hasMore": true, "nextCursor": "c-2"},
			),
			hasMore:    true,
			nextCursor: "c-2",
		},
		"an offset list with more and a total": {
			result: assistanttest.QueuePage(0, 2, map[string]any{
				"hasMore": true, "nextOffset": float64(2), "total": float64(9),
			}),
			hasMore:    true,
			nextOffset: 2,
			total:      9,
		},
		"a list with an estimated total": {
			result: assistanttest.QueuePage(0, 2, map[string]any{
				"hasMore": true, "nextCursor": "c-2", "total": float64(1_250_000),
				"totalEstimated": true,
			}),
			hasMore:    true,
			nextCursor: "c-2",
			total:      1_250_000,
			estimated:  true,
		},
		"the whole list": {
			result: assistanttest.QueuePage(0, 2, map[string]any{"hasMore": false}),
		},
		"a page with no word on what follows": {
			result: assistanttest.QueuePage(0, 2, nil),
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			artifact := tableArtifact("call_1", assistanttest.PagedTool, tc.result)
			require.NotNil(t, artifact)
			assert.Equal(t, tc.hasMore, artifact.Payload[artifactpayload.PayloadHasMore])
			assert.Equal(t, tc.nextCursor, artifact.Payload[artifactpayload.PayloadNextCursor])
			assert.Equal(t, tc.nextOffset, artifact.Payload[artifactpayload.PayloadNextOffset])
			assert.Equal(t, tc.total, artifact.Payload[artifactpayload.PayloadTotal])
			assert.Equal(t, tc.estimated, artifact.Payload[artifactpayload.PayloadTotalEstimated])
			assert.Equal(t, 1, artifact.Payload[artifactpayload.PayloadPages])
			assert.Equal(t, tc.hasMore, artifactpayload.PagedTable(artifact))
		})
	}
}

func TestTableArtifact_ATableCutToFitHasMoreButNoPlaceToGoOnFrom(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("x", assistantartifact.MaxPayloadBytes/10)
	items := make([]any, 0, 20)
	for n := range 20 {
		item := assistanttest.QueueItem(n)
		item["status"] = long
		items = append(items, item)
	}
	result := map[string]any{
		"items":      items,
		"columns":    []any{"number", "status"},
		"count":      float64(20),
		"hasMore":    false,
		"nextCursor": "",
	}

	artifact := tableArtifact("call_1", assistanttest.PagedTool, result)
	require.NotNil(t, artifact)
	assert.Equal(t, true, artifact.Payload[artifactpayload.PayloadTruncated])
	assert.Equal(t, true, artifact.Payload[artifactpayload.PayloadHasMore])
	assert.NotContains(t, artifact.Payload, artifactpayload.PayloadNextCursor)
	assert.NotContains(t, artifact.Payload, artifactpayload.PayloadNextOffset)
	assert.True(t, artifactpayload.PagedTable(artifact))
}
