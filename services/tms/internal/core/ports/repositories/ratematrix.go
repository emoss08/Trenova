package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
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

// MatrixAxisQuery narrows one axis of a cell lookup.
//
// Exact axes carry the candidate keys, ordered most preferred first, because a
// place can belong to several zones at once. Banded axes carry the quantity,
// and the database narrows to the bands containing it.
type MatrixAxisQuery struct {
	Position int16
	Keys     []string
	Quantity decimal.NullDecimal
}

// LookupRateMatrixCellsRequest fetches the handful of cells that could price a
// lookup. Nothing loads a matrix whole: a class tariff runs to hundreds of
// thousands of cells and rating now happens on every shipment write.
type LookupRateMatrixCellsRequest struct {
	TenantInfo   pagination.TenantInfo `json:"-"`
	RateMatrixID pulid.ID              `json:"rateMatrixId"`
	Axes         []MatrixAxisQuery     `json:"axes"`
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

// GetRateMatrixLookupDataRequest fetches every matrix a formula's lookup()
// call could address: active, and exactly one axis. Multi-axis matrices are
// excluded on purpose — lookup() supplies a single key, and loading a class
// tariff whole to answer it would be the exact cost this shape exists to avoid.
type GetRateMatrixLookupDataRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
}

// RateMatrixLookupData is one lookup-addressable matrix with its cells.
//
// Cells ride beside the matrix rather than on it because the domain model
// deliberately has no whole-matrix cell relation: a single-axis matrix is the
// only shape small enough to load in full, and this type is how that exception
// stays an exception.
type RateMatrixLookupData struct {
	Matrix *ratematrix.RateMatrix
	Cells  []*ratematrix.RateMatrixCell
}

type GetDensityScaleRequest struct {
	DensityScaleID pulid.ID              `json:"densityScaleId"`
	TenantInfo     pagination.TenantInfo `json:"-"`
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

	LookupCells(
		ctx context.Context,
		req *LookupRateMatrixCellsRequest,
	) ([]*ratematrix.RateMatrixCell, error)
	GetLookupData(
		ctx context.Context,
		req *GetRateMatrixLookupDataRequest,
	) ([]*RateMatrixLookupData, error)
	ListCells(
		ctx context.Context,
		req *ListRateMatrixCellsRequest,
	) (*pagination.ListResult[*ratematrix.RateMatrixCell], error)
	ReplaceCells(ctx context.Context, req *ReplaceRateMatrixCellsRequest) error
}

type DensityScaleRepository interface {
	List(
		ctx context.Context,
		req *ListRateMatrixRequest,
	) (*pagination.ListResult[*ratematrix.DensityScale], error)
	GetByID(ctx context.Context, req *GetDensityScaleRequest) (*ratematrix.DensityScale, error)
	GetOrgDefault(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*ratematrix.DensityScale, error)
	Create(ctx context.Context, entity *ratematrix.DensityScale) (*ratematrix.DensityScale, error)
	Update(ctx context.Context, entity *ratematrix.DensityScale) (*ratematrix.DensityScale, error)
}
