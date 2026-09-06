package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/services/workerchecklistservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type workerChecklistTemplateOpenChecklistCounter interface {
	CountOpenChecklistsForTemplates(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		templateIDs []pulid.ID,
	) (map[pulid.ID]int, error)
}

type WorkerChecklistTemplateOpenChecklistCountLoaderFactoryParams struct {
	fx.In

	WorkerChecklistService *workerchecklistservice.Service
}

type WorkerChecklistTemplateOpenChecklistCountLoaderFactory struct {
	counter workerChecklistTemplateOpenChecklistCounter
}

func NewWorkerChecklistTemplateOpenChecklistCountLoaderFactory(
	p WorkerChecklistTemplateOpenChecklistCountLoaderFactoryParams,
) *WorkerChecklistTemplateOpenChecklistCountLoaderFactory {
	return &WorkerChecklistTemplateOpenChecklistCountLoaderFactory{
		counter: p.WorkerChecklistService,
	}
}

func (f *WorkerChecklistTemplateOpenChecklistCountLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, int] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *WorkerChecklistTemplateOpenChecklistCountLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[int] {
	return batchCountFunc(func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]int, error) {
		return f.counter.CountOpenChecklistsForTemplates(ctx, tenantInfo, ids)
	})
}
