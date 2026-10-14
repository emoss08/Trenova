package factoringcompanyports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListFactoringCompanyRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type ListFactoringCompanyConnectionRequest struct {
	Filter                  *pagination.QueryOptions `json:"filter"`
	Cursor                  pagination.CursorInfo    `json:"-"`
	FactoringCompanyColumns []string                 `json:"-"`
}

type GetFactoringCompaniesByIDsRequest struct {
	IDs        []pulid.ID            `json:"ids"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

type GetFactoringCompanyByIDRequest struct {
	ID           pulid.ID              `json:"id"           form:"id"`
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"   form:"tenantInfo"`
	IncludeState bool                  `json:"includeState" form:"includeState"`
}

type FactoringCompanySelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
}

type CountAssignedCustomersRequest struct {
	TenantInfo         pagination.TenantInfo
	FactoringCompanyID pulid.ID
}

type FactoringCompanyRepository interface {
	List(
		ctx context.Context,
		req *ListFactoringCompanyRequest,
	) (*pagination.ListResult[*factoringcompany.FactoringCompany], error)
	ListConnection(
		ctx context.Context,
		req *ListFactoringCompanyConnectionRequest,
	) (*pagination.CursorListResult[*factoringcompany.FactoringCompany], error)
	GetByIDs(
		ctx context.Context,
		req GetFactoringCompaniesByIDsRequest,
	) ([]*factoringcompany.FactoringCompany, error)
	GetByID(
		ctx context.Context,
		req GetFactoringCompanyByIDRequest,
	) (*factoringcompany.FactoringCompany, error)
	Create(
		ctx context.Context,
		entity *factoringcompany.FactoringCompany,
	) (*factoringcompany.FactoringCompany, error)
	Update(
		ctx context.Context,
		entity *factoringcompany.FactoringCompany,
	) (*factoringcompany.FactoringCompany, error)
	SelectOptions(
		ctx context.Context,
		req *FactoringCompanySelectOptionsRequest,
	) (*pagination.ListResult[*factoringcompany.FactoringCompany], error)
	// CountAssignedCustomers is how many customers' invoices are assigned to
	// the factor.
	CountAssignedCustomers(ctx context.Context, req CountAssignedCustomersRequest) (int, error)
}
