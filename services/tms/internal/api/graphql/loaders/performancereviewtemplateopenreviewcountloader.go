package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/services/performancereviewservice"
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

func (f *PerformanceReviewTemplateOpenReviewCountLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, int] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *PerformanceReviewTemplateOpenReviewCountLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[int] {
	return batchCountFunc(func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]int, error) {
		return f.counter.CountOpenReviewsByTemplates(ctx, tenantInfo, ids)
	})
}
