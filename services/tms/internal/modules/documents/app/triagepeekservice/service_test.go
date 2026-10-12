package triagepeekservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadingOfFindings(t *testing.T) {
	t.Parallel()

	content := &documentcontent.Content{
		Status:                   documentcontent.StatusExtracted,
		DetectedDocumentKind:     "Other",
		ClassificationConfidence: 0.92,
		PageCount:                2,
		StructuredData: map[string]any{
			"intelligence": map[string]any{
				"kind": "ProofOfDelivery",
				"fields": map[string]any{
					"consignee_signature": map[string]any{"value": "J. Ruiz", "reviewRequired": true},
					"pro_number":          map[string]any{"label": "PRO", "value": "S-104251"},
					"empty":               map[string]any{"value": " "},
				},
			},
		},
	}

	reading := ReadingOf(content)
	require.NotNil(t, reading)
	assert.Equal(t, "Proof of delivery", reading.DocumentType)
	assert.InDelta(t, 0.92, reading.Confidence, 0.0001)
	assert.Equal(t, 2, reading.PageCount)
	require.Len(t, reading.Fields, 2)
	assert.Equal(t, "Consignee signature", reading.Fields[0].Label)
	assert.Equal(t, inboundmessage.FactToneWarning, reading.Fields[0].Tone)
	assert.Equal(t, "PRO", reading.Fields[1].Label)
	assert.Equal(t, inboundmessage.FactToneNeutral, reading.Fields[1].Tone)
}

func TestReadingOfUnsettledOrEmpty(t *testing.T) {
	t.Parallel()

	assert.Nil(t, ReadingOf(nil))
	assert.Nil(t, ReadingOf(&documentcontent.Content{Status: documentcontent.StatusPending, DetectedDocumentKind: "Invoice"}))
	assert.Nil(t, ReadingOf(&documentcontent.Content{Status: documentcontent.StatusFailed, DetectedDocumentKind: "Invoice"}))
	assert.Nil(t, ReadingOf(&documentcontent.Content{Status: documentcontent.StatusExtracted}))

	reading := ReadingOf(&documentcontent.Content{
		Status:               documentcontent.StatusExtracted,
		DetectedDocumentKind: "RateConfirmation",
	})
	require.NotNil(t, reading)
	assert.Equal(t, "Rate confirmation", reading.DocumentType)
	assert.Empty(t, reading.Fields)
}

func TestHumanizeKind(t *testing.T) {
	t.Parallel()

	assert.Empty(t, HumanizeKind("  "))
	assert.Equal(t, "Bill of lading", HumanizeKind("BillOfLading"))
	assert.Equal(t, "Bill of lading", HumanizeKind("bill_of_lading"))
}
