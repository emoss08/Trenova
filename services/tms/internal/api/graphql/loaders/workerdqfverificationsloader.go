package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/services/workerdqfservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type employmentVerificationsByWorkerIDsLister interface {
	ListVerificationsByWorkerIDs(
		ctx context.Context,
		req *workerports.ListEmploymentVerificationsByWorkerIDsRequest,
	) (map[pulid.ID][]*worker.WorkerEmploymentVerification, error)
}

type WorkerDQFVerificationsByWorkerIDLoaderFactoryParams struct {
	fx.In

	WorkerDQFService *workerdqfservice.Service
}

type WorkerDQFVerificationsByWorkerIDLoaderFactory struct {
	verifications employmentVerificationsByWorkerIDsLister
}

func NewWorkerDQFVerificationsByWorkerIDLoaderFactory(
	p WorkerDQFVerificationsByWorkerIDLoaderFactoryParams,
) *WorkerDQFVerificationsByWorkerIDLoaderFactory {
	return &WorkerDQFVerificationsByWorkerIDLoaderFactory{
		verifications: p.WorkerDQFService,
	}
}

func (f *WorkerDQFVerificationsByWorkerIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*worker.WorkerEmploymentVerification] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *WorkerDQFVerificationsByWorkerIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[[]*worker.WorkerEmploymentVerification] {
	return batchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*worker.WorkerEmploymentVerification, error) {
			return f.verifications.ListVerificationsByWorkerIDs(
				ctx,
				&workerports.ListEmploymentVerificationsByWorkerIDsRequest{
					TenantInfo:      tenantInfo,
					WorkerIDs:       ids,
					IncludeDocument: true,
				},
			)
		},
	)
}
