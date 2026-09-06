package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/workerleaveservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type leaveEntriesByCaseIDsLister interface {
	ListEntriesByCaseIDs(
		ctx context.Context,
		req *repositories.ListLeaveEntriesByCaseIDsRequest,
	) (map[pulid.ID][]*worker.WorkerLeaveEntry, error)
}

type WorkerLeaveEntriesByCaseIDLoaderFactoryParams struct {
	fx.In

	WorkerLeaveService *workerleaveservice.Service
}

type WorkerLeaveEntriesByCaseIDLoaderFactory struct {
	entries leaveEntriesByCaseIDsLister
}

func NewWorkerLeaveEntriesByCaseIDLoaderFactory(
	p WorkerLeaveEntriesByCaseIDLoaderFactoryParams,
) *WorkerLeaveEntriesByCaseIDLoaderFactory {
	return &WorkerLeaveEntriesByCaseIDLoaderFactory{
		entries: p.WorkerLeaveService,
	}
}

func (f *WorkerLeaveEntriesByCaseIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*worker.WorkerLeaveEntry] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *WorkerLeaveEntriesByCaseIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[[]*worker.WorkerLeaveEntry] {
	return batchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*worker.WorkerLeaveEntry, error) {
			return f.entries.ListEntriesByCaseIDs(
				ctx,
				&repositories.ListLeaveEntriesByCaseIDsRequest{
					TenantInfo:   tenantInfo,
					LeaveCaseIDs: ids,
				},
			)
		},
	)
}
