package documentintelligencejobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	services "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/fx"
)

var _ services.ExtractionShadowPredictor = (*ShadowPredictor)(nil)

type ShadowPredictorParams struct {
	fx.In

	Config            *config.Config
	DocumentRepo      repositories.DocumentRepository
	ContentRepo       repositories.DocumentContentRepository
	AIDocumentService services.AIDocumentService
}

type ShadowPredictor struct {
	cfg       *config.AIConfig
	documents repositories.DocumentRepository
	contents  repositories.DocumentContentRepository
	extractor services.AIDocumentService
}

func NewShadowPredictor(p ShadowPredictorParams) *ShadowPredictor {
	return &ShadowPredictor{
		cfg:       p.Config.GetAIConfig(),
		documents: p.DocumentRepo,
		contents:  p.ContentRepo,
		extractor: p.AIDocumentService,
	}
}

func (p *ShadowPredictor) PredictShadowDraft(
	ctx context.Context,
	req *services.PredictShadowDraftRequest,
) (*services.ShadowDraftPrediction, error) {
	doc, err := p.documents.GetByID(ctx, repositories.GetDocumentByIDRequest{
		ID:         req.DocumentID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	content, err := p.contents.GetByDocumentID(ctx, req.DocumentID, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	if content.LastExtractedAt == nil || *content.LastExtractedAt != req.ExtractedAt {
		return nil, services.ErrShadowSuperseded
	}

	pages, err := p.contents.ListPagesByDocumentID(ctx, req.DocumentID, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	maxChars := p.cfg.GetMaxInputChars()
	aiPages := toAIDocumentPages(pages, maxChars)
	result, err := p.extractor.ExtractRateConfirmationForShadow(ctx, &services.AIShadowExtractRequest{
		TenantInfo: req.TenantInfo,
		ProviderID: req.ProviderID,
		DocumentID: doc.ID,
		FileName:   doc.OriginalName,
		Text:       stringutils.TruncateAndTrim(content.ContentText, maxChars),
		Pages:      aiPages,
	})
	if err != nil {
		return nil, err
	}

	draft, accepted, rejectionReason := mergeAIAnalysis(
		analysisFromStructuredData(content.StructuredData),
		withFieldEvidence(result.Extract, aiPages),
	)

	return &services.ShadowDraftPrediction{
		DraftData:       draft.ToMap(),
		Accepted:        accepted,
		RejectionReason: rejectionReason,
		Model:           result.Model,
		ProviderID:      result.ProviderID,
		InputTokens:     result.InputTokens,
		OutputTokens:    result.OutputTokens,
		LatencyMs:       result.LatencyMs,
		CostUSD:         result.CostUSD,
	}, nil
}
