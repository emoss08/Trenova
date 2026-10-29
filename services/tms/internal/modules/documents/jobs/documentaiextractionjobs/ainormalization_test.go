package documentaiextractionjobs

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentanalysis"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeAIAnalysis_NormalizesEquivalentAIExtractionShape(t *testing.T) {
	t.Parallel()

	fallback := &documentanalysis.DocumentIntelligenceAnalysis{
		Kind:                 documentanalysis.KindRateConfirmation,
		OverallConfidence:    0.61,
		ReviewStatus:         "NeedsReview",
		ClassifierSource:     "ai-route",
		ProviderFingerprint:  "provider=CHRobinson",
		ClassificationReason: "AI route matched rate confirmation",
		Fields:               map[string]*documentanalysis.ReviewField{},
		Stops:                []*documentanalysis.IntelligenceStop{},
		Conflicts:            []*documentanalysis.ReviewConflict{},
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
