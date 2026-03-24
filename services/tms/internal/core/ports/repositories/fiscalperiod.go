package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListFiscalPeriodsRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetFiscalPeriodByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type DeleteFiscalPeriodRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type BulkCreateFiscalPeriodsRequest struct {
	Periods    []*fiscalperiod.FiscalPeriod `json:"periods"`
	TenantInfo pagination.TenantInfo        `json:"tenantInfo"`
}

type CloseFiscalPeriodRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	ClosedByID pulid.ID              `json:"closedById"`
	ClosedAt   int64                 `json:"closedAt"`
}

type ReopenFiscalPeriodRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type LockFiscalPeriodRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type UnlockFiscalPeriodRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetOpenPeriodsCountByFiscalYearRequest struct {
	FiscalYearID pulid.ID `json:"fiscalYearId"`
	OrgID        pulid.ID `json:"orgId"`
	BuID         pulid.ID `json:"buId"`
}

type ListByFiscalYearIDRequest struct {
	FiscalYearID pulid.ID `json:"fiscalYearId"`
	OrgID        pulid.ID `json:"orgId"`
	BuID         pulid.ID `json:"buId"`
}

type GetPeriodByDateRequest struct {
	OrgID pulid.ID `json:"orgId"`
	BuID  pulid.ID `json:"buId"`
	Date  int64    `json:"date"`
}

type CloseAllByFiscalYearRequest struct {
	FiscalYearID pulid.ID `json:"fiscalYearId"`
	OrgID        pulid.ID `json:"orgId"`
	BuID         pulid.ID `json:"buId"`
	ClosedByID   pulid.ID `json:"closedById"`
	ClosedAt     int64    `json:"closedAt"`
}

type GetExpiredOpenPeriodsRequest struct {
	OrgID      pulid.ID `json:"orgId"`
	BuID       pulid.ID `json:"buId"`
	BeforeDate int64    `json:"beforeDate"`
}

type FiscalPeriodRepository interface {
	List(
		ctx context.Context,
		req *ListFiscalPeriodsRequest,
	) (*pagination.ListResult[*fiscalperiod.FiscalPeriod], error)
	GetByID(
		ctx context.Context,
		req GetFiscalPeriodByIDRequest,
	) (*fiscalperiod.FiscalPeriod, error)
	GetByIDForUpdate(
		ctx context.Context,
		req GetFiscalPeriodByIDRequest,
	) (*fiscalperiod.FiscalPeriod, error)
	Create(
		ctx context.Context,
		entity *fiscalperiod.FiscalPeriod,
	) (*fiscalperiod.FiscalPeriod, error)
	BulkCreate(
		ctx context.Context,
		req *BulkCreateFiscalPeriodsRequest,
	) error
	Update(
		ctx context.Context,
		entity *fiscalperiod.FiscalPeriod,
	) (*fiscalperiod.FiscalPeriod, error)
	Delete(
		ctx context.Context,
		req DeleteFiscalPeriodRequest,
	) error
	Close(
		ctx context.Context,
		req CloseFiscalPeriodRequest,
	) (*fiscalperiod.FiscalPeriod, error)
	Reopen(
		ctx context.Context,
		req ReopenFiscalPeriodRequest,
	) (*fiscalperiod.FiscalPeriod, error)
	Lock(
		ctx context.Context,
		req LockFiscalPeriodRequest,
	) (*fiscalperiod.FiscalPeriod, error)
	Unlock(
		ctx context.Context,
		req UnlockFiscalPeriodRequest,
	) (*fiscalperiod.FiscalPeriod, error)
	GetOpenPeriodsCountByFiscalYear(
		ctx context.Context,
		req GetOpenPeriodsCountByFiscalYearRequest,
	) (int, error)
	ListByFiscalYearID(
		ctx context.Context,
		req ListByFiscalYearIDRequest,
	) ([]*fiscalperiod.FiscalPeriod, error)
	ListByFiscalYearIDForUpdate(
		ctx context.Context,
		req ListByFiscalYearIDRequest,
	) ([]*fiscalperiod.FiscalPeriod, error)
	GetPeriodByDate(
		ctx context.Context,
		req GetPeriodByDateRequest,
	) (*fiscalperiod.FiscalPeriod, error)
	CloseAllByFiscalYear(
		ctx context.Context,
		req CloseAllByFiscalYearRequest,
	) (int, error)
	GetExpiredOpenPeriods(
		ctx context.Context,
		req GetExpiredOpenPeriodsRequest,
	) ([]*fiscalperiod.FiscalPeriod, error)
}
