package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/services/workercredentialservice"
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

func (f *WorkerCredentialTypeActiveCredentialCountLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, int] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *WorkerCredentialTypeActiveCredentialCountLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[int] {
	return batchCountFunc(func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]int, error) {
		return f.counter.CountActiveCredentialsByTypes(ctx, tenantInfo, ids)
	})
}
