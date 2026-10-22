package ratematrixports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListRateMatrixRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type ListRateMatrixConnectionRequest struct {
	Filter            *pagination.QueryOptions `json:"filter"`
	Cursor            pagination.CursorInfo    `json:"-"`
	RateMatrixColumns []string                 `json:"-"`
}

type GetRateMatrixByIDRequest struct {
	RateMatrixID      pulid.ID              `json:"rateMatrixId"`
	TenantInfo        pagination.TenantInfo `json:"-"`
	IncludeDimensions bool                  `json:"includeDimensions"`
}

type GetRateMatrixByCodeRequest struct {
	Code       string                `json:"code"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

type LoadTableCellsRequest struct {
	TenantInfo   pagination.TenantInfo
	RateMatrixID pulid.ID
	Keys         map[int16][]string
}

type ListAxisKeysRequest struct {
	TenantInfo   pagination.TenantInfo
	RateMatrixID pulid.ID
	Position     int16
}

type ListTableCellsRequest struct {
	TenantInfo    pagination.TenantInfo
	RateMatrixIDs []pulid.ID
}

type ListRateMatrixCellsRequest struct {
	TenantInfo   pagination.TenantInfo    `json:"-"`
	RateMatrixID pulid.ID                 `json:"rateMatrixId"`
	Filter       *pagination.QueryOptions `json:"filter"`
}

type ReplaceRateMatrixCellsRequest struct {
	TenantInfo   pagination.TenantInfo        `json:"-"`
	RateMatrixID pulid.ID                     `json:"rateMatrixId"`
	Cells        []*ratematrix.RateMatrixCell `json:"cells"`
}

type RateMatrixRepository interface {
	List(
		ctx context.Context,
		req *ListRateMatrixRequest,
	) (*pagination.ListResult[*ratematrix.RateMatrix], error)
	ListConnection(
		ctx context.Context,
		req *ListRateMatrixConnectionRequest,
	) (*pagination.CursorListResult[*ratematrix.RateMatrix], error)
	SelectOptions(
		ctx context.Context,
		req *pagination.SelectQueryRequest,
	) (*pagination.ListResult[*ratematrix.RateMatrix], error)
	GetByID(ctx context.Context, req *GetRateMatrixByIDRequest) (*ratematrix.RateMatrix, error)
	GetByCode(ctx context.Context, req *GetRateMatrixByCodeRequest) (*ratematrix.RateMatrix, error)
	Create(ctx context.Context, entity *ratematrix.RateMatrix) (*ratematrix.RateMatrix, error)
	Update(ctx context.Context, entity *ratematrix.RateMatrix) (*ratematrix.RateMatrix, error)
	Delete(ctx context.Context, req *GetRateMatrixByIDRequest) error

	GetLookupStamp(ctx context.Context, tenantInfo pagination.TenantInfo) (string, error)
	ListTables(ctx context.Context, tenantInfo pagination.TenantInfo) ([]*ratematrix.RateMatrix, error)
	LoadTableCells(
		ctx context.Context,
		req *LoadTableCellsRequest,
	) ([]*ratematrix.RateMatrixCell, error)
	ListTableCells(
		ctx context.Context,
		req *ListTableCellsRequest,
	) ([]*ratematrix.RateMatrixCell, error)
	ListAxisKeys(ctx context.Context, req *ListAxisKeysRequest) ([]string, error)
	ListCells(
		ctx context.Context,
		req *ListRateMatrixCellsRequest,
	) (*pagination.ListResult[*ratematrix.RateMatrixCell], error)
	ReplaceCells(ctx context.Context, req *ReplaceRateMatrixCellsRequest) error
}
