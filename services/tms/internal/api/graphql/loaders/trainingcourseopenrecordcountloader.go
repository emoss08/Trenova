package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/services/workertrainingservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type trainingCourseOpenRecordCounter interface {
	CountOpenRecordsByCourses(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		courseIDs []pulid.ID,
	) (map[pulid.ID]int, error)
}

type TrainingCourseOpenRecordCountLoaderFactoryParams struct {
	fx.In

	WorkerTrainingService *workertrainingservice.Service
}

type TrainingCourseOpenRecordCountLoaderFactory struct {
	counter trainingCourseOpenRecordCounter
}

func NewTrainingCourseOpenRecordCountLoaderFactory(
	p TrainingCourseOpenRecordCountLoaderFactoryParams,
) *TrainingCourseOpenRecordCountLoaderFactory {
	return &TrainingCourseOpenRecordCountLoaderFactory{counter: p.WorkerTrainingService}
}

func (f *TrainingCourseOpenRecordCountLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, int] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *TrainingCourseOpenRecordCountLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[int] {
	return batchCountFunc(func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]int, error) {
		return f.counter.CountOpenRecordsByCourses(ctx, tenantInfo, ids)
	})
}
