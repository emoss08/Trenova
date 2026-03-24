package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListGLAccountsRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetGLAccountByIDRequest struct {
	ID         pulid.ID              `json:"id"         form:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo" form:"tenantInfo"`
}

type GetGLAccountsByIDsRequest struct {
	TenantInfo   pagination.TenantInfo `json:"-"`
	GLAccountIDs []pulid.ID            `json:"glAccountIds"`
}

type BulkUpdateGLAccountStatusRequest struct {
	TenantInfo   pagination.TenantInfo `json:"-"`
	GLAccountIDs []pulid.ID            `json:"glAccountIds"`
	Status       domaintypes.Status    `json:"status"`
}

type GLAccountSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
}

type DeleteGLAccountRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GLAccountRepository interface {
	List(
		ctx context.Context,
		req *ListGLAccountsRequest,
	) (*pagination.ListResult[*glaccount.GLAccount], error)
	GetByID(
		ctx context.Context,
		req GetGLAccountByIDRequest,
	) (*glaccount.GLAccount, error)
	GetByIDs(
		ctx context.Context,
		req GetGLAccountsByIDsRequest,
	) ([]*glaccount.GLAccount, error)
	Create(
		ctx context.Context,
		entity *glaccount.GLAccount,
	) (*glaccount.GLAccount, error)
	Update(
		ctx context.Context,
		entity *glaccount.GLAccount,
	) (*glaccount.GLAccount, error)
	BulkUpdateStatus(
		ctx context.Context,
		req *BulkUpdateGLAccountStatusRequest,
	) ([]*glaccount.GLAccount, error)
	SelectOptions(
		ctx context.Context,
		req *GLAccountSelectOptionsRequest,
	) (*pagination.ListResult[*glaccount.GLAccount], error)
	Delete(
		ctx context.Context,
		req DeleteGLAccountRequest,
	) error
}
