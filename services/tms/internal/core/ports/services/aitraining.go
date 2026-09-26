package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type StartAITrainingExportRequest struct {
	CapturedFrom       int64
	CapturedTo         int64
	MaxPerOrganization int
	ValidationPercent  int
	RequestedBy        string
	Note               string
}

type ExportTrainingOrganizationRequest struct {
	ExportID  pulid.ID
	Ordinal   int
	Consent   repositories.TrainingConsent
	Heartbeat func(ctx context.Context, details ...any)
}

type FinishAITrainingExportRequest struct {
	ExportID pulid.ID
}

type AITrainingExportOperator interface {
	Start(
		ctx context.Context,
		req *StartAITrainingExportRequest,
	) (*aitraining.TrainingExport, error)
	List(ctx context.Context, limit int) ([]*aitraining.TrainingExport, error)
	Get(ctx context.Context, id pulid.ID) (*aitraining.TrainingExport, error)
	Cancel(ctx context.Context, id pulid.ID) (*aitraining.TrainingExport, error)
	WithdrawnExamples(
		ctx context.Context,
		req repositories.ListWithdrawnTrainingExamplesRequest,
	) ([]repositories.WithdrawnTrainingExample, error)
}

type AITrainingExportStarter interface {
	StartAITrainingExport(ctx context.Context, exportID pulid.ID) (string, error)
}

type AITrainingExportRunner interface {
	Begin(ctx context.Context, exportID pulid.ID) (*aitraining.TrainingExport, error)
	IsActive(ctx context.Context, exportID pulid.ID) (bool, error)
	ListOrganizations(
		ctx context.Context,
		req repositories.ListConsentingOrganizationsRequest,
	) ([]repositories.TrainingConsent, error)
	ExportOrganization(
		ctx context.Context,
		req *ExportTrainingOrganizationRequest,
	) (*aitraining.OrganizationProgress, error)
	Finish(
		ctx context.Context,
		req *FinishAITrainingExportRequest,
	) (*aitraining.TrainingExport, error)
	Fail(ctx context.Context, exportID pulid.ID, message string) error
}

type AITrainingHistoryService interface {
	ListHistory(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*aitraining.ExportHistoryEntry, error)
}
