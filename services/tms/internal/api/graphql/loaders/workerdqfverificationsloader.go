package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/app/workerdqfservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment"
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
	) (map[pulid.ID][]*employment.WorkerEmploymentVerification, error)
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

func newWorkerDQFVerificationsByWorkerIDLoader(f *WorkerDQFVerificationsByWorkerIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*employment.WorkerEmploymentVerification] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *WorkerDQFVerificationsByWorkerIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[[]*employment.WorkerEmploymentVerification] {
	return batchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*employment.WorkerEmploymentVerification, error) {
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
