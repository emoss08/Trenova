package workforceloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workertrainingservice"
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

func newTrainingCourseOpenRecordCountLoader(f *TrainingCourseOpenRecordCountLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, int] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *TrainingCourseOpenRecordCountLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[int] {
	return loadercore.BatchCountFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]int, error) {
			return f.counter.CountOpenRecordsByCourses(ctx, tenantInfo, ids)
		},
	)
}

var trainingCourseOpenRecordCountKey = loadercore.NewKey(newTrainingCourseOpenRecordCountLoader)

func TrainingCourseOpenRecordCount(ctx context.Context, key string) (int, error) {
	return loadercore.Load(ctx, trainingCourseOpenRecordCountKey, key)
}

func (f *TrainingCourseOpenRecordCountLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
