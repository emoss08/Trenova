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

func TestExtractRateConfirmationStops_CHRobinsonStyleSectionBlocks(t *testing.T) {
	t.Parallel()

	pages := []pageExtractionResult{
		{
			PageNumber: 1,
			SourceKind: "native_text",
			Text: `SHIPPER#1:
Pick Up Date:

Pick Up Time:
Pickup#:
Appointment#:

Schmidt
123456
Van - Min L=53
123456

7/13/21
7/14/21
Anyco Clothes #425

7/12/21

04:00 Appt.

Anyco Clothes #176
1234
Main Drive
Houston TX 78705

(800) 123-1234

RECEIVER #4:
Delivery Date:

7/15/21
*Scheduled Delivery*
Delivery Time:

11:00-22:00
Delivery#:
Appointment#:

Anyco Clothes #255
123 S. 2nd BLVD, STE 100
Denver, CO 80014

(800) 555-5555`,
		},
	}

	stops := extractRateConfirmationStops(pages)
	require.Len(t, stops, 2)

	assert.Equal(t, "pickup", stops[0].Role)
	assert.Equal(t, "Anyco Clothes #176", stops[0].Name)
	assert.Equal(t, "1234 Main Drive", stops[0].AddressLine1)
	assert.Equal(t, "Houston", stops[0].City)
	assert.Equal(t, "TX", stops[0].State)
	assert.Equal(t, "78705", stops[0].PostalCode)
	assert.Equal(t, "7/12/21", stops[0].Date)
	assert.Equal(t, "04:00 Appt", stops[0].TimeWindow)
	assert.True(t, stops[0].AppointmentRequired)

	assert.Equal(t, "delivery", stops[1].Role)
	assert.Equal(t, "Anyco Clothes #255", stops[1].Name)
	assert.Equal(t, "123 S. 2nd BLVD, STE 100", stops[1].AddressLine1)
	assert.Equal(t, "Denver", stops[1].City)
	assert.Equal(t, "CO", stops[1].State)
	assert.Equal(t, "80014", stops[1].PostalCode)
	assert.Equal(t, "7/15/21", stops[1].Date)
	assert.Equal(t, "11:00-22:00", stops[1].TimeWindow)

	for _, stop := range stops {
		assert.NotEqual(t, "Appointment#:", stop.Name)
		assert.NotEqual(t, "Delivery#:", stop.Name)
		assert.NotEqual(t, "Pick Up Time:", stop.Name)
	}
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

	data := buildStructuredData(intelligence, aiExtractionDiagnostics{
		FallbackAnalysis: intelligence,
		AcceptanceStatus: aiAcceptanceStatusRejected,
		RejectionReason:  "ai_candidate_missing_pickup_stop",
	})
	require.Equal(t, 6, data["schemaVersion"])

	intelligenceMap, ok := data["intelligence"].(map[string]any)
	require.True(t, ok)
	metadata, ok := intelligenceMap["parsingRuleMetadata"].(*services.DocumentParsingRuleMetadata)
	require.True(t, ok)
	require.Equal(t, "Generic", metadata.RuleSetName)

	aiDiagnosticsMap, ok := data["aiDiagnostics"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, aiAcceptanceStatusRejected, aiDiagnosticsMap["acceptanceStatus"])
	require.Equal(t, "ai_candidate_missing_pickup_stop", aiDiagnosticsMap["rejectionReason"])
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

func TestMergeAIAnalysis_NormalizesEquivalentAIExtractionShape(t *testing.T) {
	t.Parallel()

	fallback := documentIntelligenceAnalysis{
		Kind:                 "RateConfirmation",
		OverallConfidence:    0.61,
		ReviewStatus:         "NeedsReview",
		ClassifierSource:     "ai-route",
		ProviderFingerprint:  "provider=CHRobinson",
		ClassificationReason: "AI route matched rate confirmation",
		Fields:               map[string]reviewField{},
		Stops:                []intelligenceStop{},
		Conflicts:            []reviewConflict{},
	}

	aiExtract := &services.AIExtractResult{
		DocumentKind:      "rate_confirmation",
		OverallConfidence: 0.72,
		ReviewStatus:      "REVIEW_REQUIRED",
		Fields: map[string]services.AIDocumentField{
			"totalRate": {
				Label:           "Total Rate",
				Value:           "4500.00",
				Confidence:      0.95,
				EvidenceExcerpt: "$4,500.00",
				PageNumber:      3,
				Source:          "page-3",
			},
		},
		Stops: []services.AIDocumentStop{
			{
				Sequence:        1,
				Role:            "origin",
				Name:            "Anyco Clothes #176",
				AddressLine1:    "1234 Main Drive",
				City:            "Houston",
				State:           "TX",
				PostalCode:      "78705",
				PageNumber:      1,
				EvidenceExcerpt: "Anyco Clothes #176\n1234 Main Drive\nHouston, TX 78705",
				Confidence:      0.94,
				Source:          "page-1",
			},
			{
				Sequence:        2,
				Role:            "destination",
				Name:            "Anyco Clothes #255",
				AddressLine1:    "123 S. 2nd BLVD, STE 100",
				City:            "Denver",
				State:           "CO",
				PostalCode:      "80014",
				PageNumber:      2,
				EvidenceExcerpt: "Anyco Clothes #255\n123 S. 2nd BLVD, STE 100\nDenver, CO 80014",
				Confidence:      0.92,
				Source:          "page-2",
			},
		},
	}

	merged, ok, rejectionReason := mergeAIAnalysis(fallback, aiExtract)
	require.True(t, ok)
	assert.Empty(t, rejectionReason)
	assert.Equal(t, "4500.00", merged.Fields["rate"].Value)
	assert.Equal(t, "Anyco Clothes #176", merged.Fields["shipper"].Value)
	assert.Equal(t, "Anyco Clothes #255", merged.Fields["consignee"].Value)
	require.Len(t, merged.Stops, 2)
	assert.Equal(t, "pickup", merged.Stops[0].Role)
	assert.Equal(t, "delivery", merged.Stops[1].Role)
}
