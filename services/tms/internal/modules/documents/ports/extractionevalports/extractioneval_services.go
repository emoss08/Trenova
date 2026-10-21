package extractionevalports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aicorrectionports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractioneval"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
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
	Cases       ExtractionEvalCaseCounts
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
	RefreezeCases(
		ctx context.Context,
		req *RefreezeExtractionEvalCasesRequest,
		actor *permissionports.RequestActor,
	) (*RefreezeExtractionEvalCasesResult, error)
	RefreezeProgress(
		ctx context.Context,
		tenant pagination.TenantInfo,
	) (*ExtractionEvalRefreezeProgress, error)
	PromoteCorrection(
		ctx context.Context,
		req *PromoteCorrectionRequest,
		actor *permissionports.RequestActor,
	) (*extractioneval.ExtractionCase, error)
	UpdateCase(
		ctx context.Context,
		req *UpdateExtractionCaseRequest,
		actor *permissionports.RequestActor,
	) (*extractioneval.ExtractionCase, error)
	DeleteCase(
		ctx context.Context,
		req GetExtractionEvalCaseRequest,
		actor *permissionports.RequestActor,
	) error
	GetCase(
		ctx context.Context,
		req GetExtractionEvalCaseRequest,
	) (*extractioneval.ExtractionCase, error)
	ListCases(
		ctx context.Context,
		req *ListExtractionEvalCaseConnectionRequest,
	) (*pagination.CursorListResult[*extractioneval.ExtractionCase], error)
	GetCorrection(
		ctx context.Context,
		req aicorrectionports.GetAICorrectionRequest,
	) (*aicorrection.Correction, error)
	ListCorrections(
		ctx context.Context,
		req *aicorrectionports.ListAICorrectionConnectionRequest,
	) (*pagination.CursorListResult[*aicorrection.Correction], error)
	StartRun(
		ctx context.Context,
		req *StartExtractionEvalRunRequest,
		actor *permissionports.RequestActor,
	) (*extractioneval.ExtractionRun, error)
	CancelRun(
		ctx context.Context,
		req GetExtractionEvalRunRequest,
		actor *permissionports.RequestActor,
	) (*extractioneval.ExtractionRun, error)
	GetRun(
		ctx context.Context,
		req GetExtractionEvalRunRequest,
	) (*extractioneval.ExtractionRun, error)
	ListRuns(
		ctx context.Context,
		req *ListExtractionEvalRunConnectionRequest,
	) (*pagination.CursorListResult[*extractioneval.ExtractionRun], error)
	ListResults(
		ctx context.Context,
		req *ListExtractionEvalResultConnectionRequest,
	) (*pagination.CursorListResult[*extractioneval.ExtractionResult], error)
	GetResult(
		ctx context.Context,
		req GetExtractionEvalResultRequest,
	) (*extractioneval.ExtractionResult, error)
	Accuracy(ctx context.Context, req *ExtractionAccuracyRequest) (*ExtractionAccuracy, error)
	ProviderTrends(
		ctx context.Context,
		tenant pagination.TenantInfo,
	) (*ExtractionProviderTrends, error)
}

const ExtractionAccuracyDriftEvent = "ai.extraction_accuracy_drift"

type ExtractionProviderTrend struct {
	ProviderID      pulid.ID
	ProviderName    string
	Model           string
	ProviderRemoved bool
	Weeks           []aicorrection.WeekAccuracy
	Checked         aicorrection.WeekAccuracy
	Baseline        aicorrection.WeekAccuracy
	DropPoints      float64
	Comparable      bool
	Drifting        bool
}

type ExtractionProviderTrends struct {
	Weeks             []int64
	CheckedWeek       int64
	BaselineStart     int64
	Providers         []ExtractionProviderTrend
	DriftPoints       int
	MinWeekFields     int
	MinBaselineFields int
}

type ExtractionDriftCheckRequest struct {
	TenantInfo pagination.TenantInfo
	Now        int64
}

type ExtractionDriftChecker interface {
	CheckDrift(ctx context.Context, req *ExtractionDriftCheckRequest) (int, error)
}

type ExtractionEvalRunner interface {
	BeginRun(
		ctx context.Context,
		req GetExtractionEvalRunRequest,
	) (*extractioneval.ExtractionRun, error)
	ListPending(
		ctx context.Context,
		req ListPendingExtractionEvalResultsRequest,
	) ([]*extractioneval.ExtractionResult, error)
	CheckContinue(
		ctx context.Context,
		req GetExtractionEvalRunRequest,
	) (*ExtractionEvalDecision, error)
	EvaluateResult(ctx context.Context, req *EvaluateExtractionResultRequest) error
	FinishRun(
		ctx context.Context,
		req *FinishExtractionEvalRunRequest,
	) (*extractioneval.ExtractionRun, error)
	FailRun(ctx context.Context, req GetExtractionEvalRunRequest, message string) error
	PurgeExpiredRuns(
		ctx context.Context,
		req kernelports.PurgeExpiredAICorrectionsRequest,
	) (int64, error)
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

type ExtractionEvalRefreezeStart struct {
	TenantInfo pagination.TenantInfo
	CaseIDs    []pulid.ID
}

type ExtractionEvalRefreezeStarted struct {
	WorkflowID     string
	AlreadyRunning bool
}

type ExtractionEvalRefreezeProgress struct {
	WorkflowID string
	Status     extractioneval.RefreezeStatus
	Total      int
	Frozen     int
	Skipped    int
	Failed     int
}

type ExtractionEvalRefreezeStarter interface {
	StartExtractionEvalRefreeze(
		ctx context.Context,
		start *ExtractionEvalRefreezeStart,
	) (*ExtractionEvalRefreezeStarted, error)
	ExtractionEvalRefreezeProgress(
		ctx context.Context,
		tenant pagination.TenantInfo,
	) (*ExtractionEvalRefreezeProgress, error)
}

type RefreezeExtractionEvalCasesRequest struct {
	TenantInfo pagination.TenantInfo
	CaseIDs    []pulid.ID
}

type RefreezeExtractionEvalCasesResult struct {
	Queued         int
	WorkflowID     string
	AlreadyRunning bool
}

type ConvertedCasePages struct {
	Pages            []extractioneval.Page
	ConverterVersion string
}

type ExtractionCaseConverter interface {
	ConvertCaseDocument(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		documentID pulid.ID,
	) (*ConvertedCasePages, error)
}

type FreezeCasePagesRequest struct {
	TenantInfo pagination.TenantInfo
	CaseID     pulid.ID
}

type FreezeCasePagesResult struct {
	Frozen bool
	Reason string
}

type ExtractionEvalCaseFreezer interface {
	FreezeDoclingPages(
		ctx context.Context,
		req *FreezeCasePagesRequest,
	) (*FreezeCasePagesResult, error)
}
