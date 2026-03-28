package documentintelligencejobs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyDocument_RateConfirmationCarriesSignalsAndConfidence(t *testing.T) {
	t.Parallel()

	result := classifyDocument(
		"rate-confirmation.pdf",
		"Rate Confirmation\nPickup: Dallas, TX\nDelivery: Atlanta, GA\nRate: $1,250\nEquipment: Van\nLoad #: RC12345",
	)

	assert.Equal(t, "RateConfirmation", result.Kind)
	assert.GreaterOrEqual(t, result.Confidence, 0.8)
	assert.NotEmpty(t, result.Signals)
	assert.False(t, result.ReviewRequired)
}

func TestClassifyDocument_CHRobinsonLoadConfirmationBeatsBillOfLadingHeuristics(t *testing.T) {
	t.Parallel()

	result := classifyDocument(
		"rate_confirmation_ch_robinson.pdf",
		`C.H. Robinson Contract Addendum and Carrier Load Confirmation - #123456789
Shipper Instructions
SHIPPER#1:
Receiver Instructions
RECEIVER #1:
Service for Load #123456789
Line Haul - FLAT RATE
Rate Details
Fuel Surcharge Information
THIS LOAD CONFIRMATION IS SUBJECT TO THE TERMS OF THE AGREEMENT`,
	)

	assert.Equal(t, "RateConfirmation", result.Kind)
	assert.GreaterOrEqual(t, result.Confidence, 0.8)
	assert.Contains(t, result.Signals, "rate confirmation phrase")
}

func TestAnalyzeDocument_ReturnsCanonicalFieldsAndReviewState(t *testing.T) {
	t.Parallel()

	classification := classifyDocument(
		"rate-confirmation.pdf",
		"Rate Confirmation\nShipper: ACME Foods\nConsignee: Blue Market\nPickup: ACME Foods\n123 Main St\nDallas, TX 75001\nPickup Date: 03/27/2026 08:00 AM - 10:00 AM\nDelivery: Blue Market\n500 Peachtree Rd\nAtlanta, GA 30301\nDelivery Date: 03/28/2026 01:00 PM - 03:00 PM\nRate: $1,250\nCommodity: Produce\nEquipment: Reefer\nLoad #: RC12345",
	)
	analysis := analyzeDocument(
		classification,
		&extractionResult{
			Text: "Rate Confirmation\nShipper: ACME Foods\nConsignee: Blue Market\nPickup: ACME Foods\n123 Main St\nDallas, TX 75001\nPickup Date: 03/27/2026 08:00 AM - 10:00 AM\nDelivery: Blue Market\n500 Peachtree Rd\nAtlanta, GA 30301\nDelivery Date: 03/28/2026 01:00 PM - 03:00 PM\nRate: $1,250\nCommodity: Produce\nEquipment: Reefer\nLoad #: RC12345",
			Pages: []pageExtractionResult{{
				PageNumber: 1,
				SourceKind: "native_text",
				Text:       "Rate Confirmation\nShipper: ACME Foods\nConsignee: Blue Market\nPickup: ACME Foods\n123 Main St\nDallas, TX 75001\nPickup Date: 03/27/2026 08:00 AM - 10:00 AM\nDelivery: Blue Market\n500 Peachtree Rd\nAtlanta, GA 30301\nDelivery Date: 03/28/2026 01:00 PM - 03:00 PM\nRate: $1,250\nCommodity: Produce\nEquipment: Reefer\nLoad #: RC12345",
			}},
		},
	)

	require.NotEmpty(t, analysis.Fields)
	assert.Equal(t, "RateConfirmation", analysis.Kind)
	assert.Equal(t, "Ready", analysis.ReviewStatus)
	assert.GreaterOrEqual(t, analysis.OverallConfidence, 0.8)
	assert.Empty(t, analysis.MissingFields)
	assert.Contains(t, analysis.Fields, "shipper")
	assert.Contains(t, analysis.Fields, "consignee")
	assert.Contains(t, analysis.Fields, "pickupWindow")
	assert.Contains(t, analysis.Fields, "deliveryWindow")
	assert.Contains(t, analysis.Fields, "rate")
	assert.Equal(t, "ACME Foods", analysis.Fields["shipper"].Value)
	assert.Equal(t, "$1,250", analysis.Fields["rate"].Value)
	assert.Equal(t, 1, analysis.Fields["shipper"].PageNumber)
	require.Len(t, analysis.Stops, 2)
	assert.Equal(t, "pickup", analysis.Stops[0].Role)
	assert.Equal(t, "delivery", analysis.Stops[1].Role)
	assert.Equal(t, "Dallas", analysis.Stops[0].City)
	assert.Equal(t, "Atlanta", analysis.Stops[1].City)
	assert.Empty(t, analysis.Conflicts)
}

func TestAnalyzeDocument_MarksMissingCriticalFieldsForReview(t *testing.T) {
	t.Parallel()

	classification := classifyDocument(
		"rate-confirmation.pdf",
		"Rate Confirmation\nRate: $900\nPickup: Dallas, TX\nEquipment: Van",
	)
	analysis := analyzeDocument(
		classification,
		&extractionResult{
			Text: "Rate Confirmation\nRate: $900\nPickup: Dallas, TX\nEquipment: Van",
			Pages: []pageExtractionResult{{
				PageNumber: 1,
				SourceKind: "native_text",
				Text:       "Rate Confirmation\nRate: $900\nPickup: Dallas, TX\nEquipment: Van",
			}},
		},
	)

	assert.Equal(t, "NeedsReview", analysis.ReviewStatus)
	assert.NotEmpty(t, analysis.MissingFields)
	assert.Contains(t, analysis.MissingFields, "Consignee")
	assert.Contains(t, analysis.MissingFields, "Delivery Window")
}

func TestAnalyzeDocument_FlagsConflictingRateConfirmationData(t *testing.T) {
	t.Parallel()

	classification := classifyDocument(
		"rate-confirmation.pdf",
		"Rate Confirmation\nShipper: ACME Foods\nConsignee: Blue Market\nRate: $900\nRate: $1,100\nPickup: ACME Foods\n123 Main St\nDallas, TX 75001\nPickup Date: 03/27/2026\nDelivery: Blue Market\n500 Peachtree Rd\nAtlanta, GA 30301\nDelivery Date: 03/28/2026",
	)
	analysis := analyzeDocument(
		classification,
		&extractionResult{
			Text: "Rate Confirmation\nShipper: ACME Foods\nConsignee: Blue Market\nRate: $900\nRate: $1,100\nPickup: ACME Foods\n123 Main St\nDallas, TX 75001\nPickup Date: 03/27/2026\nDelivery: Blue Market\n500 Peachtree Rd\nAtlanta, GA 30301\nDelivery Date: 03/28/2026",
			Pages: []pageExtractionResult{{
				PageNumber: 1,
				SourceKind: "native_text",
				Text:       "Rate Confirmation\nShipper: ACME Foods\nConsignee: Blue Market\nRate: $900\nRate: $1,100\nPickup: ACME Foods\n123 Main St\nDallas, TX 75001\nPickup Date: 03/27/2026\nDelivery: Blue Market\n500 Peachtree Rd\nAtlanta, GA 30301\nDelivery Date: 03/28/2026",
			}},
		},
	)

	assert.Equal(t, "NeedsReview", analysis.ReviewStatus)
	assert.NotEmpty(t, analysis.Conflicts)
	assert.True(t, analysis.Fields["rate"].Conflict)
}
