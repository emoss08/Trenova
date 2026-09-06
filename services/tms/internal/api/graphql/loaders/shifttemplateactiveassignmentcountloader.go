package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/services/schedulingservice"
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

func (f *ShiftTemplateActiveAssignmentCountLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, int] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *ShiftTemplateActiveAssignmentCountLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[int] {
	return batchCountFunc(func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]int, error) {
		return f.counter.CountTemplateAssignmentsByIDs(ctx, tenantInfo, ids)
	})
}
