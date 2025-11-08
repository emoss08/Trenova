package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accounting"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type GLAccountSelectOptionsRequest struct {
	*pagination.SelectQueryOptions
}

type GLAccountSelectOptionResponse struct {
	ID          pulid.ID `json:"id"          form:"id"          bun:"id"`
	AccountCode string   `json:"accountCode" form:"accountCode" bun:"account_code"`
	Name        string   `json:"name"        form:"name"        bun:"name"`
}

type GLAccountFilterOptions struct {
	IncludeAccountType bool   `form:"includeAccountType"`
	IncludeParent      bool   `form:"includeParent"`
	IncludeChildren    bool   `form:"includeChildren"`
	Status             string `form:"status"`
	AccountTypeID      string `form:"accountTypeId"`
	ParentID           string `form:"parentId"`
	IsActive           *bool  `form:"isActive"`
	IsSystem           *bool  `form:"isSystem"`
	AllowManualJE      *bool  `form:"allowManualJE"`
}

type ListGLAccountRequest struct {
	Filter        *pagination.QueryOptions
	FilterOptions *GLAccountFilterOptions `form:"filterOptions"`
}

type GetGLAccountByIDRequest struct {
	GLAccountID   pulid.ID                `form:"glAccountId"`
	OrgID         pulid.ID                `form:"orgId"`
	BuID          pulid.ID                `form:"buId"`
	UserID        pulid.ID                `form:"userId"`
	FilterOptions *GLAccountFilterOptions `form:"filterOptions"`
}

type GetGLAccountByCodeRequest struct {
	AccountCode   string                  `form:"accountCode"`
	OrgID         pulid.ID                `form:"orgId"`
	BuID          pulid.ID                `form:"buId"`
	FilterOptions *GLAccountFilterOptions `form:"filterOptions"`
}

type GetGLAccountsByTypeRequest struct {
	AccountTypeID pulid.ID                `form:"accountTypeId"`
	OrgID         pulid.ID                `form:"orgId"`
	BuID          pulid.ID                `form:"buId"`
	FilterOptions *GLAccountFilterOptions `form:"filterOptions"`
}

type GetGLAccountsByParentRequest struct {
	ParentID      pulid.ID                `form:"parentId"`
	OrgID         pulid.ID                `form:"orgId"`
	BuID          pulid.ID                `form:"buId"`
	FilterOptions *GLAccountFilterOptions `form:"filterOptions"`
}

type GetGLAccountHierarchyRequest struct {
	OrgID         pulid.ID                `form:"orgId"`
	BuID          pulid.ID                `form:"buId"`
	FilterOptions *GLAccountFilterOptions `form:"filterOptions"`
}

type DeleteGLAccountRequest struct {
	GLAccountID pulid.ID `form:"glAccountId"`
	OrgID       pulid.ID `form:"orgId"`
	BuID        pulid.ID `form:"buId"`
	UserID      pulid.ID `form:"userId"`
}

type BulkCreateGLAccountsRequest struct {
	Accounts []*accounting.GLAccount
	OrgID    pulid.ID
	BuID     pulid.ID
}

type UpdateGLAccountBalanceRequest struct {
	GLAccountID    pulid.ID `form:"glAccountId"`
	OrgID          pulid.ID `form:"orgId"`
	BuID           pulid.ID `form:"buId"`
	DebitAmount    int64    `form:"debitAmount"`
	CreditAmount   int64    `form:"creditAmount"`
	CurrentBalance int64    `form:"currentBalance"`
}

type GLAccountRepository interface {
	List(
		ctx context.Context,
		opts *ListGLAccountRequest,
	) (*pagination.ListResult[*accounting.GLAccount], error)
	GetOption(
		ctx context.Context,
		req GetGLAccountByIDRequest,
	) (*accounting.GLAccount, error)
	SelectOptions(
		ctx context.Context,
		req GLAccountSelectOptionsRequest,
	) ([]*GLAccountSelectOptionResponse, error)
	GetByID(
		ctx context.Context,
		opts *GetGLAccountByIDRequest,
	) (*accounting.GLAccount, error)
	GetByCode(
		ctx context.Context,
		req *GetGLAccountByCodeRequest,
	) (*accounting.GLAccount, error)
	GetByType(
		ctx context.Context,
		req *GetGLAccountsByTypeRequest,
	) ([]*accounting.GLAccount, error)
	GetByParent(
		ctx context.Context,
		req *GetGLAccountsByParentRequest,
	) ([]*accounting.GLAccount, error)
	GetHierarchy(
		ctx context.Context,
		req *GetGLAccountHierarchyRequest,
	) ([]*accounting.GLAccount, error)
	Create(ctx context.Context, gla *accounting.GLAccount) (*accounting.GLAccount, error)
	BulkCreate(ctx context.Context, req *BulkCreateGLAccountsRequest) error
	Update(ctx context.Context, gla *accounting.GLAccount) (*accounting.GLAccount, error)
	UpdateBalance(
		ctx context.Context,
		req *UpdateGLAccountBalanceRequest,
	) (*accounting.GLAccount, error)
	Delete(ctx context.Context, req *DeleteGLAccountRequest) error
}
