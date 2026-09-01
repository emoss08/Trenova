package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/graph-gophers/dataloader/v7"
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
) *dataloader.Loader[string, repositories.TemplateStats] {
	return dataloader.NewBatchedLoader(f.batchFunc(tenantInfo))
}

func (f *FormulaTemplateStatsLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) dataloader.BatchFunc[string, repositories.TemplateStats] {
	return func(
		ctx context.Context,
		keys []string,
	) []*dataloader.Result[repositories.TemplateStats] {
		results := make([]*dataloader.Result[repositories.TemplateStats], len(keys))

		ids := make([]pulid.ID, 0, len(keys))
		for _, key := range keys {
			parsed, err := pulid.MustParse(key)
			if err != nil {
				continue
			}
			ids = append(ids, parsed)
		}

		stats, err := f.templateRepo.CountStatsByIDs(
			ctx,
			&repositories.GetFormulaTemplateStatsRequest{
				TenantInfo:  tenantInfo,
				TemplateIDs: ids,
			},
		)

		for i, key := range keys {
			if err != nil {
				results[i] = &dataloader.Result[repositories.TemplateStats]{Error: err}
				continue
			}
			parsed, parseErr := pulid.MustParse(key)
			if parseErr != nil {
				results[i] = &dataloader.Result[repositories.TemplateStats]{Error: parseErr}
				continue
			}
			// A template with no consumers and no scenarios has no row; that is
			// a zero, not a miss.
			results[i] = &dataloader.Result[repositories.TemplateStats]{Data: stats[parsed]}
		}

		return results
	}
}
