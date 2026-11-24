package identityloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/personphoto"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/identity/app/userservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type userPhotoSigner interface {
	URLsByUser(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		userIDs []pulid.ID,
	) (map[pulid.ID]*personphoto.URLs, error)
}

type UserPhotoByIDLoaderFactoryParams struct {
	fx.In

	UserService *userservice.Service
}

type UserPhotoByIDLoaderFactory struct {
	photos userPhotoSigner
}

func NewUserPhotoByIDLoaderFactory(p UserPhotoByIDLoaderFactoryParams) *UserPhotoByIDLoaderFactory {
	return &UserPhotoByIDLoaderFactory{photos: p.UserService}
}

func newUserPhotoByIDLoader(
	f *UserPhotoByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *personphoto.URLs] {
	return loadercore.NewLoader(loadercore.BatchValueFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]*personphoto.URLs, error) {
			return f.photos.URLsByUser(ctx, tenantInfo, ids)
		},
	))
}

var userPhotoByIDKey = loadercore.NewKey(newUserPhotoByIDLoader)

func UserPhotoByID(ctx context.Context, userID pulid.ID) (*personphoto.URLs, error) {
	if userID.IsNil() {
		return nil, nil //nolint:nilnil // no user has no photo
	}
	return loadercore.Load(ctx, userPhotoByIDKey, userID.String())
}

func (f *UserPhotoByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
