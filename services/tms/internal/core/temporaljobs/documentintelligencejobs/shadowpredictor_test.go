package documentintelligencejobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	services "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type shadowExtractor struct {
	services.AIDocumentService
	result   *services.AIEvaluationExtractResult
	requests []*services.AIShadowExtractRequest
}

func (f *shadowExtractor) ExtractRateConfirmationForShadow(
	_ context.Context,
	req *services.AIShadowExtractRequest,
) (*services.AIEvaluationExtractResult, error) {
	f.requests = append(f.requests, req)
	return f.result, nil
}

func acceptableExtract() *services.AIExtractResult {
	return &services.AIExtractResult{
		DocumentKind:      "RateConfirmation",
		OverallConfidence: 0.9,
		ReviewStatus:      "Ready",
		Fields: map[string]services.AIDocumentField{
			"shipper":   {Value: "Juniper Manufacturing", Confidence: 0.9, PageNumber: 1},
			"consignee": {Value: "Granite Brands", Confidence: 0.9, PageNumber: 1},
			"rate":      {Value: "2563.12", Confidence: 0.9, PageNumber: 2},
		},
		Stops: []*services.AIDocumentStop{
			{
				Sequence: 1, Role: "pickup", Name: "Juniper Manufacturing", PageNumber: 1,
				EvidenceExcerpt: "Shipper: Juniper Manufacturing", Confidence: 0.9,
			},
			{
				Sequence: 2, Role: "delivery", Name: "Granite Brands", PageNumber: 1,
				EvidenceExcerpt: "Consignee: Granite Brands", Confidence: 0.9,
			},
		},
	}
}

type shadowFixture struct {
	predictor *ShadowPredictor
	extractor *shadowExtractor
	request   *services.PredictShadowDraftRequest
}

func newShadowFixture(t *testing.T, lastExtractedAt int64, fallback map[string]any) *shadowFixture {
	t.Helper()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	request := &services.PredictShadowDraftRequest{
		TenantInfo:  tenant,
		DocumentID:  pulid.MustNew("doc_"),
		ExtractedAt: 1_700_000_000,
		ProviderID:  pulid.MustNew("aip_"),
	}

	docRepo := mocks.NewMockDocumentRepository(t)
	docRepo.EXPECT().GetByID(mock.Anything, repositories.GetDocumentByIDRequest{
		ID:         request.DocumentID,
		TenantInfo: tenant,
	}).Return(&document.Document{ID: request.DocumentID, OriginalName: "rc-4411.pdf"}, nil)

	pages := make([]*documentcontent.Page, 0, len(evidencePages))
	for _, page := range evidencePages {
		pages = append(pages, &documentcontent.Page{PageNumber: page.PageNumber, ExtractedText: page.Text})
	}
	contentRepo := mocks.NewMockDocumentContentRepository(t)
	contentRepo.EXPECT().GetByDocumentID(mock.Anything, request.DocumentID, tenant).
		Return(&documentcontent.Content{
			DocumentID:      request.DocumentID,
			ContentText:     "RATE CONFIRMATION Load # OUG-8393964",
			LastExtractedAt: &lastExtractedAt,
			StructuredData: map[string]any{
				"aiDiagnostics": map[string]any{"fallbackAnalysis": fallback},
			},
		}, nil)
	contentRepo.EXPECT().ListPagesByDocumentID(mock.Anything, request.DocumentID, tenant).
		Return(pages, nil).Maybe()

	extractor := &shadowExtractor{}
	return &shadowFixture{
		predictor: &ShadowPredictor{
			cfg:       &config.AIConfig{},
			documents: docRepo,
			contents:  contentRepo,
			extractor: extractor,
		},
		extractor: extractor,
		request:   request,
	}
}

func parserFallback() map[string]any {
	return map[string]any{
		"kind":             "RateConfirmation",
		"classifierSource": "deterministic",
		"fields": map[string]any{
			"rate": map[string]any{"value": "2500.00", "source": "template", "confidence": 0.6},
		},
		"stops": []any{},
	}
}

func TestShadowPredictorSendsProductionsInputToThePinnedProvider(t *testing.T) {
	t.Parallel()

	fixture := newShadowFixture(t, 1_700_000_000, parserFallback())
	cost := decimal.RequireFromString("0.01")
	fixture.extractor.result = &services.AIEvaluationExtractResult{
		Extract:      acceptableExtract(),
		Model:        "trenova-extract",
		ProviderID:   fixture.request.ProviderID,
		InputTokens:  900,
		OutputTokens: 400,
		LatencyMs:    1800,
		CostUSD:      &cost,
	}

	prediction, err := fixture.predictor.PredictShadowDraft(t.Context(), fixture.request)
	require.NoError(t, err)

	require.Len(t, fixture.extractor.requests, 1)
	sent := fixture.extractor.requests[0]
	assert.Equal(t, fixture.request.ProviderID, sent.ProviderID)
	assert.Equal(t, fixture.request.DocumentID, sent.DocumentID)
	assert.Equal(t, "rc-4411.pdf", sent.FileName)
	assert.Equal(t, "RATE CONFIRMATION Load # OUG-8393964", sent.Text)
	require.Len(t, sent.Pages, len(evidencePages))

	assert.True(t, prediction.Accepted)
	assert.Empty(t, prediction.RejectionReason)
	assert.Equal(t, "trenova-extract", prediction.Model)
	assert.Equal(t, 900, prediction.InputTokens)
	fields, ok := prediction.DraftData["fields"].(map[string]any)
	require.True(t, ok)
	rate, ok := fields["rate"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "2563.12", rate["value"], "the candidate's answer replaces the parser's reading")
	assert.Equal(t, "deterministic", prediction.DraftData["classifierSource"],
		"the merge keeps the parser's classification, as production's did")
	assert.Contains(t, rate["evidenceExcerpt"], "$2,563.12", "evidence is located on the pages as in production")
}

func TestShadowPredictorFallsBackToTheParserWhenTheAnswerIsRejected(t *testing.T) {
	t.Parallel()

	fixture := newShadowFixture(t, 1_700_000_000, parserFallback())
	rejected := acceptableExtract()
	delete(rejected.Fields, "rate")
	fixture.extractor.result = &services.AIEvaluationExtractResult{Extract: rejected}

	prediction, err := fixture.predictor.PredictShadowDraft(t.Context(), fixture.request)
	require.NoError(t, err)

	assert.False(t, prediction.Accepted)
	assert.Equal(t, "ai_candidate_missing_required_field_rate", prediction.RejectionReason)
	fields, ok := prediction.DraftData["fields"].(map[string]any)
	require.True(t, ok)
	rate, ok := fields["rate"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "2500.00", rate["value"], "a rejected answer leaves the draft production would have kept")
}

func TestShadowPredictorRefusesASupersededExtraction(t *testing.T) {
	t.Parallel()

	fixture := newShadowFixture(t, 1_700_000_500, parserFallback())

	_, err := fixture.predictor.PredictShadowDraft(t.Context(), fixture.request)
	require.ErrorIs(t, err, services.ErrShadowSuperseded)
	assert.Empty(t, fixture.extractor.requests)
}
