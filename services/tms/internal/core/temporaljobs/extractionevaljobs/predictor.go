package extractionevaljobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/documentintelligencejobs"
	"go.uber.org/fx"
)

var _ services.ExtractionPredictor = (*Predictor)(nil)

type PredictorParams struct {
	fx.In

	Documents services.AIDocumentService
}

type Predictor struct {
	documents services.AIDocumentService
}

func NewPredictor(p PredictorParams) *Predictor {
	return &Predictor{documents: p.Documents}
}

func (p *Predictor) Predict(
	ctx context.Context,
	req *services.PredictExtractionRequest,
) (*services.ExtractionPrediction, error) {
	pages := make([]services.AIDocumentPage, 0, len(req.Pages))
	for _, page := range req.Pages {
		pages = append(pages, services.AIDocumentPage{PageNumber: page.Number, Text: page.Text})
	}

	result, err := p.documents.ExtractRateConfirmationForEvaluation(
		ctx,
		&services.AIEvaluationExtractRequest{
			TenantInfo: req.TenantInfo,
			ProviderID: req.ProviderID,
			FileName:   req.FileName,
			Pages:      pages,
		},
	)
	if err != nil {
		return nil, err
	}

	return &services.ExtractionPrediction{
		DraftData:    documentintelligencejobs.ShipmentDraftDataFromAIExtract(result.Extract),
		Model:        result.Model,
		ProviderID:   result.ProviderID,
		InputTokens:  result.InputTokens,
		OutputTokens: result.OutputTokens,
		LatencyMs:    result.LatencyMs,
		CostUSD:      result.CostUSD,
	}, nil
}
