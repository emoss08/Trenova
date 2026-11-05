package documentextractionjobs

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingfake"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentsearchprojectionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/client"
)

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

func unprobedImaging() *imagingfake.Fake {
	fake := imagingfake.New()
	fake.ProbeFunc = func(
		context.Context,
		*kernelports.ImagingProbeRequest,
	) (*kernelports.ImagingProbe, error) {
		return &kernelports.ImagingProbe{}, nil
	}

	return fake
}

type noopPaperwork struct{}

func (noopPaperwork) RequestRead(context.Context, *document.Document, pulid.ID) {}

type noopWorkflowStarter struct{}

func (noopWorkflowStarter) StartWorkflow(
	context.Context,
	client.StartWorkflowOptions,
	any,
	...any,
) (client.WorkflowRun, error) {
	return nil, errors.New("workflow starter is disabled")
}

func (noopWorkflowStarter) CancelWorkflow(context.Context, string, string) error {
	return nil
}

func (noopWorkflowStarter) SignalWorkflow(context.Context, string, string, string, any) error {
	return nil
}

var (
	_ documentsearchprojectionports.DocumentSearchProjectionService = noopDocumentSearchProjectionService{}
	_ kernelports.WorkflowStarter                                   = noopWorkflowStarter{}
)
