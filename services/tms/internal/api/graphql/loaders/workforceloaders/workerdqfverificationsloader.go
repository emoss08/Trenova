package workforceloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerdqfservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerdqfports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type employmentVerificationsByWorkerIDsLister interface {
	ListVerificationsByWorkerIDs(
		ctx context.Context,
		req *workerdqfports.ListEmploymentVerificationsByWorkerIDsRequest,
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
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *WorkerDQFVerificationsByWorkerIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[[]*employment.WorkerEmploymentVerification] {
	return loadercore.BatchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*employment.WorkerEmploymentVerification, error) {
			return f.verifications.ListVerificationsByWorkerIDs(
				ctx,
				&workerdqfports.ListEmploymentVerificationsByWorkerIDsRequest{
					TenantInfo:      tenantInfo,
					WorkerIDs:       ids,
					IncludeDocument: true,
				},
			)
		},
	)
}

var workerDQFVerificationsByWorkerIDKey = loadercore.NewKey(
	newWorkerDQFVerificationsByWorkerIDLoader,
)

func WorkerDQFVerificationsByWorkerID(
	ctx context.Context,
	key string,
) ([]*employment.WorkerEmploymentVerification, error) {
	return loadercore.Load(ctx, workerDQFVerificationsByWorkerIDKey, key)
}

func (f *WorkerDQFVerificationsByWorkerIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
