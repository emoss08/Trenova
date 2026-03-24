package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type CustomerFilterOptions struct {
	IncludeState          bool `form:"includeState"`
	IncludeBillingProfile bool `form:"includeBillingProfile"`
	IncludeEmailProfile   bool `form:"includeEmailProfile"`
}

type ListCustomerRequest struct {
	Filter                *pagination.QueryOptions `json:"filter" form:"filter"`
	CustomerFilterOptions ` form:"customerFilterOptions"`
}

type GetCustomerByIDRequest struct {
	ID                    pulid.ID              `json:"id"         form:"id"`
	TenantInfo            pagination.TenantInfo `json:"tenantInfo" form:"tenantInfo"`
	CustomerFilterOptions ` form:"customerFilterOptions"`
}

type BulkUpdateCustomerStatusRequest struct {
	TenantInfo  pagination.TenantInfo `json:"-"`
	CustomerIDs []pulid.ID            `json:"customerIds"`
	Status      domaintypes.Status    `json:"status"`
}

type CustomerDocRequirementResponse struct {
	Name        string `json:"name"`
	DocID       string `json:"docId"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

type GetCustomersByIDsRequest struct {
	TenantInfo  pagination.TenantInfo `json:"-"`
	CustomerIDs []pulid.ID            `json:"customerIds"`
}

type CustomerSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
}

type CustomerRepository interface {
	List(
		ctx context.Context,
		req *ListCustomerRequest,
	) (*pagination.ListResult[*customer.Customer], error)
	GetByID(
		ctx context.Context,
		req GetCustomerByIDRequest,
	) (*customer.Customer, error)
	GetByIDs(
		ctx context.Context,
		req GetCustomersByIDsRequest,
	) ([]*customer.Customer, error)
	GetDocumentRequirements(
		ctx context.Context,
		cusID pulid.ID,
	) ([]*CustomerDocRequirementResponse, error)
	GetBillingProfile(
		ctx context.Context,
		cusID pulid.ID,
	) (*customer.CustomerBillingProfile, error)
	Create(
		ctx context.Context,
		entity *customer.Customer,
	) (*customer.Customer, error)
	Update(
		ctx context.Context,
		entity *customer.Customer,
	) (*customer.Customer, error)
	BulkUpdateStatus(
		ctx context.Context,
		req *BulkUpdateCustomerStatusRequest,
	) ([]*customer.Customer, error)
	SelectOptions(
		ctx context.Context,
		req *CustomerSelectOptionsRequest,
	) (*pagination.ListResult[*customer.Customer], error)
}
