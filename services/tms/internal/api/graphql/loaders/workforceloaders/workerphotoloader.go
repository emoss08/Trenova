package workforceloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/personphoto"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerphotoservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type workerPhotoSigner interface {
	URLsByWorker(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerIDs []pulid.ID,
	) (map[pulid.ID]*personphoto.URLs, error)
}

type WorkerPhotoByIDLoaderFactoryParams struct {
	fx.In

	WorkerPhotoService *workerphotoservice.Service
}

type WorkerPhotoByIDLoaderFactory struct {
	photos workerPhotoSigner
}

func NewWorkerPhotoByIDLoaderFactory(
	p WorkerPhotoByIDLoaderFactoryParams,
) *WorkerPhotoByIDLoaderFactory {
	return &WorkerPhotoByIDLoaderFactory{photos: p.WorkerPhotoService}
}

func newWorkerPhotoByIDLoader(
	f *WorkerPhotoByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *personphoto.URLs] {
	return loadercore.NewLoader(loadercore.BatchValueFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]*personphoto.URLs, error) {
			return f.photos.URLsByWorker(ctx, tenantInfo, ids)
		},
	))
}

var workerPhotoByIDKey = loadercore.NewKey(newWorkerPhotoByIDLoader)

func WorkerPhotoByID(ctx context.Context, workerID pulid.ID) (*personphoto.URLs, error) {
	if workerID.IsNil() {
		return nil, nil //nolint:nilnil // no worker has no photo
	}
	return loadercore.Load(ctx, workerPhotoByIDKey, workerID.String())
}

func (f *WorkerPhotoByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
