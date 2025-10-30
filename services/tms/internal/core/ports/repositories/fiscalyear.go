package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accounting"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type FiscalYearFilterOptions struct {
	IncludeUserDetails bool   `form:"includeUserDetails"`
	Status             string `form:"status"`
	Year               int    `form:"year"`
	IsCurrent          bool   `form:"isCurrent"`
}

type ListFiscalYearRequest struct {
	Filter        *pagination.QueryOptions
	FilterOptions FiscalYearFilterOptions `form:"filterOptions"`
}

type GetFiscalYearByIDRequest struct {
	FiscalYearID  pulid.ID                `form:"fiscalYearId"`
	OrgID         pulid.ID                `form:"orgId"`
	BuID          pulid.ID                `form:"buId"`
	UserID        pulid.ID                `form:"userId"`
	FilterOptions FiscalYearFilterOptions `form:"filterOptions"`
}

type GetFiscalYearByYearRequest struct {
	Year          int                     `form:"year"          json:"year"`
	OrgID         pulid.ID                `form:"orgId"         json:"orgId"`
	BuID          pulid.ID                `form:"buId"          json:"buId"`
	FilterOptions FiscalYearFilterOptions `form:"filterOptions"`
}

type GetCurrentFiscalYearRequest struct {
	OrgID         pulid.ID                `form:"orgId"`
	BuID          pulid.ID                `form:"buId"`
	FilterOptions FiscalYearFilterOptions `form:"filterOptions"`
}

type CloseFiscalYearRequest struct {
	FiscalYearID pulid.ID `form:"fiscalYearId"`
	OrgID        pulid.ID `form:"orgId"`
	BuID         pulid.ID `form:"buId"`
	ClosedByID   pulid.ID `form:"closedById"`
	ClosedAt     int64    `form:"closedAt"`
}

type LockFiscalYearRequest struct {
	FiscalYearID pulid.ID `form:"fiscalYearId"`
	OrgID        pulid.ID `form:"orgId"`
	BuID         pulid.ID `form:"buId"`
	LockedByID   pulid.ID `form:"lockedById"`
	LockedAt     int64    `form:"lockedAt"`
}

type UnlockFiscalYearRequest struct {
	FiscalYearID pulid.ID `form:"fiscalYearId"`
	OrgID        pulid.ID `form:"orgId"`
	BuID         pulid.ID `form:"buId"`
	UserID       pulid.ID `form:"userId"`
}

type ActivateFiscalYearRequest struct {
	FiscalYearID pulid.ID `form:"fiscalYearId"`
	OrgID        pulid.ID `form:"orgId"`
	BuID         pulid.ID `form:"buId"`
	UserID       pulid.ID `form:"userId"`
}

type DeleteFiscalYearRequest struct {
	FiscalYearID pulid.ID `form:"fiscalYearId"`
	OrgID        pulid.ID `form:"orgId"`
	BuID         pulid.ID `form:"buId"`
	UserID       pulid.ID `form:"userId"`
}

type CheckOverlappingFiscalYearsRequest struct {
	StartDate int64     `form:"startDate"`
	EndDate   int64     `form:"endDate"`
	OrgID     pulid.ID  `form:"orgId"`
	BuID      pulid.ID  `form:"buId"`
	ExcludeID *pulid.ID `form:"excludeId"`
}

type OverlappingFiscalYearResponse struct {
	FiscalYearID pulid.ID `json:"fiscalYearId"`
	Year         int      `json:"year"`
	Name         string   `json:"name"`
	StartDate    int64    `json:"startDate"`
	EndDate      int64    `json:"endDate"`
}

type FiscalYearRepository interface {
	List(
		ctx context.Context,
		opts *ListFiscalYearRequest,
	) (*pagination.ListResult[*accounting.FiscalYear], error)
	GetByID(
		ctx context.Context,
		opts *GetFiscalYearByIDRequest,
	) (*accounting.FiscalYear, error)
	GetByYear(
		ctx context.Context,
		req *GetFiscalYearByYearRequest,
	) (*accounting.FiscalYear, error)
	GetCurrent(
		ctx context.Context,
		req *GetCurrentFiscalYearRequest,
	) (*accounting.FiscalYear, error)
	Create(ctx context.Context, fy *accounting.FiscalYear) (*accounting.FiscalYear, error)
	Update(ctx context.Context, fy *accounting.FiscalYear) (*accounting.FiscalYear, error)
	Delete(ctx context.Context, req *DeleteFiscalYearRequest) error
	Close(ctx context.Context, req *CloseFiscalYearRequest) (*accounting.FiscalYear, error)
	Lock(ctx context.Context, req *LockFiscalYearRequest) (*accounting.FiscalYear, error)
	Unlock(ctx context.Context, req *UnlockFiscalYearRequest) (*accounting.FiscalYear, error)
	Activate(ctx context.Context, req *ActivateFiscalYearRequest) (*accounting.FiscalYear, error)

	CheckOverlappingFiscalYears(
		ctx context.Context,
		req *CheckOverlappingFiscalYearsRequest,
	) ([]*OverlappingFiscalYearResponse, error)
}
