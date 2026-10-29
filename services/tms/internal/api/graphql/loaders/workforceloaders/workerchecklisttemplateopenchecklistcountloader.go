package workforceloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerchecklistservice"
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

func newWorkerChecklistTemplateOpenChecklistCountLoader(
	f *WorkerChecklistTemplateOpenChecklistCountLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, int] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *WorkerChecklistTemplateOpenChecklistCountLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[int] {
	return loadercore.BatchCountFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]int, error) {
			return f.counter.CountOpenChecklistsForTemplates(ctx, tenantInfo, ids)
		},
	)
}

var workerChecklistTemplateOpenChecklistCountKey = loadercore.NewKey(
	newWorkerChecklistTemplateOpenChecklistCountLoader,
)

func WorkerChecklistTemplateOpenChecklistCount(ctx context.Context, key string) (int, error) {
	return loadercore.Load(ctx, workerChecklistTemplateOpenChecklistCountKey, key)
}

func (f *WorkerChecklistTemplateOpenChecklistCountLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
