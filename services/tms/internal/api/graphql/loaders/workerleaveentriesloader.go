package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/app/workerleaveservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerleave"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type leaveEntriesByCaseIDsLister interface {
	ListEntriesByCaseIDs(
		ctx context.Context,
		req *workerports.ListLeaveEntriesByCaseIDsRequest,
	) (map[pulid.ID][]*workerleave.WorkerLeaveEntry, error)
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

func newWorkerLeaveEntriesByCaseIDLoader(f *WorkerLeaveEntriesByCaseIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*workerleave.WorkerLeaveEntry] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *WorkerLeaveEntriesByCaseIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[[]*workerleave.WorkerLeaveEntry] {
	return batchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*workerleave.WorkerLeaveEntry, error) {
			return f.entries.ListEntriesByCaseIDs(
				ctx,
				&workerports.ListLeaveEntriesByCaseIDsRequest{
					TenantInfo:   tenantInfo,
					LeaveCaseIDs: ids,
				},
			)
		},
	)
}
