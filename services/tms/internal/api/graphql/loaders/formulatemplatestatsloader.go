package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type FormulaTemplateStatsLoaderFactoryParams struct {
	fx.In

	TemplateRepo repositories.FormulaTemplateRepository
}

// FormulaTemplateStatsLoaderFactory batches the per-row usage and scenario
// counts a list page shows, so a page of fifty templates costs one query.
type FormulaTemplateStatsLoaderFactory struct {
	templateRepo repositories.FormulaTemplateRepository
}

func NewFormulaTemplateStatsLoaderFactory(
	p FormulaTemplateStatsLoaderFactoryParams,
) *FormulaTemplateStatsLoaderFactory {
	return &FormulaTemplateStatsLoaderFactory{templateRepo: p.TemplateRepo}
}

func (f *FormulaTemplateStatsLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, repositories.TemplateStats] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *FormulaTemplateStatsLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[repositories.TemplateStats] {
	return func(ctx context.Context, keys []string) ([]repositories.TemplateStats, []error) {
		values := make([]repositories.TemplateStats, len(keys))
		errs := make([]error, len(keys))

		ids, indexesByID := parseBatchKeys(keys, errs)
		if len(ids) == 0 {
			return values, errs
		}

		stats, err := f.templateRepo.CountStatsByIDs(
			ctx,
			&repositories.GetFormulaTemplateStatsRequest{
				TenantInfo:  tenantInfo,
				TemplateIDs: ids,
			},
		)
		if err != nil {
			fillMissingErrors(errs, err)
			return values, errs
		}

		// A template with no consumers and no scenarios has no row; that is
		// a zero, not a miss.
		for _, id := range ids {
			for _, idx := range indexesByID[id] {
				values[idx] = stats[id]
			}
		}

		return values, errs
	}
}
