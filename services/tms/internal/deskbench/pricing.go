package deskbench

import (
	"context"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
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

type pricer struct {
	providers agentports.AIProviderRepository
	reference agentports.ModelPriceReference

	mu    sync.Mutex
	known map[pulid.ID]pricedProvider
}

func newPricer(
	providers agentports.AIProviderRepository,
	reference agentports.ModelPriceReference,
) *pricer {
	return &pricer{
		providers: providers,
		reference: reference,
		known:     make(map[pulid.ID]pricedProvider, 8),
	}
}

type callPrice struct {
	cost   *decimal.Decimal
	source PriceSource
}

func (p *pricer) price(
	ctx context.Context,
	tenant pagination.TenantInfo,
	providerID pulid.ID,
	model string,
	usage aiprovider.TokenUsage,
	charged *decimal.Decimal,
) callPrice {
	if charged != nil {
		return callPrice{cost: charged, source: PriceSourceProvider}
	}
	if p == nil || providerID.IsNil() {
		return callPrice{}
	}

	priced := p.lookup(ctx, tenant, providerID, model)
	if priced.provider == nil || !priced.provider.Priced() {
		return callPrice{}
	}

	return callPrice{cost: priced.provider.CostFor(usage), source: priced.source}
}

func (p *pricer) lookup(
	ctx context.Context,
	tenant pagination.TenantInfo,
	providerID pulid.ID,
	model string,
) pricedProvider {
	p.mu.Lock()
	known, ok := p.known[providerID]
	p.mu.Unlock()
	if ok {
		return known
	}

	resolved := p.resolve(ctx, tenant, providerID, model)
	p.mu.Lock()
	p.known[providerID] = resolved
	p.mu.Unlock()

	return resolved
}

func (p *pricer) resolve(
	ctx context.Context,
	tenant pagination.TenantInfo,
	providerID pulid.ID,
	model string,
) pricedProvider {
	stored, err := p.providers.GetByID(ctx, agentports.GetAIProviderByIDRequest{
		ID:         providerID,
		TenantInfo: tenant,
	})
	if err != nil || stored == nil {
		return pricedProvider{}
	}
	if stored.Priced() {
		return pricedProvider{provider: stored, source: PriceSourceProvider}
	}

	estimated := *stored
	if model == "" {
		model = stored.Model
	}
	estimated.Model = model
	if card, listed := aiprovider.ListPrice(stored.Kind, model); listed {
		input, output := card.InputCostPerMillion, card.OutputCostPerMillion
		read, write := card.CacheReadCostPerMillion(), card.CacheWriteCostPerMillion()
		estimated.InputCostPerMillion = &input
		estimated.OutputCostPerMillion = &output
		estimated.CacheReadCostPerMillion = &read
		estimated.CacheWriteCostPerMillion = &write

		return pricedProvider{provider: &estimated, source: PriceSourceListPrice}
	}
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

func (b *Bench) priceOf(ctx context.Context, session *Session) callPricing {
	return func(call *ModelCall) callPrice {
		tokens := callTokens(call)

		return b.pricer.price(ctx, session.Tenant, tokens.provider, tokens.model, tokens.raw,
			tokens.charged)
	}
}

func (b *Bench) measure(ctx context.Context, session *Session, record *TurnRecord, from time.Time) {
	ctx = session.Context(ctx)
	priceOf := b.priceOf(ctx, session)
	ownerID := record.TurnID
	if record.RunID.IsNotNil() {
		ownerID = record.RunID
	}

	record.ModelCalls = b.recorder.For(record.ThreadID, ownerID, from, record.FinishedAt)
	sortCalls(record.ModelCalls)
	record.Calls = summarizeCalls(record.ModelCalls, priceOf)
	record.Usage, record.Models = usageOf(record.ModelCalls, priceOf)
	record.Unattributed = summarizeCalls(b.recorder.Unattributed(from, record.FinishedAt), priceOf)
	record.Disclosures = b.disclosures.For(record.ThreadID, ownerID, from, record.FinishedAt)
	record.FirstTokenMs = firstTokenOf(record)
}

func firstTokenOf(record *TurnRecord) *int64 {
	var first time.Time
	for _, call := range record.ModelCalls {
		if call.FirstTokenAt.IsZero() {
			continue
		}
		if first.IsZero() || call.FirstTokenAt.Before(first) {
			first = call.FirstTokenAt
		}
	}
	if first.IsZero() {
		return nil
	}
	elapsed := first.Sub(record.StartedAt).Milliseconds()

	return &elapsed
}
