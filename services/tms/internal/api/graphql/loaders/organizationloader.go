package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type OrganizationByIDLoaderFactoryParams struct {
	fx.In

	OrganizationService tenantports.OrganizationService
}

type OrganizationByIDLoaderFactory struct {
	organizationService tenantports.OrganizationService
}

func NewOrganizationByIDLoaderFactory(
	p OrganizationByIDLoaderFactoryParams,
) *OrganizationByIDLoaderFactory {
	return &OrganizationByIDLoaderFactory{
		organizationService: p.OrganizationService,
	}
}

func (f *OrganizationByIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *tenant.Organization] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *OrganizationByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[*tenant.Organization] {
	return batchByIDFunc(func(ctx context.Context, ids []pulid.ID) ([]*tenant.Organization, error) {
		return f.organizationService.GetByIDs(ctx, tenantports.GetOrganizationsByIDsRequest{
			TenantInfo:      tenantInfo,
			OrganizationIDs: ids,
			IncludeState:    true,
			IncludeBU:       true,
		})
	}, "Organization not found")
}
