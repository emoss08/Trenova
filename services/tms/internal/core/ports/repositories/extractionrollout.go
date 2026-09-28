package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/extractionrollout"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ExtractionRolloutRepository interface {
	Get(
		ctx context.Context,
		tenant pagination.TenantInfo,
	) (*extractionrollout.ExtractionRollout, error)
	Save(
		ctx context.Context,
		entity *extractionrollout.ExtractionRollout,
	) (*extractionrollout.ExtractionRollout, error)
}

type GetRolloutAssignmentRequest struct {
	TenantInfo  pagination.TenantInfo
	DocumentID  pulid.ID
	ExtractedAt int64
}

type TotalRolloutAssignmentsRequest struct {
	TenantInfo          pagination.TenantInfo
	CandidateProviderID pulid.ID
	Since               int64
}

type RolloutAssignmentTotal struct {
	Arm      extractionrollout.Arm      `bun:"arm"`
	ServedBy extractionrollout.ServedBy `bun:"served_by"`
	Outcome  extractionrollout.Outcome  `bun:"outcome"`
	Count    int                        `bun:"count"`
}

type PurgeRolloutAssignmentsRequest struct {
	TenantInfo pagination.TenantInfo
	Before     int64
	Limit      int
}

type RolloutAssignmentRepository interface {
	Create(
		ctx context.Context,
		entity *extractionrollout.RolloutAssignment,
	) (assignment *extractionrollout.RolloutAssignment, created bool, err error)
	GetByExtraction(
		ctx context.Context,
		req GetRolloutAssignmentRequest,
	) (*extractionrollout.RolloutAssignment, error)
	Save(
		ctx context.Context,
		entity *extractionrollout.RolloutAssignment,
	) (*extractionrollout.RolloutAssignment, error)
	Totals(
		ctx context.Context,
		req TotalRolloutAssignmentsRequest,
	) ([]RolloutAssignmentTotal, error)
	PurgeBefore(ctx context.Context, req PurgeRolloutAssignmentsRequest) (int64, error)
}
