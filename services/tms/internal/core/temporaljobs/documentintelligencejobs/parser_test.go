package documentintelligencejobs

import (
	"testing"

	services "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
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

func TestClassifyDocument_InvoiceLikeUploadFallsBackToOther(t *testing.T) {
	t.Parallel()

	result := classifyDocument(
		"customer_invoice.pdf",
		`INVOICE
Invoice Number: INV-10239
Invoice Date: 03/28/2026
Due Date: 04/15/2026
Amount Due: $1,240.00`,
	)

	assert.Equal(t, "Other", result.Kind)
	assert.True(t, result.ReviewRequired)
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

func TestAnalyzeDocument_ExtractsBillOfLadingFieldsFromSectionBlocks(t *testing.T) {
	t.Parallel()

	text := `BILL OF LADING
BOL #: BOL123456
SHIP FROM
ACME Foods
123 Main St
Dallas, TX 75001

SHIP TO
Blue Market
500 Peachtree Rd
Atlanta, GA 30301

COMMODITY
Frozen Produce
Pieces: 18
Weight: 42,000 lbs`

	analysis := analyzeDocument(
		classificationResult{
			Kind:           "BillOfLading",
			Confidence:     0.9,
			Signals:        []string{"bill of lading"},
			Source:         "deterministic",
			Reason:         "header matched BOL",
			ReviewRequired: false,
		},
		&extractionResult{
			Text: text,
			Pages: []pageExtractionResult{{
				PageNumber: 1,
				SourceKind: "native_text",
				Text:       text,
			}},
		},
	)

	assert.Equal(t, "BillOfLading", analysis.Kind)
	assert.Equal(t, "Ready", analysis.ReviewStatus)
	assert.Equal(t, "ACME Foods", analysis.Fields["shipper"].Value)
	assert.Equal(t, "Blue Market", analysis.Fields["consignee"].Value)
	assert.Equal(t, "BOL123456", analysis.Fields["referenceNumber"].Value)
	assert.Equal(t, "Frozen Produce", analysis.Fields["commodity"].Value)
	assert.Equal(t, "18", analysis.Fields["pieceCount"].Value)
	assert.Equal(t, "42,000 lbs", analysis.Fields["weight"].Value)
	assert.Empty(t, analysis.Conflicts)
}

func TestAnalyzeDocument_ExtractsProofOfDeliveryFieldsFromSectionBlocks(t *testing.T) {
	t.Parallel()

	text := `PROOF OF DELIVERY
Reference #: POD99881
DELIVERED TO
Blue Market Receiving
500 Peachtree Rd
Atlanta, GA 30301

DELIVERY DATE
03/28/2026 01:15 PM

RECEIVER SIGNATURE
Jane Smith

REMARKS
Received in good order`

	analysis := analyzeDocument(
		classificationResult{
			Kind:           "ProofOfDelivery",
			Confidence:     0.9,
			Signals:        []string{"proof of delivery"},
			Source:         "deterministic",
			Reason:         "header matched POD",
			ReviewRequired: false,
		},
		&extractionResult{
			Text: text,
			Pages: []pageExtractionResult{{
				PageNumber: 1,
				SourceKind: "native_text",
				Text:       text,
			}},
		},
	)

	assert.Equal(t, "ProofOfDelivery", analysis.Kind)
	assert.Equal(t, "Ready", analysis.ReviewStatus)
	assert.Equal(t, "Blue Market Receiving", analysis.Fields["consignee"].Value)
	assert.Equal(t, "03/28/2026", analysis.Fields["deliveryWindow"].Value)
	assert.Equal(t, "Jane Smith", analysis.Fields["signature"].Value)
	assert.Equal(t, "POD99881", analysis.Fields["referenceNumber"].Value)
	assert.Equal(t, "Received in good order", analysis.Fields["receiptNotes"].Value)
	assert.Empty(t, analysis.MissingFields)
}

func TestBuildStructuredDataIncludesParsingRuleMetadata(t *testing.T) {
	t.Parallel()

	intelligence := documentIntelligenceAnalysis{
		Kind:          "RateConfirmation",
		ReviewStatus:  "Ready",
		MissingFields: []string{},
		Signals:       []string{"rule:generic"},
		Fields:        map[string]reviewField{},
		Stops:         []intelligenceStop{},
		Conflicts:     []reviewConflict{},
		ParsingRuleMetadata: &services.DocumentParsingRuleMetadata{
			RuleSetID:        pulid.MustNew("dprs_"),
			RuleSetName:      "Generic",
			RuleVersionID:    pulid.MustNew("dprv_"),
			VersionNumber:    2,
			ParserMode:       "merge_with_base",
			ProviderMatched:  "GenericBroker",
			MatchSpecificity: 220,
		},
	}

	data := buildStructuredData(intelligence)
	require.Equal(t, 5, data["schemaVersion"])

	intelligenceMap, ok := data["intelligence"].(map[string]any)
	require.True(t, ok)
	metadata, ok := intelligenceMap["parsingRuleMetadata"].(*services.DocumentParsingRuleMetadata)
	require.True(t, ok)
	require.Equal(t, "Generic", metadata.RuleSetName)
}

func TestAnalyzeDocument_ProofOfDeliveryMissingSignatureStaysNeedsReview(t *testing.T) {
	t.Parallel()

	text := `PROOF OF DELIVERY
Reference #: POD99881
DELIVERED TO
Blue Market Receiving
500 Peachtree Rd
Atlanta, GA 30301

DELIVERY DATE
03/28/2026 01:15 PM`

	analysis := analyzeDocument(
		classificationResult{
			Kind:           "ProofOfDelivery",
			Confidence:     0.88,
			Signals:        []string{"proof of delivery"},
			Source:         "deterministic",
			Reason:         "header matched POD",
			ReviewRequired: false,
		},
		&extractionResult{
			Text: text,
			Pages: []pageExtractionResult{{
				PageNumber: 1,
				SourceKind: "native_text",
				Text:       text,
			}},
		},
	)

	assert.Equal(t, "NeedsReview", analysis.ReviewStatus)
	assert.Contains(t, analysis.MissingFields, "Signature")
}
