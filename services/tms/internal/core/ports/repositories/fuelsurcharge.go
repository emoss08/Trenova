package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListFuelIndexConnectionRequest struct {
	Filter           *pagination.QueryOptions `json:"filter"`
	Cursor           pagination.CursorInfo    `json:"-"`
	FuelIndexColumns []string                 `json:"-"`
}

type GetFuelIndexByIDRequest struct {
	FuelIndexID pulid.ID
	TenantInfo  pagination.TenantInfo
}

type FuelIndexSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest
}

type FuelIndexRepository interface {
	ListConnection(
		ctx context.Context,
		req *ListFuelIndexConnectionRequest,
	) (*pagination.CursorListResult[*fuelsurcharge.FuelIndex], error)

	ListActiveEIA(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*fuelsurcharge.FuelIndex, error)

	ListActive(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*fuelsurcharge.FuelIndex, error)

	GetByID(
		ctx context.Context,
		req *GetFuelIndexByIDRequest,
	) (*fuelsurcharge.FuelIndex, error)

	Create(
		ctx context.Context,
		entity *fuelsurcharge.FuelIndex,
	) (*fuelsurcharge.FuelIndex, error)

	Update(
		ctx context.Context,
		entity *fuelsurcharge.FuelIndex,
	) (*fuelsurcharge.FuelIndex, error)

	Delete(ctx context.Context, req *GetFuelIndexByIDRequest) error

	SelectOptions(
		ctx context.Context,
		req *FuelIndexSelectOptionsRequest,
	) (*pagination.ListResult[*fuelsurcharge.FuelIndex], error)
}

type ListFuelIndexPricesRequest struct {
	FuelIndexID pulid.ID
	TenantInfo  pagination.TenantInfo
	From        string
	To          string
	Limit       int
}

type GetLatestFuelPricesRequest struct {
	FuelIndexID pulid.ID
	TenantInfo  pagination.TenantInfo
	Date        string
	Limit       int
}

type LatestPricesPerIndexRequest struct {
	TenantInfo pagination.TenantInfo
	PerIndex   int
}

type GetFuelIndexPriceByIDRequest struct {
	PriceID    pulid.ID
	TenantInfo pagination.TenantInfo
}

type HasFuelPriceForDateRequest struct {
	FuelIndexID pulid.ID
	TenantInfo  pagination.TenantInfo
	Date        string
}

type FuelIndexPriceRepository interface {
	UpsertPrices(
		ctx context.Context,
		prices []*fuelsurcharge.FuelIndexPrice,
	) (int, error)

	ListByIndex(
		ctx context.Context,
		req *ListFuelIndexPricesRequest,
	) ([]*fuelsurcharge.FuelIndexPrice, error)

	GetLatestOnOrBefore(
		ctx context.Context,
		req *GetLatestFuelPricesRequest,
	) ([]*fuelsurcharge.FuelIndexPrice, error)

	LatestPerIndex(
		ctx context.Context,
		req *LatestPricesPerIndexRequest,
	) (map[pulid.ID][]*fuelsurcharge.FuelIndexPrice, error)

	HasPriceForDate(
		ctx context.Context,
		req *HasFuelPriceForDateRequest,
	) (bool, error)

	Create(
		ctx context.Context,
		entity *fuelsurcharge.FuelIndexPrice,
	) (*fuelsurcharge.FuelIndexPrice, error)

	Update(
		ctx context.Context,
		entity *fuelsurcharge.FuelIndexPrice,
	) (*fuelsurcharge.FuelIndexPrice, error)

	Delete(ctx context.Context, req *GetFuelIndexPriceByIDRequest) error

	GetByID(
		ctx context.Context,
		req *GetFuelIndexPriceByIDRequest,
	) (*fuelsurcharge.FuelIndexPrice, error)
}

type ListFuelSurchargeProgramConnectionRequest struct {
	Filter                      *pagination.QueryOptions `json:"filter"`
	Cursor                      pagination.CursorInfo    `json:"-"`
	FuelSurchargeProgramColumns []string                 `json:"-"`
}

type GetFuelSurchargeProgramByIDRequest struct {
	ProgramID     pulid.ID
	TenantInfo    pagination.TenantInfo
	IncludeRows   bool
	IncludeIndex  bool
	IncludeCharge bool
}

type FuelSurchargeProgramSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest
}

type FuelSurchargeProgramRepository interface {
	ListConnection(
		ctx context.Context,
		req *ListFuelSurchargeProgramConnectionRequest,
	) (*pagination.CursorListResult[*fuelsurcharge.FuelSurchargeProgram], error)

	ListActive(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*fuelsurcharge.FuelSurchargeProgram, error)

	GetByID(
		ctx context.Context,
		req *GetFuelSurchargeProgramByIDRequest,
	) (*fuelsurcharge.FuelSurchargeProgram, error)

	Create(
		ctx context.Context,
		entity *fuelsurcharge.FuelSurchargeProgram,
	) (*fuelsurcharge.FuelSurchargeProgram, error)

	Update(
		ctx context.Context,
		entity *fuelsurcharge.FuelSurchargeProgram,
	) (*fuelsurcharge.FuelSurchargeProgram, error)

	Delete(ctx context.Context, req *GetFuelSurchargeProgramByIDRequest) error

	SelectOptions(
		ctx context.Context,
		req *FuelSurchargeProgramSelectOptionsRequest,
	) (*pagination.ListResult[*fuelsurcharge.FuelSurchargeProgram], error)

	ListFallbackShipmentIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		limit int,
	) ([]pulid.ID, error)
}
