package extractionevaljobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/temporaljobs/documentintelligencejobs"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionevalports"
	"go.uber.org/fx"
)

var _ extractionevalports.ExtractionPredictor = (*Predictor)(nil)

type PredictorParams struct {
	fx.In

	Documents documentaiextractionports.AIDocumentService
}

type Predictor struct {
	documents documentaiextractionports.AIDocumentService
}

func NewPredictor(p PredictorParams) *Predictor {
	return &Predictor{documents: p.Documents}
}

func (p *Predictor) Predict(
	ctx context.Context,
	req *extractionevalports.PredictExtractionRequest,
) (*extractionevalports.ExtractionPrediction, error) {
	pages := make([]documentaiextractionports.AIDocumentPage, 0, len(req.Pages))
	for _, page := range req.Pages {
		pages = append(pages, documentaiextractionports.AIDocumentPage{PageNumber: page.Number, Text: page.Text})
	}

	result, err := p.documents.ExtractRateConfirmationForEvaluation(
		ctx,
		&documentaiextractionports.AIEvaluationExtractRequest{
			TenantInfo: req.TenantInfo,
			ProviderID: req.ProviderID,
			FileName:   req.FileName,
			Pages:      pages,
		},
	)
	if err != nil {
		return nil, err
	}

	return &extractionevalports.ExtractionPrediction{
		DraftData:    documentintelligencejobs.ShipmentDraftDataFromAIExtract(result.Extract),
		Model:        result.Model,
		ProviderID:   result.ProviderID,
		InputTokens:  result.InputTokens,
		OutputTokens: result.OutputTokens,
		LatencyMs:    result.LatencyMs,
		CostUSD:      result.CostUSD,
	}, nil
}
