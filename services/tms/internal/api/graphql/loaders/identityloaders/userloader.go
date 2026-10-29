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

func newUserByIDLoader(f *UserByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *tenant.User] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *UserByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[*tenant.User] {
	return loadercore.BatchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*tenant.User, error) {
			return f.users.GetByIDs(ctx, tenantports.GetUsersByIDsRequest{
				TenantInfo: tenantInfo,
				UserIDs:    ids,
			})
		},
		"User not found within your organization",
	)
}

var userByIDKey = loadercore.NewKey(newUserByIDLoader)

func UserByID(ctx context.Context, key string) (*tenant.User, error) {
	return loadercore.Load(ctx, userByIDKey, key)
}

func (f *UserByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
