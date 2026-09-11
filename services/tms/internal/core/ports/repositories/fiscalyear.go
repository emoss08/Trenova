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

type ListFiscalYearConnectionRequest struct {
	Filter            *pagination.QueryOptions `json:"filter"`
	Cursor            pagination.CursorInfo    `json:"-"`
	FiscalYearColumns []string                 `json:"-"`
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

type ReopenFiscalYearRequest struct {
	ID           pulid.ID              `json:"id"`
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
	ReopenedByID pulid.ID              `json:"reopenedById"`
	ReopenedAt   int64                 `json:"reopenedAt"`
	ReopenReason string                `json:"reopenReason"`
}

// GetNextFiscalYearRequest finds the fiscal year that picks up where another one
// stops, matched on the successor's start date rather than on the year number so
// that offset calendars resolve the same way calendar years do.
type GetNextFiscalYearRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	AfterDate  int64                 `json:"afterDate"`
}

type ActivateFiscalYearRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetCurrentFiscalYearRequest struct {
	OrgID pulid.ID `json:"orgId"`
	BuID  pulid.ID `json:"buId"`
}

// GetExpiredOpenFiscalYearsRequest finds fiscal years that are still Open after
// their end date has passed — the ones a controller owes a close.
type GetExpiredOpenFiscalYearsRequest struct {
	OrgID      pulid.ID `json:"orgId"`
	BuID       pulid.ID `json:"buId"`
	BeforeDate int64    `json:"beforeDate"`
}

type CountFiscalYearsByTenantRequest struct {
	OrgID pulid.ID `json:"orgId"`
	BuID  pulid.ID `json:"buId"`
}

type FiscalYearSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest
}

type FiscalYearRepository interface {
	List(
		ctx context.Context,
		req *ListFiscalYearsRequest,
	) (*pagination.ListResult[*fiscalyear.FiscalYear], error)
	ListConnection(
		ctx context.Context,
		req *ListFiscalYearConnectionRequest,
	) (*pagination.CursorListResult[*fiscalyear.FiscalYear], error)
	SelectOptions(
		ctx context.Context,
		req *FiscalYearSelectOptionsRequest,
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
	GetNextFiscalYear(
		ctx context.Context,
		req GetNextFiscalYearRequest,
	) (*fiscalyear.FiscalYear, error)
	GetExpiredOpenFiscalYears(
		ctx context.Context,
		req GetExpiredOpenFiscalYearsRequest,
	) ([]*fiscalyear.FiscalYear, error)
	Close(
		ctx context.Context,
		req CloseFiscalYearRequest,
	) (*fiscalyear.FiscalYear, error)
	Reopen(
		ctx context.Context,
		req ReopenFiscalYearRequest,
	) (*fiscalyear.FiscalYear, error)
	Activate(
		ctx context.Context,
		req ActivateFiscalYearRequest,
	) (*fiscalyear.FiscalYear, error)
}
