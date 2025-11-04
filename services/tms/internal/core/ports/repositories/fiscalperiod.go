package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accounting"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type FiscalPeriodFilterOptions struct {
	IncludeUserDetails bool   `form:"includeUserDetails"`
	IncludeFiscalYear  bool   `form:"includeFiscalYear"`
	Status             string `form:"status"`
	FiscalYearID       string `form:"fiscalYearId"`
	PeriodNumber       int    `form:"periodNumber"`
}

type ListFiscalPeriodRequest struct {
	Filter        *pagination.QueryOptions
	FilterOptions FiscalPeriodFilterOptions `form:"filterOptions"`
}

type GetFiscalPeriodByIDRequest struct {
	FiscalPeriodID pulid.ID                  `form:"fiscalPeriodId"`
	OrgID          pulid.ID                  `form:"orgId"`
	BuID           pulid.ID                  `form:"buId"`
	UserID         pulid.ID                  `form:"userId"`
	FilterOptions  FiscalPeriodFilterOptions `form:"filterOptions"`
}

type GetFiscalPeriodByNumberRequest struct {
	FiscalYearID pulid.ID                  `form:"fiscalYearId"`
	PeriodNumber int                       `form:"periodNumber"`
	OrgID        pulid.ID                  `form:"orgId"`
	BuID         pulid.ID                  `form:"buId"`
	FilterOptions FiscalPeriodFilterOptions `form:"filterOptions"`
}

type GetFiscalPeriodsByYearRequest struct {
	FiscalYearID  pulid.ID                  `form:"fiscalYearId"`
	OrgID         pulid.ID                  `form:"orgId"`
	BuID          pulid.ID                  `form:"buId"`
	FilterOptions FiscalPeriodFilterOptions `form:"filterOptions"`
}

type CloseFiscalPeriodRequest struct {
	FiscalPeriodID pulid.ID `form:"fiscalPeriodId"`
	OrgID          pulid.ID `form:"orgId"`
	BuID           pulid.ID `form:"buId"`
	ClosedByID     pulid.ID `form:"closedById"`
	ClosedAt       int64    `form:"closedAt"`
}

type ReopenFiscalPeriodRequest struct {
	FiscalPeriodID pulid.ID `form:"fiscalPeriodId"`
	OrgID          pulid.ID `form:"orgId"`
	BuID           pulid.ID `form:"buId"`
	UserID         pulid.ID `form:"userId"`
}

type LockFiscalPeriodRequest struct {
	FiscalPeriodID pulid.ID `form:"fiscalPeriodId"`
	OrgID          pulid.ID `form:"orgId"`
	BuID           pulid.ID `form:"buId"`
	UserID         pulid.ID `form:"userId"`
}

type UnlockFiscalPeriodRequest struct {
	FiscalPeriodID pulid.ID `form:"fiscalPeriodId"`
	OrgID          pulid.ID `form:"orgId"`
	BuID           pulid.ID `form:"buId"`
	UserID         pulid.ID `form:"userId"`
}

type DeleteFiscalPeriodRequest struct {
	FiscalPeriodID pulid.ID `form:"fiscalPeriodId"`
	OrgID          pulid.ID `form:"orgId"`
	BuID           pulid.ID `form:"buId"`
	UserID         pulid.ID `form:"userId"`
}

type BulkCreateFiscalPeriodsRequest struct {
	Periods []*accounting.FiscalPeriod
	OrgID   pulid.ID
	BuID    pulid.ID
}

type FiscalPeriodRepository interface {
	List(
		ctx context.Context,
		opts *ListFiscalPeriodRequest,
	) (*pagination.ListResult[*accounting.FiscalPeriod], error)
	GetByID(
		ctx context.Context,
		opts *GetFiscalPeriodByIDRequest,
	) (*accounting.FiscalPeriod, error)
	GetByNumber(
		ctx context.Context,
		req *GetFiscalPeriodByNumberRequest,
	) (*accounting.FiscalPeriod, error)
	GetByFiscalYear(
		ctx context.Context,
		req *GetFiscalPeriodsByYearRequest,
	) ([]*accounting.FiscalPeriod, error)
	Create(ctx context.Context, fp *accounting.FiscalPeriod) (*accounting.FiscalPeriod, error)
	BulkCreate(ctx context.Context, req *BulkCreateFiscalPeriodsRequest) error
	Update(ctx context.Context, fp *accounting.FiscalPeriod) (*accounting.FiscalPeriod, error)
	Delete(ctx context.Context, req *DeleteFiscalPeriodRequest) error
	Close(ctx context.Context, req *CloseFiscalPeriodRequest) (*accounting.FiscalPeriod, error)
	Reopen(ctx context.Context, req *ReopenFiscalPeriodRequest) (*accounting.FiscalPeriod, error)
	Lock(ctx context.Context, req *LockFiscalPeriodRequest) (*accounting.FiscalPeriod, error)
	Unlock(ctx context.Context, req *UnlockFiscalPeriodRequest) (*accounting.FiscalPeriod, error)
}

