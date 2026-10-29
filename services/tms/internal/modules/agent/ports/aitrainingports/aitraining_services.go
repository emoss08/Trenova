package aitrainingports

import (
	"context"
	"io"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/airetraining"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitraining"
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
	Consent   TrainingConsent
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
		req ListWithdrawnTrainingExamplesRequest,
	) ([]WithdrawnTrainingExample, error)
}

type AITrainingExportStarter interface {
	StartAITrainingExport(ctx context.Context, exportID pulid.ID) (string, error)
}

type AITrainingExportRunner interface {
	Begin(ctx context.Context, exportID pulid.ID) (*aitraining.TrainingExport, error)
	IsActive(ctx context.Context, exportID pulid.ID) (bool, error)
	ListOrganizations(
		ctx context.Context,
		req ListConsentingOrganizationsRequest,
	) ([]TrainingConsent, error)
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
	Result  *airetraining.RetrainingResult
}

type FailAIRetrainingRequest struct {
	CycleID pulid.ID
	Trainer string
	Message string
}

type AIRetrainingService interface {
	Plan(ctx context.Context, req *PlanAIRetrainingRequest) (*airetraining.RetrainingCycle, error)
	Reconcile(ctx context.Context) (int, error)
	ClaimNext(
		ctx context.Context,
		req *ClaimAIRetrainingRequest,
	) (*airetraining.RetrainingCycle, error)
	Heartbeat(
		ctx context.Context,
		req *HeartbeatAIRetrainingRequest,
	) (*airetraining.RetrainingCycle, error)
	Record(
		ctx context.Context,
		req *RecordAIRetrainingRequest,
	) (*airetraining.RetrainingCycle, error)
	FailTraining(
		ctx context.Context,
		req *FailAIRetrainingRequest,
	) (*airetraining.RetrainingCycle, error)
	Cancel(ctx context.Context, id pulid.ID) (*airetraining.RetrainingCycle, error)
	List(ctx context.Context, limit int) ([]*airetraining.RetrainingCycle, error)
	Get(ctx context.Context, id pulid.ID) (*airetraining.RetrainingCycle, error)
	LeaseDuration() time.Duration
}

type RetrainingAlerter interface {
	AlertRetraining(ctx context.Context, cycle *airetraining.RetrainingCycle) error
}
