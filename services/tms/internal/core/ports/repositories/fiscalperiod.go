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
	ID           pulid.ID              `json:"id"`
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
	ReopenReason string                `json:"reopenReason"`
	ReopenedByID pulid.ID              `json:"reopenedById"`
	ReopenedAt   int64                 `json:"reopenedAt"`
}

type LockFiscalPeriodRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	LockedByID pulid.ID              `json:"lockedById"`
	LockedAt   int64                 `json:"lockedAt"`
}

type ActivateFiscalPeriodRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type UnlockFiscalPeriodRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type CountUnclosedPeriodsByFiscalYearRequest struct {
	FiscalYearID pulid.ID `json:"fiscalYearId"`
	OrgID        pulid.ID `json:"orgId"`
	BuID         pulid.ID `json:"buId"`
}

type ListByFiscalYearIDRequest struct {
	FiscalYearID pulid.ID `json:"fiscalYearId"`
	OrgID        pulid.ID `json:"orgId"`
	BuID         pulid.ID `json:"buId"`
}

type ListByFiscalYearIDsRequest struct {
	TenantInfo    pagination.TenantInfo `json:"-"`
	FiscalYearIDs []pulid.ID            `json:"fiscalYearIds"`
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

type GetExpiredUnclosedPeriodsRequest struct {
	OrgID      pulid.ID `json:"orgId"`
	BuID       pulid.ID `json:"buId"`
	BeforeDate int64    `json:"beforeDate"`
}

type FiscalPeriodSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest
	FiscalYearID       pulid.ID
}

type FiscalPeriodRepository interface {
	List(
		ctx context.Context,
		req *ListFiscalPeriodsRequest,
	) (*pagination.ListResult[*fiscalperiod.FiscalPeriod], error)
	SelectOptions(
		ctx context.Context,
		req *FiscalPeriodSelectOptionsRequest,
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
	Activate(
		ctx context.Context,
		req ActivateFiscalPeriodRequest,
	) (*fiscalperiod.FiscalPeriod, error)
	CountUnclosedPeriodsByFiscalYear(
		ctx context.Context,
		req CountUnclosedPeriodsByFiscalYearRequest,
	) (int, error)
	ListByFiscalYearID(
		ctx context.Context,
		req ListByFiscalYearIDRequest,
	) ([]*fiscalperiod.FiscalPeriod, error)
	ListByFiscalYearIDForUpdate(
		ctx context.Context,
		req ListByFiscalYearIDRequest,
	) ([]*fiscalperiod.FiscalPeriod, error)
	ListByFiscalYearIDs(
		ctx context.Context,
		req ListByFiscalYearIDsRequest,
	) (map[pulid.ID][]*fiscalperiod.FiscalPeriod, error)
	GetPeriodByDate(
		ctx context.Context,
		req GetPeriodByDateRequest,
	) (*fiscalperiod.FiscalPeriod, error)
	CloseAllByFiscalYear(
		ctx context.Context,
		req CloseAllByFiscalYearRequest,
	) (int, error)
	GetExpiredUnclosedPeriods(
		ctx context.Context,
		req GetExpiredUnclosedPeriodsRequest,
	) ([]*fiscalperiod.FiscalPeriod, error)
}
