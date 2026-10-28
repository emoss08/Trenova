package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/app/workersafetyservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersafety"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workersafetyports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type safetyViolationsLister interface {
	ListViolations(
		ctx context.Context,
		req *workersafetyports.ListWorkerSafetyViolationsRequest,
	) ([]*workersafety.WorkerSafetyViolation, error)
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
) *dataloadgen.Loader[string, []*workersafety.WorkerSafetyViolation] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *WorkerSafetyViolationsByEventIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[[]*workersafety.WorkerSafetyViolation] {
	return batchGroupFunc(
		func(
			ctx context.Context,
			ids []pulid.ID,
		) (map[pulid.ID][]*workersafety.WorkerSafetyViolation, error) {
			violations, err := f.violations.ListViolations(
				ctx,
				&workersafetyports.ListWorkerSafetyViolationsRequest{
					TenantInfo:     tenantInfo,
					SafetyEventIDs: ids,
				},
			)
			if err != nil {
				return nil, err
			}

			groups := make(map[pulid.ID][]*workersafety.WorkerSafetyViolation, len(ids))
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
