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

type AccountTypeRepository interface {
	List(
		ctx context.Context,
		req *ListAccountTypeRequest,
	) (*pagination.ListResult[*accounting.AccountType], error)
	GetByID(
		ctx context.Context,
		req *GetAccountTypeByIDRequest,
	) (*accounting.AccountType, error)
	Create(ctx context.Context, at *accounting.AccountType) (*accounting.AccountType, error)
	Update(ctx context.Context, at *accounting.AccountType) (*accounting.AccountType, error)
}
