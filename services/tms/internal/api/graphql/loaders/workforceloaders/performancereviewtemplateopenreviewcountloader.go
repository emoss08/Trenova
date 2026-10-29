package workforceloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/workforce/app/performancereviewservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type performanceReviewTemplateOpenReviewCounter interface {
	CountOpenReviewsByTemplates(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		templateIDs []pulid.ID,
	) (map[pulid.ID]int, error)
}

type PerformanceReviewTemplateOpenReviewCountLoaderFactoryParams struct {
	fx.In

	PerformanceReviewService *performancereviewservice.Service
}

type PerformanceReviewTemplateOpenReviewCountLoaderFactory struct {
	counter performanceReviewTemplateOpenReviewCounter
}

func NewPerformanceReviewTemplateOpenReviewCountLoaderFactory(
	p PerformanceReviewTemplateOpenReviewCountLoaderFactoryParams,
) *PerformanceReviewTemplateOpenReviewCountLoaderFactory {
	return &PerformanceReviewTemplateOpenReviewCountLoaderFactory{
		counter: p.PerformanceReviewService,
	}
}

func newPerformanceReviewTemplateOpenReviewCountLoader(
	f *PerformanceReviewTemplateOpenReviewCountLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, int] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *PerformanceReviewTemplateOpenReviewCountLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[int] {
	return loadercore.BatchCountFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]int, error) {
			return f.counter.CountOpenReviewsByTemplates(ctx, tenantInfo, ids)
		},
	)
}

var performanceReviewTemplateOpenReviewCountKey = loadercore.NewKey(
	newPerformanceReviewTemplateOpenReviewCountLoader,
)

func PerformanceReviewTemplateOpenReviewCount(ctx context.Context, key string) (int, error) {
	return loadercore.Load(ctx, performanceReviewTemplateOpenReviewCountKey, key)
}

func (f *PerformanceReviewTemplateOpenReviewCountLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
