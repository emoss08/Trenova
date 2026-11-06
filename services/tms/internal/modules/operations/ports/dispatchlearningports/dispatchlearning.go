package dispatchlearningports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchlearning"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type DecisionCursor struct {
	At int64
	ID pulid.ID
}

type ListDecisionsRequest struct {
	TenantInfo pagination.TenantInfo
	After      DecisionCursor
	Limit      int
}

type DecisionPage struct {
	Decisions []*dispatchlearning.Decision
	Scanned   int
	Last      DecisionCursor
}

type ListTenantsRequest struct {
	After pagination.TenantInfo
	Limit int
}

type Repository interface {
	Get(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*dispatchlearning.DispatchStyle, error)
	GetOrCreate(
		ctx context.Context,
		style *dispatchlearning.DispatchStyle,
	) (*dispatchlearning.DispatchStyle, error)
	Save(
		ctx context.Context,
		style *dispatchlearning.DispatchStyle,
	) (*dispatchlearning.DispatchStyle, error)
	ListDecisions(ctx context.Context, req *ListDecisionsRequest) (*DecisionPage, error)
	ListLearningTenants(
		ctx context.Context,
		req *ListTenantsRequest,
	) ([]pagination.TenantInfo, error)
}

type LearningStarter interface {
	StartLearning(ctx context.Context, tenantInfo pagination.TenantInfo) error
}

type MultiplierSource interface {
	Multipliers(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (map[dispatcheconomics.Component]float64, error)
}

type Style struct {
	Enabled       bool                             `json:"enabled"`
	Decisions     int64                            `json:"decisions"`
	Approvals     int64                            `json:"approvals"`
	Overrides     int64                            `json:"overrides"`
	Declines      int64                            `json:"declines"`
	LearningSince *int64                           `json:"learningSince"`
	UpdatedAt     *int64                           `json:"updatedAt"`
	Weights       []dispatchlearning.LearnedWeight `json:"weights"`
}

type GetStyleRequest struct {
	TenantInfo pagination.TenantInfo
}

type ResetStyleRequest struct {
	TenantInfo pagination.TenantInfo
	Actor      *permissionports.RequestActor
}

type LearnRequest struct {
	TenantInfo pagination.TenantInfo
}

type LearnResult struct {
	Scanned   int `json:"scanned"`
	Learned   int `json:"learned"`
	Approvals int `json:"approvals"`
	Overrides int `json:"overrides"`
	Declines  int `json:"declines"`
}

type Service interface {
	Get(ctx context.Context, req *GetStyleRequest) (*Style, error)
	Reset(ctx context.Context, req *ResetStyleRequest) (*Style, error)
}
