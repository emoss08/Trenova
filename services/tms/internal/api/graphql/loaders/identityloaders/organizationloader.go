package identityloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
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

func newOrganizationByIDLoader(f *OrganizationByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *tenant.Organization] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *OrganizationByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[*tenant.Organization] {
	return loadercore.BatchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*tenant.Organization, error) {
			return f.organizationService.GetByIDs(ctx, tenantports.GetOrganizationsByIDsRequest{
				TenantInfo:      tenantInfo,
				OrganizationIDs: ids,
				IncludeState:    true,
				IncludeBU:       true,
			})
		},
		"Organization not found",
	)
}

var organizationByIDKey = loadercore.NewKey(newOrganizationByIDLoader)

func OrganizationByID(ctx context.Context, key string) (*tenant.Organization, error) {
	return loadercore.Load(ctx, organizationByIDKey, key)
}

func (f *OrganizationByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
