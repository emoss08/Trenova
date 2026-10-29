package toolartifacts

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArtifactFromObservation_ABatchGetIsATable(t *testing.T) {
	t.Parallel()

	artifact := ArtifactFromObservation(assistanttest.Observation("invoice_get", map[string]any{
		"count":   2,
		"columns": []any{"id", "number", "status"},
		"items": []any{
			map[string]any{
				"id":     pulid.MustNew("inv_").String(),
				"number": "INV-1",
				"status": "Draft",
			},
			map[string]any{
				"id":     pulid.MustNew("inv_").String(),
				"number": "INV-2",
				"status": "Draft",
			},
		},
	}))

	require.NotNil(t, artifact)
	assert.Equal(t, assistantartifact.KindTableView, artifact.Kind)
	assert.Equal(t, "invoices", artifact.Payload["entity"])
	assert.Len(t, artifact.Payload["rows"], 2)
}
