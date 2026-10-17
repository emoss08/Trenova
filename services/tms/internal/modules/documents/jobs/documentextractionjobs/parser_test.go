package documentextractionjobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentparsingruleports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractRateConfirmationStops_CHRobinsonStyleSectionBlocks(t *testing.T) {
	t.Parallel()

	pages := []*PageExtractionResult{
		{
			PageNumber: 1,
			SourceKind: documentcontent.SourceKindNative,
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

func TestAnalyzeDocument_ReadsARateConfirmationLaidOutInRows(t *testing.T) {
	t.Parallel()

	extracted := finalizeExtraction([]*PageExtractionResult{{
		PageNumber: 1,
		SourceKind: documentcontent.SourceKindNative,
		Text: "Customer-Specified Equipment Requirements\n" +
			"Driver Name:   carlos del said   Trailer #:   10\n" +
			"Equipment:   Van - Min L=53\n\n" +
			"SHIPPER#1:   BEX Logistics-OH Consolidation   Pick Up Date:   08/02/22\n" +
			"Address:   3685 Alum Creek Dr   *Open Pick Up*\n" +
			"STE 2040   Pick Up Time:   08:00-18:00\n" +
			"Columbus, OH 43207   Pickup#:   405515639\n" +
			"Phone:   (614) 305-7811   Appointment#:\n\n" +
			"RECEIVER #1:   Freight, Land, Sky   Delivery Date:   08/03/22\n" +
			"Address:   1000 Sycamore St.   *Scheduled Delivery*\n" +
			"HARRISBURG, PA 17104   Delivery Time:   12:00 Appt.\n" +
			"Delivery#:\n" +
			"Phone:   (707) 236-6700   Appointment#:\n\n" +
			"Rate Details\n" +
			"Line Haul - FLAT RATE   1   $1,800.00   $1,800.00\n" +
			"Total:   $2,481.00",
	}}, 200_000)

	analysis := analyzeDocument(
		&ClassificationResult{Kind: kindRateConfirmation, Confidence: 0.99},
		extracted,
	)

	assert.Empty(t, analysis.MissingFields)
	require.Len(t, analysis.Stops, 2)
	pickup, delivery := analysis.Stops[0], analysis.Stops[1]
	assert.Equal(t, stopRolePickup, pickup.Role)
	assert.Equal(t, "BEX Logistics-OH Consolidation", pickup.Name)
	assert.Equal(t, "3685 Alum Creek Dr", pickup.AddressLine1)
	assert.Equal(t, "Columbus", pickup.City)
	assert.Equal(t, "OH", pickup.State)
	assert.Equal(t, "08/02/22", pickup.Date)
	assert.Equal(t, "08:00-18:00", pickup.TimeWindow)
	assert.Equal(t, stopRoleDelivery, delivery.Role)
	assert.Equal(t, "Freight, Land, Sky", delivery.Name)
	assert.Equal(t, "1000 Sycamore St.", delivery.AddressLine1)
	assert.Equal(t, "HARRISBURG", delivery.City)
	assert.Equal(t, "08/03/22", delivery.Date)
	assert.Equal(t, "BEX Logistics-OH Consolidation", analysis.Fields["shipper"].Value)
	assert.Equal(t, "Freight, Land, Sky", analysis.Fields["consignee"].Value)
	assert.Equal(t, "$2,481.00", analysis.Fields["rate"].Value)
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
		&ClassificationResult{
			Kind:           kindBillOfLading,
			Confidence:     0.9,
			Signals:        []string{"bill of lading"},
			Source:         "deterministic",
			Reason:         "header matched BOL",
			ReviewRequired: false,
		},
		&ExtractionResult{
			Text: text,
			Pages: []*PageExtractionResult{{
				PageNumber: 1,
				SourceKind: documentcontent.SourceKindNative,
				Text:       text,
			}},
		},
	)

	assert.Equal(t, kindBillOfLading, analysis.Kind)
	assert.Equal(t, reviewStatusReady, analysis.ReviewStatus)
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
		&ClassificationResult{
			Kind:           kindProofOfDelivery,
			Confidence:     0.9,
			Signals:        []string{"proof of delivery"},
			Source:         "deterministic",
			Reason:         "header matched POD",
			ReviewRequired: false,
		},
		&ExtractionResult{
			Text: text,
			Pages: []*PageExtractionResult{{
				PageNumber: 1,
				SourceKind: documentcontent.SourceKindNative,
				Text:       text,
			}},
		},
	)

	assert.Equal(t, kindProofOfDelivery, analysis.Kind)
	assert.Equal(t, reviewStatusReady, analysis.ReviewStatus)
	assert.Equal(t, "Blue Market Receiving", analysis.Fields["consignee"].Value)
	assert.Equal(t, "03/28/2026", analysis.Fields["deliveryWindow"].Value)
	assert.Equal(t, "Jane Smith", analysis.Fields["signature"].Value)
	assert.Equal(t, "POD99881", analysis.Fields["referenceNumber"].Value)
	assert.Equal(t, "Received in good order", analysis.Fields["receiptNotes"].Value)
	assert.Empty(t, analysis.MissingFields)
}

func TestBuildStructuredDataIncludesParsingRuleMetadata(t *testing.T) {
	t.Parallel()

	intelligence := &DocumentIntelligenceAnalysis{
		Kind:          kindRateConfirmation,
		ReviewStatus:  reviewStatusReady,
		MissingFields: []string{},
		Signals:       []string{"rule:generic"},
		Fields:        map[string]*ReviewField{},
		Stops:         []*IntelligenceStop{},
		Conflicts:     []*ReviewConflict{},
		ParsingRuleMetadata: &documentparsingruleports.DocumentParsingRuleMetadata{
			RuleSetID:        pulid.MustNew("dprs_"),
			RuleSetName:      "Generic",
			RuleVersionID:    pulid.MustNew("dprv_"),
			VersionNumber:    2,
			ParserMode:       "merge_with_base",
			ProviderMatched:  "GenericBroker",
			MatchSpecificity: 220,
		},
	}

	data := buildStructuredData(intelligence, &AIDiagnostics{
		FallbackAnalysis: intelligence,
		AcceptanceStatus: aiAcceptanceStatusRejected,
		RejectionReason:  "ai_candidate_missing_pickup_stop",
	})
	require.Equal(t, 6, data["schemaVersion"])

	intelligenceMap, ok := data["intelligence"].(map[string]any)
	require.True(t, ok)
	metadata, ok := intelligenceMap["parsingRuleMetadata"].(*documentparsingruleports.DocumentParsingRuleMetadata)
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
		&ClassificationResult{
			Kind:           kindProofOfDelivery,
			Confidence:     0.88,
			Signals:        []string{"proof of delivery"},
			Source:         "deterministic",
			Reason:         "header matched POD",
			ReviewRequired: false,
		},
		&ExtractionResult{
			Text: text,
			Pages: []*PageExtractionResult{{
				PageNumber: 1,
				SourceKind: documentcontent.SourceKindNative,
				Text:       text,
			}},
		},
	)

	assert.Equal(t, "NeedsReview", analysis.ReviewStatus)
	assert.Contains(t, analysis.MissingFields, "Signature")
}

func TestMergeAIAnalysis_NormalizesEquivalentAIExtractionShape(t *testing.T) {
	t.Parallel()

	fallback := &DocumentIntelligenceAnalysis{
		Kind:                 kindRateConfirmation,
		OverallConfidence:    0.61,
		ReviewStatus:         "NeedsReview",
		ClassifierSource:     "ai-route",
		ProviderFingerprint:  "provider=CHRobinson",
		ClassificationReason: "AI route matched rate confirmation",
		Fields:               map[string]*ReviewField{},
		Stops:                []*IntelligenceStop{},
		Conflicts:            []*ReviewConflict{},
	}

	aiExtract := &documentaiextractionports.AIExtractResult{
		DocumentKind:      "rate_confirmation",
		OverallConfidence: 0.72,
		ReviewStatus:      "REVIEW_REQUIRED",
		Fields: map[string]documentaiextractionports.AIDocumentField{
			"totalRate": {
				Label:           "Total Rate",
				Value:           "4500.00",
				Confidence:      0.95,
				EvidenceExcerpt: "$4,500.00",
				PageNumber:      3,
				Source:          "page-3",
			},
		},
		Stops: []*documentaiextractionports.AIDocumentStop{
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
