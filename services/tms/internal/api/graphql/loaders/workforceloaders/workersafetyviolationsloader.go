package workforceloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workersafetyservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/fleetsafety"
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
	) ([]*fleetsafety.WorkerSafetyViolation, error)
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
) *dataloadgen.Loader[string, []*fleetsafety.WorkerSafetyViolation] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *WorkerSafetyViolationsByEventIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[[]*fleetsafety.WorkerSafetyViolation] {
	return loadercore.BatchGroupFunc(
		func(
			ctx context.Context,
			ids []pulid.ID,
		) (map[pulid.ID][]*fleetsafety.WorkerSafetyViolation, error) {
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

			groups := make(map[pulid.ID][]*fleetsafety.WorkerSafetyViolation, len(ids))
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

var workerSafetyViolationsByEventIDKey = loadercore.NewKey(newWorkerSafetyViolationsByEventIDLoader)

func WorkerSafetyViolationsByEventID(
	ctx context.Context,
	key string,
) ([]*fleetsafety.WorkerSafetyViolation, error) {
	return loadercore.Load(ctx, workerSafetyViolationsByEventIDKey, key)
}

func (f *WorkerSafetyViolationsByEventIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
