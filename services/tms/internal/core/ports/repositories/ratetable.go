package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ratetable"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListRateTablesRequest struct {
	Filter     *pagination.QueryOptions
	LookupType string
	Active     *bool
}

type ListRateTableConnectionRequest struct {
	Filter           *pagination.QueryOptions `json:"filter"`
	Cursor           pagination.CursorInfo    `json:"-"`
	RateTableColumns []string                 `json:"-"`
}

type GetRateTableByIDRequest struct {
	RateTableID pulid.ID
	TenantInfo  pagination.TenantInfo
}

type GetRateTablesByKeysRequest struct {
	TenantInfo pagination.TenantInfo
	Keys       []string
}

type GetRateTableLookupDataRequest struct {
	TenantInfo pagination.TenantInfo
}

type RateTableSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest
}

type RateTableRepository interface {
	List(
		ctx context.Context,
		req *ListRateTablesRequest,
	) (*pagination.ListResult[*ratetable.RateTable], error)

	ListConnection(
		ctx context.Context,
		req *ListRateTableConnectionRequest,
	) (*pagination.CursorListResult[*ratetable.RateTable], error)

	GetByID(
		ctx context.Context,
		req *GetRateTableByIDRequest,
	) (*ratetable.RateTable, error)

	GetByKeys(
		ctx context.Context,
		req *GetRateTablesByKeysRequest,
	) ([]*ratetable.RateTable, error)

	GetLookupData(
		ctx context.Context,
		req *GetRateTableLookupDataRequest,
	) ([]*ratetable.RateTable, error)

	Create(ctx context.Context, entity *ratetable.RateTable) (*ratetable.RateTable, error)

	Update(ctx context.Context, entity *ratetable.RateTable) (*ratetable.RateTable, error)

	Delete(ctx context.Context, req *GetRateTableByIDRequest) error

	SelectOptions(
		ctx context.Context,
		req *RateTableSelectOptionsRequest,
	) (*pagination.ListResult[*ratetable.RateTable], error)
}
