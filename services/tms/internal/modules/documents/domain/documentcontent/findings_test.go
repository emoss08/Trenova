package documentcontent_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindings_ReadsWhatTheAnalysisFoundAndNothingElse(t *testing.T) {
	t.Parallel()

	content := &documentcontent.Content{StructuredData: map[string]any{
		"schemaVersion": 6,
		"intelligence": map[string]any{
			"kind":         "RateConfirmation",
			"reviewStatus": "NeedsReview",
			"rawExcerpt":   "everything again",
			"fields": map[string]any{
				"shipper": map[string]any{
					"label": "Shipper", "value": "BEX Logistics", "confidence": 0.88,
					"evidenceExcerpt": "SHIPPER#1:\nBEX Logistics", "pageNumber": 1,
				},
				"rate":  map[string]any{"label": "Rate", "value": "$2,481.00", "pageNumber": 3},
				"empty": map[string]any{"label": "Empty", "value": ""},
			},
			"stops": []any{map[string]any{
				"sequence": 1, "role": "pickup", "name": "BEX Logistics",
				"city": "Columbus", "state": "OH", "date": "08/02/22",
				"evidenceExcerpt": "a long excerpt",
			}},
			"missingFields": []any{},
			"conflicts": []any{map[string]any{
				"label": "Delivery Date", "values": []any{"08/03/22", "08/04/22"},
				"evidenceExcerpt": "a long excerpt",
			}},
		},
		"aiDiagnostics": map[string]any{"fallbackAnalysis": map[string]any{"rawExcerpt": "x"}},
	}}

	findings := content.Findings()

	require.NotNil(t, findings)
	assert.Equal(t, "RateConfirmation", findings.Kind)
	assert.Equal(t, []documentcontent.FoundField{
		{Key: "rate", Label: "Rate", Value: "$2,481.00", PageNumber: 3},
		{Key: "shipper", Label: "Shipper", Value: "BEX Logistics", Confidence: 0.88, PageNumber: 1},
	}, findings.Fields)
	require.Len(t, findings.Stops, 1)
	assert.Equal(t, "Columbus", findings.Stops[0].City)
	require.Len(t, findings.Conflicts, 1)
	assert.Equal(t, []string{"08/03/22", "08/04/22"}, findings.Conflicts[0].Values)
}

func TestFindings_NothingAnalyzedIsNil(t *testing.T) {
	t.Parallel()

	assert.Nil(t, (&documentcontent.Content{}).Findings())
	assert.Nil(t, (&documentcontent.Content{StructuredData: map[string]any{
		"intelligence": map[string]any{},
	}}).Findings())
}
