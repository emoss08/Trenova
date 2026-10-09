package assistantservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/stretchr/testify/assert"
)

// A table built from a list result hands the loop its first column, so a
// reply that writes the rows out again can be recognised as a reprint.
func TestShownArtifact_CarriesATablesFirstColumn(t *testing.T) {
	t.Parallel()

	artifact := tableArtifact("call_1", "list_shipments", map[string]any{
		"columns": []any{"proNumber", "status"},
		"count":   float64(2),
		"items": []any{
			map[string]any{"proNumber": "PRO-1", "status": "Late"},
			map[string]any{"proNumber": "PRO-2", "status": "Late"},
		},
	})
	if !assert.NotNil(t, artifact) {
		return
	}

	shown := shownArtifact(artifact)

	assert.Equal(t, []string{"PRO-1", "PRO-2"}, shown.Labels)
}

// A table read back keeps its columns as decoded JSON, and a report's rows
// may be lists; both are read, and anything that is not a table has none.
func TestTableLabels_ReadsDecodedColumnsAndListRows(t *testing.T) {
	t.Parallel()

	decoded := &assistantartifact.Artifact{
		Kind: assistantartifact.KindTableView,
		Payload: map[string]any{
			"columns": []any{map[string]any{"key": "customer"}},
			"rows":    []any{map[string]any{"customer": "Acme"}, map[string]any{"customer": ""}},
		},
	}
	assert.Equal(t, []string{"Acme"}, tableLabels(decoded))

	report := &assistantartifact.Artifact{
		Kind: assistantartifact.KindReportPreview,
		Payload: map[string]any{
			"rows": []any{[]any{"Dallas", float64(3)}, []any{float64(42)}},
		},
	}
	assert.Equal(t, []string{"Dallas", "42"}, tableLabels(report))

	card := &assistantartifact.Artifact{
		Kind:    assistantartifact.KindEntityCard,
		Payload: map[string]any{"rows": []any{map[string]any{"a": "b"}}},
	}
	assert.Nil(t, tableLabels(card))
}
