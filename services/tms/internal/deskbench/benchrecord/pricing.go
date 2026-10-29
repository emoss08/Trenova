package benchrecord

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aipricing"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type PriceSource string

const (
	PriceSourceProvider  = PriceSource("provider")
	PriceSourceListPrice = PriceSource("list")
	PriceSourceReference = PriceSource("reference")
	PriceSourceNone      = PriceSource("")
)

type pricedProvider struct {
	provider *aiprovider.Provider
	source   PriceSource
}

type Pricer struct {
	providers aiproviderports.AIProviderRepository
	reference aiproviderports.ModelPriceReference

	mu    sync.Mutex
	known map[priceKey]pricedProvider
}

type priceKey struct {
	provider pulid.ID
	model    string
}

func NewPricer(
	providers aiproviderports.AIProviderRepository,
	reference aiproviderports.ModelPriceReference,
) *Pricer {
	return &Pricer{
		providers: providers,
		reference: reference,
		known:     make(map[priceKey]pricedProvider, 8),
	}
}

type CallPrice struct {
	cost   *decimal.Decimal
	source PriceSource
}

func (p *Pricer) Price(
	ctx context.Context,
	tenant pagination.TenantInfo,
	tokens *tokenCount,
) CallPrice {
	if tokens.charged != nil {
		return CallPrice{cost: tokens.charged, source: PriceSourceProvider}
	}
	if p == nil || tokens.provider.IsNil() {
		return CallPrice{}
	}

	priced := p.lookup(ctx, tenant, tokens.provider, tokens.model)
	if priced.provider == nil || !priced.provider.Priced() {
		return CallPrice{}
	}
	return CallPrice{
		cost:   aipricing.AdvisedCostFor(priced.provider, tokens.raw, tokens.advisor),
		source: priced.source,
	}
}

func (p *Pricer) lookup(
	ctx context.Context,
	tenant pagination.TenantInfo,
	providerID pulid.ID,
	model string,
) pricedProvider {
	key := priceKey{provider: providerID, model: model}
	p.mu.Lock()
	known, ok := p.known[key]
	p.mu.Unlock()
	if ok {
		return known
	}

	resolved := p.resolve(ctx, tenant, providerID, model)
	p.mu.Lock()
	p.known[key] = resolved
	p.mu.Unlock()

	return resolved
}

func (p *Pricer) resolve(
	ctx context.Context,
	tenant pagination.TenantInfo,
	providerID pulid.ID,
	model string,
) pricedProvider {
	stored, err := p.providers.GetByID(ctx, aiproviderports.GetAIProviderByIDRequest{
		ID:         providerID,
		TenantInfo: tenant,
	})
	if err != nil || stored == nil {
		return pricedProvider{}
	}
	if stored.Priced() {
		return pricedProvider{provider: stored, source: PriceSourceProvider}
	}

	if model == "" {
		model = stored.Model
	}
	if listed, ok := aipricing.ListPriced(stored, model); ok {
		return pricedProvider{provider: listed, source: PriceSourceListPrice}
	}
	estimated := *stored
	estimated.Model = model
	if p.reference == nil {
		return pricedProvider{}
	}

	reference, found := p.reference.Prices(ctx, stored.Kind, []string{model})[model]
	if !found || reference.OutputCostPerMillion == nil {
		return pricedProvider{}
	}
	input := reference.InputCostPerMillion
	estimated.InputCostPerMillion = &input
	estimated.OutputCostPerMillion = reference.OutputCostPerMillion
	estimated.CacheReadCostPerMillion = reference.CacheReadCostPerMillion
	estimated.CacheWriteCostPerMillion = reference.CacheWriteCostPerMillion

	return pricedProvider{provider: &estimated, source: PriceSourceReference}
}
