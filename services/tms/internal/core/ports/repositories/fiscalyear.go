package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListFiscalYearsRequest struct {
	Filter         *pagination.QueryOptions `json:"filter"`
	IncludePeriods bool                     `json:"includePeriods"`
}

type GetFiscalYearByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type DeleteFiscalYearRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type CloseFiscalYearRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	ClosedByID pulid.ID              `json:"closedById"`
	ClosedAt   int64                 `json:"closedAt"`
}

type LockFiscalYearRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	LockedByID pulid.ID              `json:"lockedById"`
	LockedAt   int64                 `json:"lockedAt"`
}

type UnlockFiscalYearRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ActivateFiscalYearRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetCurrentFiscalYearRequest struct {
	OrgID pulid.ID `json:"orgId"`
	BuID  pulid.ID `json:"buId"`
}

type CountFiscalYearsByTenantRequest struct {
	OrgID pulid.ID `json:"orgId"`
	BuID  pulid.ID `json:"buId"`
}

type FiscalYearRepository interface {
	List(
		ctx context.Context,
		req *ListFiscalYearsRequest,
	) (*pagination.ListResult[*fiscalyear.FiscalYear], error)
	GetByID(
		ctx context.Context,
		req GetFiscalYearByIDRequest,
	) (*fiscalyear.FiscalYear, error)
	GetCurrentFiscalYear(
		ctx context.Context,
		req GetCurrentFiscalYearRequest,
	) (*fiscalyear.FiscalYear, error)
	GetCurrentFiscalYearForUpdate(
		ctx context.Context,
		req GetCurrentFiscalYearRequest,
	) (*fiscalyear.FiscalYear, error)
	CountByTenant(
		ctx context.Context,
		req CountFiscalYearsByTenantRequest,
	) (int, error)
	GetByIDForUpdate(
		ctx context.Context,
		req GetFiscalYearByIDRequest,
	) (*fiscalyear.FiscalYear, error)
	Create(
		ctx context.Context,
		entity *fiscalyear.FiscalYear,
	) (*fiscalyear.FiscalYear, error)
	Update(
		ctx context.Context,
		entity *fiscalyear.FiscalYear,
	) (*fiscalyear.FiscalYear, error)
	Delete(
		ctx context.Context,
		req DeleteFiscalYearRequest,
	) error
	Close(
		ctx context.Context,
		req CloseFiscalYearRequest,
	) (*fiscalyear.FiscalYear, error)
	Activate(
		ctx context.Context,
		req ActivateFiscalYearRequest,
	) (*fiscalyear.FiscalYear, error)
}
