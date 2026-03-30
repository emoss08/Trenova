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
	return nil, nil
}

func (noopAIDocumentService) ExtractRateConfirmation(
	context.Context,
	*services.AIExtractRequest,
) (*services.AIExtractResult, error) {
	return nil, nil
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
