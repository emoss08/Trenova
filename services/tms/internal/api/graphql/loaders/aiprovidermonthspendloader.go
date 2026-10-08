package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type AIProviderMonthSpendLoaderFactoryParams struct {
	fx.In

	Spend services.AIProviderSpendService
}

type AIProviderMonthSpendLoaderFactory struct {
	spend services.AIProviderSpendService
}

func NewAIProviderMonthSpendLoaderFactory(
	p AIProviderMonthSpendLoaderFactoryParams,
) *AIProviderMonthSpendLoaderFactory {
	return &AIProviderMonthSpendLoaderFactory{spend: p.Spend}
}

func (f *AIProviderMonthSpendLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, decimal.Decimal] {
	return newLoader(batchValueFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]decimal.Decimal, error) {
			return f.spend.MonthSpend(ctx, tenantInfo, ids)
		},
	))
}
