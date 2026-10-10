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

type usersByIDsGetter interface {
	GetByIDs(ctx context.Context, req tenantports.GetUsersByIDsRequest) ([]*tenant.User, error)
}

type UserByIDLoaderFactoryParams struct {
	fx.In

	Repo tenantports.UserRepository
}

type UserByIDLoaderFactory struct {
	users usersByIDsGetter
}

func NewUserByIDLoaderFactory(p UserByIDLoaderFactoryParams) *UserByIDLoaderFactory {
	return &UserByIDLoaderFactory{
		users: p.Repo,
	}
}

func (f *UserByIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *tenant.User] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *UserByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[*tenant.User] {
	return batchByIDFunc(func(ctx context.Context, ids []pulid.ID) ([]*tenant.User, error) {
		return f.users.GetByIDs(ctx, tenantports.GetUsersByIDsRequest{
			TenantInfo: tenantInfo,
			UserIDs:    ids,
		})
	}, "User not found within your organization")
}
