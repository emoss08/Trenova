package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/rateimport"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetRateImportBatchByIDRequest struct {
	RateImportBatchID pulid.ID              `json:"rateImportBatchId"`
	TenantInfo        pagination.TenantInfo `json:"-"`
	// IncludeRows loads the staged rows. A list does not want them and a sheet
	// can carry thousands.
	IncludeRows bool `json:"includeRows"`
}

type ListRateImportBatchesRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	// RateAgreementID narrows to one contract's imports, which is what the
	// agreement panel's tab shows.
	RateAgreementID *pulid.ID `json:"rateAgreementId"`
}

type ListRateImportRowsRequest struct {
	RateImportBatchID pulid.ID                 `json:"rateImportBatchId"`
	TenantInfo        pagination.TenantInfo    `json:"-"`
	Filter            *pagination.QueryOptions `json:"filter"`
	// FailedOnly shows only the rows that would not read, which on a good sheet
	// is none of thousands.
	FailedOnly bool `json:"failedOnly"`
}

type RateImportRepository interface {
	GetByID(
		ctx context.Context,
		req *GetRateImportBatchByIDRequest,
	) (*rateimport.RateImportBatch, error)
	List(
		ctx context.Context,
		req *ListRateImportBatchesRequest,
	) (*pagination.ListResult[*rateimport.RateImportBatch], error)
	ListRows(
		ctx context.Context,
		req *ListRateImportRowsRequest,
	) (*pagination.ListResult[*rateimport.RateImportRow], error)

	// Create stages a batch and its rows together, so a batch never exists
	// claiming a row count it cannot show.
	Create(
		ctx context.Context,
		entity *rateimport.RateImportBatch,
		rows []*rateimport.RateImportRow,
	) (*rateimport.RateImportBatch, error)

	Update(
		ctx context.Context,
		entity *rateimport.RateImportBatch,
	) (*rateimport.RateImportBatch, error)
}
