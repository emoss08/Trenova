package replycheck

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/stretchr/testify/assert"
)

func TestFiledLabels_ComeFromTheToolsSchema(t *testing.T) {
	t.Parallel()

	labels := schemaLabels(runtimetest.GroundedShipmentSchema())

	assert.Contains(t, labels, schemaLabel{path: "commodities", words: "commodities"})
	assert.Contains(t, labels, schemaLabel{path: "additionalCharges", words: "additional charges"})
	assert.Contains(t, labels, schemaLabel{path: "baseRate", words: "base rate"})
	for _, label := range labels {
		assert.NotEqual(t, "customerId", label.path, "an id is not something a reply claims")
		assert.NotEqual(t, "commodities[].pieces", label.path,
			"a single word scalar is too common to hold a reply to")
	}
}

func TestFiledLabels_MarkANestedOneWordCollection(t *testing.T) {
	t.Parallel()

	labels := schemaLabels(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"requirements": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"codes": map[string]any{
							"type":  "array",
							"items": map[string]any{"type": "string"},
						},
						"allowedCodeLists": map[string]any{
							"type":  "array",
							"items": map[string]any{"type": "string"},
						},
					},
				},
			},
		},
	})

	assert.Contains(t, labels, schemaLabel{path: "requirements", words: "requirements"})
	assert.Contains(t, labels,
		schemaLabel{path: "requirements[].allowedCodeLists", words: "allowed code lists"})
	assert.Contains(t, labels,
		schemaLabel{path: "requirements[].codes", words: "codes", nestedWord: true})
}

func TestGroundingFinding_DropsOnlyWordsNoTopLevelLabelClaims(t *testing.T) {
	t.Parallel()

	finding := GroundingFinding{
		fields: []string{"codes", "commodities"},
		tools: map[string][]string{
			"codes":       {"edi_apply_connection_draft"},
			"commodities": {"shipment_create"},
		},
		nested: []string{"codes"},
	}
	finding.dropNestedWords()

	assert.Equal(t, []string{"commodities"}, finding.fields)
	assert.NotContains(t, finding.tools, "codes")
	assert.False(t, finding.empty())
}
