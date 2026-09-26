package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/extractioneval"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type PromoteCorrectionRequest struct {
	TenantInfo   pagination.TenantInfo
	CorrectionID pulid.ID
	Title        string
	Activate     bool
}

type UpdateExtractionCaseRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
	Version    int64
	Title      *string
	Notes      *string
	Status     *extractioneval.CaseStatus
}

type StartExtractionEvalRunRequest struct {
	TenantInfo pagination.TenantInfo
	ProviderID pulid.ID
	CaseLimit  int
}

type ExtractionAccuracyRequest struct {
	TenantInfo pagination.TenantInfo
	WindowDays int
}

type ExtractionAccuracyGroup struct {
	Key         string
	Corrections int
	Scored      int
	Correct     int
	Accuracy    float64
}

type ExtractionAccuracy struct {
	WindowDays  int
	Since       int64
	Corrections int
	Sampled     bool
	Scored      int
	Correct     int
	Corrected   int
	Missed      int
	Unconfirmed int
	Accuracy    float64
	ByModel     []ExtractionAccuracyGroup
	ByKind      []ExtractionAccuracyGroup
	Fields      []aicorrection.FieldAccuracy
	Cases       repositories.ExtractionEvalCaseCounts
	RecentRuns  []*extractioneval.ExtractionRun
}

type ExtractionEvalDecision struct {
	Stop   bool
	Status extractioneval.RunStatus
	Reason string
}

type EvaluateExtractionResultRequest struct {
	TenantInfo   pagination.TenantInfo
	ResultID     pulid.ID
	FinalAttempt bool
}

type FinishExtractionEvalRunRequest struct {
	TenantInfo pagination.TenantInfo
	RunID      pulid.ID
	Status     extractioneval.RunStatus
	Reason     string
}

type ExtractionEvalService interface {
	PromoteCorrection(
		ctx context.Context,
		req *PromoteCorrectionRequest,
		actor *RequestActor,
	) (*extractioneval.ExtractionCase, error)
	UpdateCase(
		ctx context.Context,
		req *UpdateExtractionCaseRequest,
		actor *RequestActor,
	) (*extractioneval.ExtractionCase, error)
	DeleteCase(
		ctx context.Context,
		req repositories.GetExtractionEvalCaseRequest,
		actor *RequestActor,
	) error
	GetCase(
		ctx context.Context,
		req repositories.GetExtractionEvalCaseRequest,
	) (*extractioneval.ExtractionCase, error)
	ListCases(
		ctx context.Context,
		req *repositories.ListExtractionEvalCaseConnectionRequest,
	) (*pagination.CursorListResult[*extractioneval.ExtractionCase], error)
	GetCorrection(
		ctx context.Context,
		req repositories.GetAICorrectionRequest,
	) (*aicorrection.Correction, error)
	ListCorrections(
		ctx context.Context,
		req *repositories.ListAICorrectionConnectionRequest,
	) (*pagination.CursorListResult[*aicorrection.Correction], error)
	StartRun(
		ctx context.Context,
		req *StartExtractionEvalRunRequest,
		actor *RequestActor,
	) (*extractioneval.ExtractionRun, error)
	CancelRun(
		ctx context.Context,
		req repositories.GetExtractionEvalRunRequest,
		actor *RequestActor,
	) (*extractioneval.ExtractionRun, error)
	GetRun(
		ctx context.Context,
		req repositories.GetExtractionEvalRunRequest,
	) (*extractioneval.ExtractionRun, error)
	ListRuns(
		ctx context.Context,
		req *repositories.ListExtractionEvalRunConnectionRequest,
	) (*pagination.CursorListResult[*extractioneval.ExtractionRun], error)
	ListResults(
		ctx context.Context,
		req *repositories.ListExtractionEvalResultConnectionRequest,
	) (*pagination.CursorListResult[*extractioneval.ExtractionResult], error)
	GetResult(
		ctx context.Context,
		req repositories.GetExtractionEvalResultRequest,
	) (*extractioneval.ExtractionResult, error)
	Accuracy(ctx context.Context, req *ExtractionAccuracyRequest) (*ExtractionAccuracy, error)
}

type ExtractionEvalRunner interface {
	BeginRun(
		ctx context.Context,
		req repositories.GetExtractionEvalRunRequest,
	) (*extractioneval.ExtractionRun, error)
	ListPending(
		ctx context.Context,
		req repositories.ListPendingExtractionEvalResultsRequest,
	) ([]*extractioneval.ExtractionResult, error)
	CheckContinue(
		ctx context.Context,
		req repositories.GetExtractionEvalRunRequest,
	) (*ExtractionEvalDecision, error)
	EvaluateResult(ctx context.Context, req *EvaluateExtractionResultRequest) error
	FinishRun(
		ctx context.Context,
		req *FinishExtractionEvalRunRequest,
	) (*extractioneval.ExtractionRun, error)
	FailRun(ctx context.Context, req repositories.GetExtractionEvalRunRequest, message string) error
	PurgeExpiredRuns(ctx context.Context, req PurgeExpiredAICorrectionsRequest) (int64, error)
}

type ExtractionPrediction struct {
	DraftData    map[string]any
	Model        string
	ProviderID   pulid.ID
	InputTokens  int
	OutputTokens int
	LatencyMs    int64
	CostUSD      *decimal.Decimal
}

type PredictExtractionRequest struct {
	TenantInfo pagination.TenantInfo
	ProviderID pulid.ID
	FileName   string
	Pages      []extractioneval.Page
}

type ExtractionPredictor interface {
	Predict(ctx context.Context, req *PredictExtractionRequest) (*ExtractionPrediction, error)
}

type ExtractionEvalRunStart struct {
	TenantInfo pagination.TenantInfo
	RunID      pulid.ID
}

type ExtractionEvalRunStarter interface {
	StartExtractionEvalRun(ctx context.Context, start *ExtractionEvalRunStart) (string, error)
}
