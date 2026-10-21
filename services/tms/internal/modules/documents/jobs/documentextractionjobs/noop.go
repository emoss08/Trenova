package documentextractionjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/ports/documentparsingruleports"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/client"
)

type noopAIDocumentService struct{}

func (noopAIDocumentService) RouteDocument(
	context.Context,
	*documentaiextractionports.AIRouteRequest,
) (*documentaiextractionports.AIRouteResult, error) {
	return nil, nil //nolint:nilnil // intentional noop
}

func (noopAIDocumentService) ExtractRateConfirmation(
	context.Context,
	*documentaiextractionports.AIExtractRequest,
) (*documentaiextractionports.AIExtractResult, error) {
	return nil, nil //nolint:nilnil // intentional noop
}

func (noopAIDocumentService) SubmitRateConfirmationBackgroundExtraction(
	context.Context,
	*documentaiextractionports.AIExtractRequest,
) (*documentaiextractionports.AIBackgroundExtractSubmission, error) {
	return nil, nil //nolint:nilnil // intentional noop
}

func (noopAIDocumentService) ExtractionAvailability(
	context.Context,
	pagination.TenantInfo,
) (documentaiextractionports.ExtractionAvailability, error) {
	return documentaiextractionports.ExtractionNoProviderAssigned, nil
}

func (noopAIDocumentService) PollRateConfirmationBackgroundExtraction(
	context.Context,
	*documentaiextractionports.AIBackgroundExtractPollRequest,
) (*documentaiextractionports.AIBackgroundExtractPollResult, error) {
	return nil, nil //nolint:nilnil // intentional noop
}

func (noopAIDocumentService) ExtractRateConfirmationForEvaluation(
	context.Context,
	*documentaiextractionports.AIEvaluationExtractRequest,
) (*documentaiextractionports.AIEvaluationExtractResult, error) {
	return nil, nil //nolint:nilnil // intentional noop
}

func (noopAIDocumentService) ExtractRateConfirmationForShadow(
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

type noopWorkflowStarter struct{}

func (noopWorkflowStarter) StartWorkflow(
	context.Context,
	client.StartWorkflowOptions,
	any,
	...any,
) (client.WorkflowRun, error) {
	return nil, kernelports.ErrWorkflowStarterDisabled
}

func (noopWorkflowStarter) CancelWorkflow(context.Context, string, string) error {
	return nil
}

func (noopWorkflowStarter) SignalWorkflow(context.Context, string, string, string, any) error {
	return nil
}

func (noopWorkflowStarter) Enabled() bool {
	return false
}

type noopDocumentParsingRuleRuntime struct{}

func parsingRuleRuntimeOrNoop(
	runtime documentparsingruleports.DocumentParsingRuleRuntime,
) documentparsingruleports.DocumentParsingRuleRuntime {
	if runtime == nil {
		return noopDocumentParsingRuleRuntime{}
	}
	return runtime
}

func (noopDocumentParsingRuleRuntime) ApplyPublished(
	context.Context,
	*documentparsingruleports.DocumentParsingRuntimeInput,
	*documentparsingruleports.DocumentParsingAnalysis,
) (*documentparsingruleports.DocumentParsingAnalysis, error) {
	return nil, nil //nolint:nilnil // intentional noop
}

func (noopDocumentParsingRuleRuntime) SimulateVersion(
	context.Context,
	*documentparsingruleports.DocumentParsingSimulationRequest,
) (*documentparsingruleports.DocumentParsingSimulationResult, error) {
	return nil, nil //nolint:nilnil // intentional noop
}
