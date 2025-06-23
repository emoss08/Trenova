package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

var CustomerFieldConfig = &ports.FieldConfiguration{
	FilterableFields: map[string]bool{
		"name":         true,
		"code":         true,
		"status":       true,
		"customerType": true,
		"email":        true,
		"city":         true,
		"postalCode":   true,
		"createdAt":    true,
		"updatedAt":    true,
	},
	SortableFields: map[string]bool{
		"name":         true,
		"code":         true,
		"status":       true,
		"customerType": true,
		"createdAt":    true,
		"updatedAt":    true,
	},
	FieldMap: map[string]string{
		"name":         "name",
		"code":         "code",
		"status":       "status",
		"customerType": "customer_type",
		"email":        "email",
		"createdAt":    "created_at",
		"updatedAt":    "updated_at",
		"city":         "city",
		"postalCode":   "postal_code",
	},
	EnumMap: map[string]bool{
		"status": true,
	},
}

type ListCustomerOptions struct {
	IncludeState          bool `query:"includeState"`
	IncludeBillingProfile bool `query:"includeBillingProfile"`
	IncludeEmailProfile   bool `query:"includeEmailProfile"`
	Filter                *ports.QueryOptions
}

// BuildCustomerListOptions is a specific helper for customer list options
func BuildCustomerListOptions(
	filter *ports.QueryOptions,
	additionalOpts *ListCustomerOptions,
) *ListCustomerOptions {
	return &ListCustomerOptions{
		Filter:                filter,
		IncludeState:          additionalOpts.IncludeState,
		IncludeBillingProfile: additionalOpts.IncludeBillingProfile,
		IncludeEmailProfile:   additionalOpts.IncludeEmailProfile,
	}
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

type CustomerDocRequirementResponse struct {
	Name        string `json:"name"`
	DocID       string `json:"docId"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

type CustomerRepository interface {
	List(
		ctx context.Context,
		opts *ListCustomerOptions,
	) (*ports.ListResult[*customer.Customer], error)
	GetByID(ctx context.Context, opts GetCustomerByIDOptions) (*customer.Customer, error)
	GetDocumentRequirements(
		ctx context.Context,
		cusID pulid.ID,
	) ([]*CustomerDocRequirementResponse, error)
	Create(ctx context.Context, c *customer.Customer) (*customer.Customer, error)
	Update(ctx context.Context, c *customer.Customer) (*customer.Customer, error)
}
