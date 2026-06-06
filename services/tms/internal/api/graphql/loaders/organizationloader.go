package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/graph-gophers/dataloader/v7"
	"go.uber.org/fx"
)

type OrganizationByIDLoaderFactoryParams struct {
	fx.In

	OrganizationService services.OrganizationService
}

type OrganizationByIDLoaderFactory struct {
	organizationService services.OrganizationService
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
) *dataloader.Loader[string, *tenant.Organization] {
	return dataloader.NewBatchedLoader(f.batchFunc(tenantInfo))
}

func (f *OrganizationByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) dataloader.BatchFunc[string, *tenant.Organization] {
	return batchByIDFunc(func(ctx context.Context, ids []pulid.ID) ([]*tenant.Organization, error) {
		return f.organizationService.GetByIDs(ctx, services.GetOrganizationsByIDsRequest{
			TenantInfo:      tenantInfo,
			OrganizationIDs: ids,
			IncludeState:    true,
			IncludeBU:       true,
		})
	}, "Organization not found")
}
