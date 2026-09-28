package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/extractionrollout"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const ExtractionRolloutHaltedEvent = "ai.extraction_rollout_halted"

type UpdateExtractionRolloutRequest struct {
	TenantInfo                 pagination.TenantInfo
	Enabled                    bool
	ProviderID                 pulid.ID
	Percent                    int
	MaxAccuracyDropPoints      int
	MaxRejectionIncreasePoints int
	Version                    int64
}

type ExtractionRolloutArmReport struct {
	Assigned      int
	Pending       int
	Accepted      int
	Rejected      int
	Failed        int
	Superseded    int
	FellBack      int
	RejectionRate float64
}

type ExtractionRolloutAccuracy struct {
	Scored   int
	Correct  int
	Accuracy float64
}

type ExtractionRolloutReport struct {
	Rollout              *extractionrollout.ExtractionRollout
	ProviderName         string
	Candidate            ExtractionRolloutArmReport
	Control              ExtractionRolloutArmReport
	CandidateAccuracy    ExtractionRolloutAccuracy
	ProductionAccuracy   ExtractionRolloutAccuracy
	Fields               []aicorrection.FieldComparison
	Truncated            bool
	MinGuardScoredFields int
	MinGuardExtractions  int
}

type ExtractionRolloutService interface {
	Get(
		ctx context.Context,
		tenant pagination.TenantInfo,
	) (*extractionrollout.ExtractionRollout, error)
	Update(
		ctx context.Context,
		req *UpdateExtractionRolloutRequest,
		actor *RequestActor,
	) (*extractionrollout.ExtractionRollout, error)
	Report(ctx context.Context, tenant pagination.TenantInfo) (*ExtractionRolloutReport, error)
}

type AssignExtractionRolloutRequest struct {
	TenantInfo  pagination.TenantInfo
	DocumentID  pulid.ID
	ExtractedAt int64
}

type SettleExtractionRolloutRequest struct {
	TenantInfo       pagination.TenantInfo
	DocumentID       pulid.ID
	ExtractedAt      int64
	Outcome          extractionrollout.Outcome
	ServedProviderID pulid.ID
	ServedModel      string
}

type ExtractionRolloutRouter interface {
	AssignExtraction(ctx context.Context, req *AssignExtractionRolloutRequest) (pulid.ID, error)
	SettleExtraction(ctx context.Context, req *SettleExtractionRolloutRequest) error
}

type ExtractionRolloutGuard interface {
	ObserveCorrection(ctx context.Context, correction *aicorrection.Correction) error
}

type ExtractionRolloutRetention interface {
	PurgeExpiredAssignments(
		ctx context.Context,
		req PurgeExpiredAICorrectionsRequest,
	) (int64, error)
}
