package documentextractionsupport

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type NoopAIDocumentService struct{}

func (NoopAIDocumentService) RouteDocument(
	context.Context,
	*documentaiextractionports.AIRouteRequest,
) (*documentaiextractionports.AIRouteResult, error) {
	return nil, nil //nolint:nilnil // intentional noop
}

func (NoopAIDocumentService) ExtractRateConfirmation(
	context.Context,
	*documentaiextractionports.AIExtractRequest,
) (*documentaiextractionports.AIExtractResult, error) {
	return nil, nil //nolint:nilnil // intentional noop
}

func (NoopAIDocumentService) SubmitRateConfirmationBackgroundExtraction(
	context.Context,
	*documentaiextractionports.AIExtractRequest,
) (*documentaiextractionports.AIBackgroundExtractSubmission, error) {
	return nil, nil //nolint:nilnil // intentional noop
}

func (NoopAIDocumentService) ExtractionAvailability(
	context.Context,
	pagination.TenantInfo,
) (documentaiextractionports.ExtractionAvailability, error) {
	return documentaiextractionports.ExtractionNoProviderAssigned, nil
}

func (NoopAIDocumentService) PollRateConfirmationBackgroundExtraction(
	context.Context,
	*documentaiextractionports.AIBackgroundExtractPollRequest,
) (*documentaiextractionports.AIBackgroundExtractPollResult, error) {
	return nil, nil //nolint:nilnil // intentional noop
}

func (NoopAIDocumentService) ExtractRateConfirmationForEvaluation(
	context.Context,
	*documentaiextractionports.AIEvaluationExtractRequest,
) (*documentaiextractionports.AIEvaluationExtractResult, error) {
	return nil, nil //nolint:nilnil // intentional noop
}

func (NoopAIDocumentService) ExtractRateConfirmationForShadow(
	context.Context,
	*documentaiextractionports.AIShadowExtractRequest,
) (*documentaiextractionports.AIEvaluationExtractResult, error) {
	return nil, nil //nolint:nilnil // intentional noop
}

type noopDocumentSearchProjectionService struct{}

func (noopDocumentSearchProjectionService) Upsert(
	context.Context,
	*document.Document,
	string,
) error {
	return nil
}

func (noopDocumentSearchProjectionService) Delete(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) error {
	return nil
}
