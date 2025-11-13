package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accounting"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type ListAccountTypeRequest struct {
	Filter *pagination.QueryOptions
}

type GetAccountTypeByIDRequest struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type AccountTypeSelectOptionsRequest struct {
	*pagination.SelectQueryOptions
}

type AccountTypeSelectOptionResponse struct {
	ID       pulid.ID `json:"id"       form:"id"       bun:"id"`
	Code     string   `json:"code"     form:"code"     bun:"code"`
	Name     string   `json:"name"     form:"name"     bun:"name"`
	Category string   `json:"category" form:"category" bun:"category"`
}

type AccountTypeRepository interface {
	List(
		ctx context.Context,
		req *ListAccountTypeRequest,
	) (*pagination.ListResult[*accounting.AccountType], error)
	GetOption(
		ctx context.Context,
		req GetAccountTypeByIDRequest,
	) (*accounting.AccountType, error)
	SelectOptions(
		ctx context.Context,
		req AccountTypeSelectOptionsRequest,
	) ([]*AccountTypeSelectOptionResponse, error)
	GetByID(
		ctx context.Context,
		req *GetAccountTypeByIDRequest,
	) (*accounting.AccountType, error)
	Create(ctx context.Context, at *accounting.AccountType) (*accounting.AccountType, error)
	Update(ctx context.Context, at *accounting.AccountType) (*accounting.AccountType, error)
}
