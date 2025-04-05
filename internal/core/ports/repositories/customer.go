package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type ListCustomerOptions struct {
	Filter                *ports.LimitOffsetQueryOptions
	IncludeState          bool `query:"includeState"`
	IncludeBillingProfile bool `query:"includeBillingProfile"`
	IncludeEmailProfile   bool `query:"includeEmailProfile"`
}

type GetCustomerByIDOptions struct {
	ID                    pulid.ID
	OrgID                 pulid.ID
	BuID                  pulid.ID
	UserID                pulid.ID
	IncludeState          bool `query:"includeState"`
	IncludeBillingProfile bool `query:"includeBillingProfile"`
	IncludeEmailProfile   bool `query:"includeEmailProfile"`
}

type CustomerRepository interface {
	List(ctx context.Context, opts *ListCustomerOptions) (*ports.ListResult[*customer.Customer], error)
	GetByID(ctx context.Context, opts GetCustomerByIDOptions) (*customer.Customer, error)
	Create(ctx context.Context, c *customer.Customer) (*customer.Customer, error)
	Update(ctx context.Context, c *customer.Customer) (*customer.Customer, error)
}
