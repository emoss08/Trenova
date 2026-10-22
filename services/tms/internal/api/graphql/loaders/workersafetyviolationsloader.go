package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/app/workersafetyservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type safetyViolationsLister interface {
	ListViolations(
		ctx context.Context,
		req *workerports.ListWorkerSafetyViolationsRequest,
	) ([]*worker.WorkerSafetyViolation, error)
}

type WorkerSafetyViolationsByEventIDLoaderFactoryParams struct {
	fx.In

	WorkerSafetyService *workersafetyservice.Service
}

type WorkerSafetyViolationsByEventIDLoaderFactory struct {
	violations safetyViolationsLister
}

func NewWorkerSafetyViolationsByEventIDLoaderFactory(
	p WorkerSafetyViolationsByEventIDLoaderFactoryParams,
) *WorkerSafetyViolationsByEventIDLoaderFactory {
	return &WorkerSafetyViolationsByEventIDLoaderFactory{violations: p.WorkerSafetyService}
}

func newWorkerSafetyViolationsByEventIDLoader(f *WorkerSafetyViolationsByEventIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*worker.WorkerSafetyViolation] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *WorkerSafetyViolationsByEventIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[[]*worker.WorkerSafetyViolation] {
	return batchGroupFunc(
		func(
			ctx context.Context,
			ids []pulid.ID,
		) (map[pulid.ID][]*worker.WorkerSafetyViolation, error) {
			violations, err := f.violations.ListViolations(
				ctx,
				&workerports.ListWorkerSafetyViolationsRequest{
					TenantInfo:     tenantInfo,
					SafetyEventIDs: ids,
				},
			)
			if err != nil {
				return nil, err
			}

			groups := make(map[pulid.ID][]*worker.WorkerSafetyViolation, len(ids))
			for _, violation := range violations {
				groups[violation.SafetyEventID] = append(
					groups[violation.SafetyEventID],
					violation,
				)
			}

			return groups, nil
		},
	)
}
