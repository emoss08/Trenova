package workforceloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workercredentialservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type workerCredentialTypeActiveCredentialCounter interface {
	CountActiveCredentialsByTypes(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		typeIDs []pulid.ID,
	) (map[pulid.ID]int, error)
}

type WorkerCredentialTypeActiveCredentialCountLoaderFactoryParams struct {
	fx.In

	WorkerCredentialService *workercredentialservice.Service
}

type WorkerCredentialTypeActiveCredentialCountLoaderFactory struct {
	counter workerCredentialTypeActiveCredentialCounter
}

func NewWorkerCredentialTypeActiveCredentialCountLoaderFactory(
	p WorkerCredentialTypeActiveCredentialCountLoaderFactoryParams,
) *WorkerCredentialTypeActiveCredentialCountLoaderFactory {
	return &WorkerCredentialTypeActiveCredentialCountLoaderFactory{
		counter: p.WorkerCredentialService,
	}
}

func newWorkerCredentialTypeActiveCredentialCountLoader(
	f *WorkerCredentialTypeActiveCredentialCountLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, int] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *WorkerCredentialTypeActiveCredentialCountLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[int] {
	return loadercore.BatchCountFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]int, error) {
			return f.counter.CountActiveCredentialsByTypes(ctx, tenantInfo, ids)
		},
	)
}

var workerCredentialTypeActiveCredentialCountKey = loadercore.NewKey(
	newWorkerCredentialTypeActiveCredentialCountLoader,
)

func WorkerCredentialTypeActiveCredentialCount(ctx context.Context, key string) (int, error) {
	return loadercore.Load(ctx, workerCredentialTypeActiveCredentialCountKey, key)
}

func (f *WorkerCredentialTypeActiveCredentialCountLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
