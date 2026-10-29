package workforceloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/workforce/app/schedulingservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type shiftTemplateAssignmentCounter interface {
	CountTemplateAssignmentsByIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		templateIDs []pulid.ID,
	) (map[pulid.ID]int, error)
}

type ShiftTemplateActiveAssignmentCountLoaderFactoryParams struct {
	fx.In

	SchedulingService *schedulingservice.Service
}

type ShiftTemplateActiveAssignmentCountLoaderFactory struct {
	counter shiftTemplateAssignmentCounter
}

func NewShiftTemplateActiveAssignmentCountLoaderFactory(
	p ShiftTemplateActiveAssignmentCountLoaderFactoryParams,
) *ShiftTemplateActiveAssignmentCountLoaderFactory {
	return &ShiftTemplateActiveAssignmentCountLoaderFactory{counter: p.SchedulingService}
}

func newShiftTemplateActiveAssignmentCountLoader(f *ShiftTemplateActiveAssignmentCountLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, int] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *ShiftTemplateActiveAssignmentCountLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[int] {
	return loadercore.BatchCountFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]int, error) {
			return f.counter.CountTemplateAssignmentsByIDs(ctx, tenantInfo, ids)
		},
	)
}

var shiftTemplateActiveAssignmentCountKey = loadercore.NewKey(
	newShiftTemplateActiveAssignmentCountLoader,
)

func ShiftTemplateActiveAssignmentCount(ctx context.Context, key string) (int, error) {
	return loadercore.Load(ctx, shiftTemplateActiveAssignmentCountKey, key)
}

func (f *ShiftTemplateActiveAssignmentCountLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
