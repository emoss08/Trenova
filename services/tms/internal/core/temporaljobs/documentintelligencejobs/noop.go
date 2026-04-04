package documentintelligencejobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/document"
	services "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/client"
)

type noopAIDocumentService struct{}

func (noopAIDocumentService) RouteDocument(
	context.Context,
	*services.AIRouteRequest,
) (*services.AIRouteResult, error) {
	return nil, nil //nolint:nilnil // intentional noop
}

func (noopAIDocumentService) ExtractRateConfirmation(
	context.Context,
	*services.AIExtractRequest,
) (*services.AIExtractResult, error) {
	return nil, nil //nolint:nilnil // intentional noop
}

func (noopAIDocumentService) SubmitRateConfirmationBackgroundExtraction(
	context.Context,
	*services.AIExtractRequest,
) (*services.AIBackgroundExtractSubmission, error) {
	return nil, nil //nolint:nilnil // intentional noop
}

func (noopAIDocumentService) PollRateConfirmationBackgroundExtraction(
	context.Context,
	*services.AIBackgroundExtractPollRequest,
) (*services.AIBackgroundExtractPollResult, error) {
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
	return nil, services.ErrWorkflowStarterDisabled
}

func (noopWorkflowStarter) Enabled() bool {
	return false
}

type noopDocumentParsingRuleRuntime struct{}

func (noopDocumentParsingRuleRuntime) ApplyPublished(
	context.Context,
	*services.DocumentParsingRuntimeInput,
	*services.DocumentParsingAnalysis,
) (*services.DocumentParsingAnalysis, error) {
	return nil, nil //nolint:nilnil // intentional noop
}

func (noopDocumentParsingRuleRuntime) SimulateVersion(
	context.Context,
	*services.DocumentParsingSimulationRequest,
) (*services.DocumentParsingSimulationResult, error) {
	return nil, nil //nolint:nilnil // intentional noop
}
