package workforceloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerleaveservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerleave"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerleaveports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type leaveEntriesByCaseIDsLister interface {
	ListEntriesByCaseIDs(
		ctx context.Context,
		req *workerleaveports.ListLeaveEntriesByCaseIDsRequest,
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
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *WorkerLeaveEntriesByCaseIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[[]*workerleave.WorkerLeaveEntry] {
	return loadercore.BatchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*workerleave.WorkerLeaveEntry, error) {
			return f.entries.ListEntriesByCaseIDs(
				ctx,
				&workerleaveports.ListLeaveEntriesByCaseIDsRequest{
					TenantInfo:   tenantInfo,
					LeaveCaseIDs: ids,
				},
			)
		},
	)
}

var workerLeaveEntriesByCaseIDKey = loadercore.NewKey(newWorkerLeaveEntriesByCaseIDLoader)

func WorkerLeaveEntriesByCaseID(
	ctx context.Context,
	key string,
) ([]*workerleave.WorkerLeaveEntry, error) {
	return loadercore.Load(ctx, workerLeaveEntriesByCaseIDKey, key)
}

func (f *WorkerLeaveEntriesByCaseIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
