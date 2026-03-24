package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accounttype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListAccountTypesRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetAccountTypeByIDRequest struct {
	ID         pulid.ID              `json:"id"         form:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo" form:"tenantInfo"`
}

type GetAccountTypesByIDsRequest struct {
	TenantInfo     pagination.TenantInfo `json:"-"`
	AccountTypeIDs []pulid.ID            `json:"accountTypeIds"`
}

type BulkUpdateAccountTypeStatusRequest struct {
	TenantInfo     pagination.TenantInfo `json:"-"`
	AccountTypeIDs []pulid.ID            `json:"accountTypeIds"`
	Status         domaintypes.Status    `json:"status"`
}

type AccountTypeSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
}

type AccountTypeRepository interface {
	List(
		ctx context.Context,
		req *ListAccountTypesRequest,
	) (*pagination.ListResult[*accounttype.AccountType], error)
	GetByID(
		ctx context.Context,
		req GetAccountTypeByIDRequest,
	) (*accounttype.AccountType, error)
	GetByIDs(
		ctx context.Context,
		req GetAccountTypesByIDsRequest,
	) ([]*accounttype.AccountType, error)
	Create(
		ctx context.Context,
		entity *accounttype.AccountType,
	) (*accounttype.AccountType, error)
	Update(
		ctx context.Context,
		entity *accounttype.AccountType,
	) (*accounttype.AccountType, error)
	BulkUpdateStatus(
		ctx context.Context,
		req *BulkUpdateAccountTypeStatusRequest,
	) ([]*accounttype.AccountType, error)
	SelectOptions(
		ctx context.Context,
		req *AccountTypeSelectOptionsRequest,
	) (*pagination.ListResult[*accounttype.AccountType], error)
}
