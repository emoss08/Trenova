package agentloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type AIProviderMonthSpendLoaderFactoryParams struct {
	fx.In

	Spend aiproviderports.AIProviderSpendService
}

type AIProviderMonthSpendLoaderFactory struct {
	spend aiproviderports.AIProviderSpendService
}

func NewAIProviderMonthSpendLoaderFactory(
	p AIProviderMonthSpendLoaderFactoryParams,
) *AIProviderMonthSpendLoaderFactory {
	return &AIProviderMonthSpendLoaderFactory{spend: p.Spend}
}

func newAIProviderMonthSpendLoader(f *AIProviderMonthSpendLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, decimal.Decimal] {
	return loadercore.NewLoader(loadercore.BatchValueFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]decimal.Decimal, error) {
			return f.spend.MonthSpend(ctx, tenantInfo, ids)
		},
	))
}

var aiProviderMonthSpendKey = loadercore.NewKey(newAIProviderMonthSpendLoader)

func AIProviderMonthSpend(ctx context.Context, key string) (decimal.Decimal, error) {
	return loadercore.Load(ctx, aiProviderMonthSpendKey, key)
}

func (f *AIProviderMonthSpendLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
