package documentintelligencejobs

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	services "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var evidencePages = []services.AIDocumentPage{
	{
		PageNumber: 1,
		Text: "RATE CONFIRMATION   Load # OUG-8393964\n" +
			"Shipper: Juniper Manufacturing, 8862 Poplar Rd, Hartwell, CA 85055\n" +
			"Consignee: Granite Brands, Crestview, CO 52999",
	},
	{
		PageNumber: 2,
		Text: "Carrier pay\nLinehaul $2,563.12 USD\nPO 4500123345\n" +
			"Reference Juniper Manufacturing order",
	},
}

func TestEvidenceNeedlesMatchMoneyAsItIsPrinted(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"2563.12", "2,563.12"}, evidenceNeedles(" 2563.12 "))
	assert.Equal(t, []string{"$2,563.12"}, evidenceNeedles("$2,563.12")[:1])
	assert.Equal(t, []string{"1500", "1,500"}, evidenceNeedles("1500"))
	assert.Equal(t, []string{"0.999", "1"}, evidenceNeedles("0.999"))
	assert.Equal(t, []string{"-1250.5", "-1,250.50"}, evidenceNeedles("-1250.5"))
	assert.Equal(t, []string{"950"}, evidenceNeedles("950"))
	assert.Equal(t, []string{"2026-03-14"}, evidenceNeedles("2026-03-14"))
	assert.Nil(t, evidenceNeedles("  "))
}

func TestLocateEvidenceQuotesThePageAroundTheValue(t *testing.T) {
	t.Parallel()

	page, excerpt := locateEvidence("2563.12", 0, evidencePages)
	assert.Equal(t, 2, page)
	assert.Contains(t, excerpt, "Linehaul $2,563.12 USD")
	assert.NotContains(t, excerpt, "\n")

	page, excerpt = locateEvidence("oug-8393964", 0, evidencePages)
	assert.Equal(t, 1, page)
	assert.Contains(t, excerpt, "OUG-8393964")

	page, excerpt = locateEvidence("Unknown Shipper LLC", 1, evidencePages)
	assert.Zero(t, page)
	assert.Empty(t, excerpt)
}

func TestLocateEvidenceSearchesTheModelsPageFirst(t *testing.T) {
	t.Parallel()

	page, excerpt := locateEvidence("Juniper Manufacturing", 2, evidencePages)
	assert.Equal(t, 2, page)
	assert.Contains(t, excerpt, "Reference Juniper Manufacturing order")

	page, _ = locateEvidence("Juniper Manufacturing", 0, evidencePages)
	assert.Equal(t, 1, page)
}

func TestLocateEvidenceHoldsExcerptsToTheirLimit(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("word ", 200) + "TARGET-1" + strings.Repeat(" word", 200)
	value := strings.Repeat("x", 250)
	pages := []services.AIDocumentPage{{PageNumber: 1, Text: long + " " + value}}

	_, excerpt := locateEvidence("TARGET-1", 0, pages)
	assert.Contains(t, excerpt, "TARGET-1")
	assert.LessOrEqual(t, len([]rune(excerpt)), aiEvidenceMaxRunes)

	_, excerpt = locateEvidence(value, 0, pages)
	assert.Len(t, []rune(excerpt), aiEvidenceMaxRunes)
}

func TestWithFieldEvidenceFillsOnlyWhatIsMissing(t *testing.T) {
	t.Parallel()

	result := &services.AIExtractResult{
		Fields: map[string]services.AIDocumentField{
			"rate":      {Value: "2563.12", PageNumber: 2},
			"poNumber":  {Value: "4500123345"},
			"shipper":   {Value: "Juniper Manufacturing", PageNumber: 1, EvidenceExcerpt: "kept"},
			"commodity": {Value: "Paper products", PageNumber: 1},
		},
	}

	located := withFieldEvidence(result, evidencePages)

	assert.Contains(t, located.Fields["rate"].EvidenceExcerpt, "$2,563.12")
	assert.Equal(t, 2, located.Fields["rate"].PageNumber)
	assert.Contains(t, located.Fields["poNumber"].EvidenceExcerpt, "4500123345")
	assert.Equal(t, 2, located.Fields["poNumber"].PageNumber,
		"a page the model left out is filled in")
	assert.Equal(t, "kept", located.Fields["shipper"].EvidenceExcerpt)
	assert.Empty(t, located.Fields["commodity"].EvidenceExcerpt)
	assert.Equal(t, 1, located.Fields["commodity"].PageNumber)

	assert.Empty(t, result.Fields["rate"].EvidenceExcerpt, "the input is not modified")
	assert.Zero(t, result.Fields["poNumber"].PageNumber)
}

func TestWithFieldEvidenceWithoutPagesLeavesTheResultAlone(t *testing.T) {
	t.Parallel()

	result := &services.AIExtractResult{
		Fields: map[string]services.AIDocumentField{"rate": {Value: "1"}},
	}
	assert.Same(t, result, withFieldEvidence(result, nil))
	assert.Nil(t, withFieldEvidence(nil, evidencePages))
}

func TestAnAcceptedExtractionCarriesEvidenceFromThePages(t *testing.T) {
	t.Parallel()

	extract := &services.AIExtractResult{
		DocumentKind:      "RateConfirmation",
		OverallConfidence: 0.9,
		ReviewStatus:      "Ready",
		Fields: map[string]services.AIDocumentField{
			"rate":     {Label: "Rate", Value: "2563.12", Confidence: 0.9, PageNumber: 2},
			"poNumber": {Label: "PO Number", Value: "4500123345", Confidence: 0.9},
		},
		Stops: []*services.AIDocumentStop{
			{
				Sequence:        1,
				Role:            "pickup",
				Name:            "Juniper Manufacturing",
				PageNumber:      1,
				EvidenceExcerpt: "Shipper: Juniper Manufacturing",
				Confidence:      0.9,
			},
			{
				Sequence:        2,
				Role:            "delivery",
				Name:            "Granite Brands",
				PageNumber:      1,
				EvidenceExcerpt: "Consignee: Granite Brands",
				Confidence:      0.9,
			},
		},
	}
	payload := &ApplyDocumentAIExtractionPayload{
		Completion: &AsyncAIExtractionCompletion{
			Status:        services.AIBackgroundExtractionStatusCompleted,
			ExtractResult: extract,
		},
	}

	analysis, diagnostics := (&Activities{}).mergeCompletionIntoIntelligence(
		&documentcontent.Content{},
		evidencePages,
		payload,
	)

	require.Equal(t, aiAcceptanceStatusAccepted, diagnostics.AcceptanceStatus,
		diagnostics.RejectionReason)
	require.Contains(t, analysis.Fields, "rate")
	assert.Contains(t, analysis.Fields["rate"].EvidenceExcerpt, "Linehaul $2,563.12")
	require.Contains(t, analysis.Fields, "ponumber")
	assert.Contains(t, analysis.Fields["ponumber"].EvidenceExcerpt, "PO 4500123345")
	assert.Equal(t, 2, analysis.Fields["ponumber"].PageNumber)
	assert.Contains(t, diagnostics.CandidateAnalysis.Fields["rate"].EvidenceExcerpt, "$2,563.12")
}
