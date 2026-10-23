package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type FormulaTemplateStatsLoaderFactoryParams struct {
	fx.In

	TemplateRepo formulatemplateports.FormulaTemplateRepository
}

type FormulaTemplateStatsLoaderFactory struct {
	templateRepo formulatemplateports.FormulaTemplateRepository
}

func NewFormulaTemplateStatsLoaderFactory(
	p FormulaTemplateStatsLoaderFactoryParams,
) *FormulaTemplateStatsLoaderFactory {
	return &FormulaTemplateStatsLoaderFactory{templateRepo: p.TemplateRepo}
}

func newFormulaTemplateStatsLoader(f *FormulaTemplateStatsLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, formulatemplateports.TemplateStats] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *FormulaTemplateStatsLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[formulatemplateports.TemplateStats] {
	return func(ctx context.Context, keys []string) ([]formulatemplateports.TemplateStats, []error) {
		values := make([]formulatemplateports.TemplateStats, len(keys))
		errs := make([]error, len(keys))

		ids, indexesByID := parseBatchKeys(keys, errs)
		if len(ids) == 0 {
			return values, errs
		}

		stats, err := f.templateRepo.CountStatsByIDs(
			ctx,
			&formulatemplateports.GetFormulaTemplateStatsRequest{
				TenantInfo:  tenantInfo,
				TemplateIDs: ids,
			},
		)
		if err != nil {
			fillMissingErrors(errs, err)
			return values, errs
		}

		for _, id := range ids {
			for _, idx := range indexesByID[id] {
				values[idx] = stats[id]
			}
		}

		return values, errs
	}
}
