package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type CustomerFilterOptions struct {
	IncludeState          bool `form:"includeState"`
	IncludeBillingProfile bool `form:"includeBillingProfile"`
	IncludeEmailProfile   bool `form:"includeEmailProfile"`
}

type ListCustomerRequest struct {
	CustomerFilterOptions `form:"customerFilterOptions"`
	Filter                *pagination.QueryOptions
}

type GetCustomerByIDRequest struct {
	ID                    pulid.ID
	OrgID                 pulid.ID
	BuID                  pulid.ID
	UserID                pulid.ID
	CustomerFilterOptions `form:"customerFilterOptions"`
}

type CustomerDocRequirementResponse struct {
	Name        string `json:"name"`
	DocID       string `json:"docId"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

type CustomerRepository interface {
	List(
		ctx context.Context,
		opts *ListCustomerRequest,
	) (*pagination.ListResult[*customer.Customer], error)
	GetByID(ctx context.Context, opts GetCustomerByIDRequest) (*customer.Customer, error)
	GetDocumentRequirements(
		ctx context.Context,
		cusID pulid.ID,
	) ([]*CustomerDocRequirementResponse, error)
	Create(ctx context.Context, c *customer.Customer) (*customer.Customer, error)
	Update(ctx context.Context, c *customer.Customer) (*customer.Customer, error)
}
