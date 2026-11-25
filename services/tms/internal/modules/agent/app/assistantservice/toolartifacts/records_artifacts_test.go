package toolartifacts

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactpayload"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArtifactFromObservation_RecordsQueryBecomesTheEntityTable(t *testing.T) {
	t.Parallel()

	observation := assistanttest.Observation("records_query", map[string]any{
		"entity":  "worker",
		"columns": []any{"id", "wholeName", "status"},
		"items": []any{
			map[string]any{"id": "wrk_01M85FQVQHDSXM0T2KTR3G5X2W", "wholeName": "Kwame Boateng", "status": "Active"},
			map[string]any{"id": "wrk_01M85FQVQEGTMAE4W4383RJEJB", "wholeName": "Rosa Lind", "status": "Active"},
		},
		"total":      414,
		"hasMore":    true,
		"nextCursor": "c2",
	})
	observation.Call.Arguments = map[string]any{"entity": "worker", "limit": 100}

	artifact := ArtifactFromObservation(observation)

	require.NotNil(t, artifact)
	assert.Equal(t, assistantartifact.KindTableView, artifact.Kind)
	assert.Equal(t, "records_query", artifact.Payload[artifactpayload.PayloadTool])
	assert.Equal(t, "workers", artifact.Payload[artifactpayload.PayloadEntity])
	assert.Equal(t, 2, artifact.Payload[artifactpayload.PayloadRowCount])
	assert.Equal(t, true, artifact.Payload[artifactpayload.PayloadHasMore])
	assert.NotEqual(t, "Table", artifact.Title)
}

func TestArtifactFromObservation_RecordGetBecomesTheRecordCard(t *testing.T) {
	t.Parallel()

	observation := assistanttest.Observation("record_get", map[string]any{
		"entity":  "worker",
		"columns": []any{"id", "wholeName", "status"},
		"items": []any{
			map[string]any{"id": "wrk_01M85FQVQHDSXM0T2KTR3G5X2W", "wholeName": "Kwame Boateng", "status": "Active"},
		},
	})
	observation.Call.Arguments = map[string]any{"entity": "worker", "ids": []any{"wrk_01M85FQVQHDSXM0T2KTR3G5X2W"}}

	artifact := ArtifactFromObservation(observation)

	require.NotNil(t, artifact)
	assert.Equal(t, assistantartifact.KindEntityCard, artifact.Kind)
	assert.Equal(t, "wrk_01M85FQVQHDSXM0T2KTR3G5X2W", artifact.Payload[PayloadRecordID])
}

func TestArtifactFromObservation_RecordsQueryWithoutAnEntityShowsNothing(t *testing.T) {
	t.Parallel()

	assert.Nil(t, ArtifactFromObservation(assistanttest.Observation("records_query", map[string]any{
		"entity": "nothing", "columns": []any{"id"}, "items": []any{map[string]any{"id": "x"}},
	})))
}
