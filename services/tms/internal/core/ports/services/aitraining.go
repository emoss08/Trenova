package services

import (
	"context"
	"io"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"

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

type TrainingDatasetSink interface {
	Create(name string) (io.WriteCloser, error)
}

type RenderTrainingDatasetRequest struct {
	ExportID             pulid.ID
	StructuredOutputMode aiprovider.StructuredOutputMode
	Sink                 TrainingDatasetSink
}

type AITrainingDatasetRenderer interface {
	Render(
		ctx context.Context,
		req *RenderTrainingDatasetRequest,
	) (*aitraining.DatasetManifest, error)
}

type ScoreTrainingPredictionsRequest struct {
	Evaluation  io.Reader
	Predictions io.Reader
}

type AITrainingScorer interface {
	Score(
		ctx context.Context,
		req *ScoreTrainingPredictionsRequest,
	) (*aitraining.ScoreReport, error)
}

type PlanAIRetrainingRequest struct {
	Manual      bool
	RequestedBy string
	Note        string
}

type ClaimAIRetrainingRequest struct {
	Trainer string
}

type HeartbeatAIRetrainingRequest struct {
	CycleID pulid.ID
	Trainer string
}

type RecordAIRetrainingRequest struct {
	CycleID pulid.ID
	Trainer string
	Result  *aitraining.RetrainingResult
}

type FailAIRetrainingRequest struct {
	CycleID pulid.ID
	Trainer string
	Message string
}

type AIRetrainingService interface {
	Plan(ctx context.Context, req *PlanAIRetrainingRequest) (*aitraining.RetrainingCycle, error)
	Reconcile(ctx context.Context) (int, error)
	ClaimNext(
		ctx context.Context,
		req *ClaimAIRetrainingRequest,
	) (*aitraining.RetrainingCycle, error)
	Heartbeat(
		ctx context.Context,
		req *HeartbeatAIRetrainingRequest,
	) (*aitraining.RetrainingCycle, error)
	Record(
		ctx context.Context,
		req *RecordAIRetrainingRequest,
	) (*aitraining.RetrainingCycle, error)
	FailTraining(
		ctx context.Context,
		req *FailAIRetrainingRequest,
	) (*aitraining.RetrainingCycle, error)
	Cancel(ctx context.Context, id pulid.ID) (*aitraining.RetrainingCycle, error)
	List(ctx context.Context, limit int) ([]*aitraining.RetrainingCycle, error)
	Get(ctx context.Context, id pulid.ID) (*aitraining.RetrainingCycle, error)
	LeaseDuration() time.Duration
}

type RetrainingAlerter interface {
	AlertRetraining(ctx context.Context, cycle *aitraining.RetrainingCycle) error
}
