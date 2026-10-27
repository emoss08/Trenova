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
	ForUpdate         bool                  `json:"-"`
}

type GetRateMatricesByIDsRequest struct {
	TenantInfo    pagination.TenantInfo
	RateMatrixIDs []pulid.ID
}

type ListRateMatrixCellsRequest struct {
	TenantInfo    pagination.TenantInfo    `json:"-"`
	RateMatrixID  pulid.ID                 `json:"rateMatrixId"`
	VersionNumber int64                    `json:"versionNumber"`
	Filter        *pagination.QueryOptions `json:"filter"`
}

type ReplaceRateMatrixCellsRequest struct {
	TenantInfo   pagination.TenantInfo        `json:"-"`
	RateMatrixID pulid.ID                     `json:"rateMatrixId"`
	Cells        []*ratematrix.RateMatrixCell `json:"cells"`
}

type CopyVersionRequest struct {
	TenantInfo   pagination.TenantInfo
	RateMatrixID pulid.ID
	From         int64
	To           int64
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
	GetByIDs(
		ctx context.Context,
		req *GetRateMatricesByIDsRequest,
	) ([]*ratematrix.RateMatrix, error)
	Create(ctx context.Context, entity *ratematrix.RateMatrix) (*ratematrix.RateMatrix, error)
	Update(ctx context.Context, entity *ratematrix.RateMatrix) (*ratematrix.RateMatrix, error)
	UpdateHeader(
		ctx context.Context,
		entity *ratematrix.RateMatrix,
	) (*ratematrix.RateMatrix, error)
	Delete(ctx context.Context, req *GetRateMatrixByIDRequest) error

	ListCells(
		ctx context.Context,
		req *ListRateMatrixCellsRequest,
	) (*pagination.ListResult[*ratematrix.RateMatrixCell], error)
	ReplaceCells(ctx context.Context, req *ReplaceRateMatrixCellsRequest) error
	CopyVersion(ctx context.Context, req *CopyVersionRequest) (int64, error)
	CountMisfitCells(ctx context.Context, req *CountMisfitCellsRequest) (int64, error)
}

type CountMisfitCellsRequest struct {
	TenantInfo    pagination.TenantInfo
	RateMatrixID  pulid.ID
	VersionNumber int64
	Dimensions    []*ratematrix.RateMatrixDimension
}

type GetVersionRequest struct {
	TenantInfo    pagination.TenantInfo
	RateMatrixID  pulid.ID
	VersionNumber int64
}

type ListVersionsRequest struct {
	TenantInfo   pagination.TenantInfo
	RateMatrixID pulid.ID
	Limit        int
}

type GetVersionsByKeysRequest struct {
	TenantInfo pagination.TenantInfo
	Keys       []ratematrix.VersionKey
}

type LoadTableCellsRequest struct {
	TenantInfo pagination.TenantInfo
	Key        ratematrix.VersionKey
	Keys       map[int16][]string
}

type ListAxisKeysRequest struct {
	TenantInfo pagination.TenantInfo
	Key        ratematrix.VersionKey
	Position   int16
}

type ListTableCellsRequest struct {
	TenantInfo pagination.TenantInfo
	Keys       []ratematrix.VersionKey
}

type RateMatrixVersionRepository interface {
	Create(
		ctx context.Context,
		version *ratematrix.RateMatrixVersion,
	) (*ratematrix.RateMatrixVersion, error)
	Get(ctx context.Context, req *GetVersionRequest) (*ratematrix.RateMatrixVersion, error)
	List(ctx context.Context, req *ListVersionsRequest) ([]*ratematrix.RateMatrixVersion, error)
	GetByKeys(
		ctx context.Context,
		req *GetVersionsByKeysRequest,
	) ([]*ratematrix.RateMatrixVersion, error)
	ListServed(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*ratematrix.RateMatrixVersion, error)
	ServedStamp(ctx context.Context, tenantInfo pagination.TenantInfo) (string, error)

	LoadTableCells(
		ctx context.Context,
		req *LoadTableCellsRequest,
	) ([]*ratematrix.RateMatrixCell, error)
	ListTableCells(
		ctx context.Context,
		req *ListTableCellsRequest,
	) ([]*ratematrix.RateMatrixCell, error)
	ListAxisKeys(ctx context.Context, req *ListAxisKeysRequest) ([]string, error)
	ListAxisRanges(
		ctx context.Context,
		req *ListAxisKeysRequest,
	) ([]ratematrix.PostalRange, error)
}

type ListReviewsRequest struct {
	TenantInfo   pagination.TenantInfo
	RateMatrixID pulid.ID
	Limit        int
}

type ListStaleSubmissionsRequest struct {
	SubmittedBefore int64
	Limit           int
}

type RateMatrixReviewRepository interface {
	Create(
		ctx context.Context,
		entity *ratematrix.RateMatrixReview,
	) (*ratematrix.RateMatrixReview, error)
	ListByMatrix(
		ctx context.Context,
		req *ListReviewsRequest,
	) ([]*ratematrix.RateMatrixReview, error)
	Latest(
		ctx context.Context,
		rateMatrixID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*ratematrix.RateMatrixReview, error)
	ListStaleSubmissions(
		ctx context.Context,
		req *ListStaleSubmissionsRequest,
	) ([]*ratematrix.RateMatrix, error)
}
