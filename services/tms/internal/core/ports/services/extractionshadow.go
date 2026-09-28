package services

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/extractionshadow"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

var ErrShadowSuperseded = errors.New("the document was extracted again before the shadow ran")

type UpdateExtractionShadowSettingsRequest struct {
	TenantInfo    pagination.TenantInfo
	Enabled       bool
	ProviderID    pulid.ID
	SamplePercent int
	DailyLimit    int
	Version       int64
}

type ExtractionShadowReportRequest struct {
	TenantInfo pagination.TenantInfo
	WindowDays int
	ProviderID pulid.ID
}

type ExtractionShadowSide struct {
	Scored    int
	Correct   int
	Corrected int
	Missed    int
	Accuracy  float64
}

type ExtractionShadowReport struct {
	WindowDays   int
	Since        int64
	ProviderID   *pulid.ID
	ProviderName string
	Sampled      int
	Pending      int
	Completed    int
	Failed       int
	Skipped      int
	Scored       int
	Truncated    bool
	Better       int
	Worse        int
	Same         int
	Candidate    ExtractionShadowSide
	Production   ExtractionShadowSide
	Fields       []aicorrection.FieldComparison
	CostUSD      decimal.Decimal
	AvgLatencyMs int64
}

type ExtractionShadowService interface {
	GetSettings(
		ctx context.Context,
		tenant pagination.TenantInfo,
	) (*extractionshadow.ShadowSettings, error)
	UpdateSettings(
		ctx context.Context,
		req *UpdateExtractionShadowSettingsRequest,
		actor *RequestActor,
	) (*extractionshadow.ShadowSettings, error)
	Report(ctx context.Context, req *ExtractionShadowReportRequest) (*ExtractionShadowReport, error)
	ListResults(
		ctx context.Context,
		req *repositories.ListExtractionShadowResultConnectionRequest,
	) (*pagination.CursorListResult[*extractionshadow.ShadowResult], error)
	GetResult(
		ctx context.Context,
		req repositories.GetExtractionShadowResultRequest,
	) (*extractionshadow.ShadowResult, error)
}

type ConsiderExtractionShadowRequest struct {
	TenantInfo           pagination.TenantInfo
	DocumentID           pulid.ID
	ExtractedAt          int64
	ProductionProviderID pulid.ID
	ProductionModel      string
}

type ExtractionShadowDecision struct {
	Sampled    bool
	ResultID   pulid.ID
	WorkflowID string
	Reason     string
}

type ExtractionShadowSampler interface {
	ConsiderExtraction(
		ctx context.Context,
		req *ConsiderExtractionShadowRequest,
	) (*ExtractionShadowDecision, error)
}

type ExtractionShadowStart struct {
	TenantInfo pagination.TenantInfo
	ResultID   pulid.ID
}

type ExtractionShadowStarter interface {
	StartExtractionShadow(ctx context.Context, start *ExtractionShadowStart) (string, error)
}

type RunExtractionShadowRequest struct {
	TenantInfo   pagination.TenantInfo
	ResultID     pulid.ID
	FinalAttempt bool
}

type ExtractionShadowRunner interface {
	RunShadow(ctx context.Context, req *RunExtractionShadowRequest) error
	FailShadow(
		ctx context.Context,
		req repositories.GetExtractionShadowResultRequest,
		message string,
	) error
	PurgeExpiredShadows(ctx context.Context, req PurgeExpiredAICorrectionsRequest) (int64, error)
}

type ExtractionShadowScorer interface {
	ScoreCorrection(ctx context.Context, correction *aicorrection.Correction) error
}

type PredictShadowDraftRequest struct {
	TenantInfo  pagination.TenantInfo
	DocumentID  pulid.ID
	ExtractedAt int64
	ProviderID  pulid.ID
}

type ShadowDraftPrediction struct {
	DraftData       map[string]any
	Accepted        bool
	RejectionReason string
	Model           string
	ProviderID      pulid.ID
	InputTokens     int
	OutputTokens    int
	LatencyMs       int64
	CostUSD         *decimal.Decimal
}

type ExtractionShadowPredictor interface {
	PredictShadowDraft(
		ctx context.Context,
		req *PredictShadowDraftRequest,
	) (*ShadowDraftPrediction, error)
}
