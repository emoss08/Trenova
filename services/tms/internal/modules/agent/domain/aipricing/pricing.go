package aipricing

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/shopspring/decimal"
)

// TokenUsage is what one call consumed, as its provider reported it. Whether
// Input already counts the cached prompt tokens depends on the protocol
// (Kind.CachesOutsideInput).
type TokenUsage struct {
	Input      int
	Output     int
	CacheRead  int
	CacheWrite int
}

func (u TokenUsage) PromptTokens(kind aiprovider.Kind) int {
	prompt := max(u.Input, 0)
	if kind.CachesOutsideInput() {
		prompt += max(u.CacheRead, 0) + max(u.CacheWrite, 0)
	}

	return prompt
}

// CostFor prices a call. Nil without a price list: an unknown cost is not a
// free one, and a sum that silently counted it as zero would understate spend
// by exactly the providers nobody got round to pricing.
//
// Prompt tokens are priced in three parts: those read from the provider's
// cache, those written to it, and the rest at the input price.
func CostFor(p *aiprovider.Provider, usage TokenUsage) *decimal.Decimal {
	if !p.Priced() {
		return nil
	}
	inputTier, outputTier := tierFor(p, usage)
	cost := promptCost(p, usage).Mul(inputTier).
		Add(perMillion(*p.OutputCostPerMillion, usage.Output).Mul(outputTier))

	return &cost
}

// InputCostFor prices a call that returns no generated tokens, an embedding.
func InputCostFor(p *aiprovider.Provider, usage TokenUsage) *decimal.Decimal {
	if p.InputCostPerMillion == nil {
		return nil
	}
	cost := promptCost(p, usage)

	return &cost
}

func CostForTask(p *aiprovider.Provider, t aitask.Task, usage TokenUsage) *decimal.Decimal {
	if !t.Generates() {
		return InputCostFor(p, usage)
	}

	return CostFor(p, usage)
}

var BatchDiscount = decimal.RequireFromString("0.5")

func BatchCostForTask(p *aiprovider.Provider, t aitask.Task, usage TokenUsage) *decimal.Decimal {
	cost := CostForTask(p, t, usage)
	if cost == nil {
		return nil
	}
	discounted := cost.Mul(BatchDiscount)

	return &discounted
}

// CacheReadPrice is what a cached prompt token costs per million: the price
// entered for it, or the protocol's usual share of the input price.
func CacheReadPrice(p *aiprovider.Provider) *decimal.Decimal {
	if card, ok := listPrice(p); ok {
		return cachePrice(p, p.CacheReadCostPerMillion, card.CacheReadMultiple)
	}

	return cachePrice(p, p.CacheReadCostPerMillion, p.Kind.CacheReadMultiple())
}

// CacheWritePrice is what a prompt token written to the cache costs per
// million: the price entered for it, or the protocol's usual multiple of the
// input price.
func CacheWritePrice(p *aiprovider.Provider) *decimal.Decimal {
	if card, ok := listPrice(p); ok {
		return cachePrice(p, p.CacheWriteCostPerMillion, card.CacheWriteMultiple)
	}

	return cachePrice(p, p.CacheWriteCostPerMillion, p.Kind.CacheWriteMultiple())
}

func cachePrice(
	p *aiprovider.Provider,
	entered *decimal.Decimal,
	multiple decimal.Decimal,
) *decimal.Decimal {
	if entered != nil {
		return entered
	}
	if p.InputCostPerMillion == nil {
		return nil
	}
	price := p.InputCostPerMillion.Mul(multiple)

	return &price
}

func promptCost(p *aiprovider.Provider, usage TokenUsage) decimal.Decimal {
	read := max(usage.CacheRead, 0)
	write := max(usage.CacheWrite, 0)
	fresh := max(usage.Input, 0)
	if !p.Kind.CachesOutsideInput() {
		fresh = max(fresh-read-write, 0)
	}

	return perMillion(*p.InputCostPerMillion, fresh).
		Add(perMillion(*CacheReadPrice(p), read)).
		Add(perMillion(*CacheWritePrice(p), write))
}

func listPrice(p *aiprovider.Provider) (PriceCard, bool) {
	return ListPrice(p.Kind, p.Model)
}

func tierFor(p *aiprovider.Provider, usage TokenUsage) (input, output decimal.Decimal) {
	one := decimal.NewFromInt(1)
	card, ok := listPrice(p)
	if !ok || card.LongPrompt == nil {
		return one, one
	}
	if usage.PromptTokens(p.Kind) <= card.LongPrompt.AbovePromptTokens {
		return one, one
	}

	return card.LongPrompt.InputMultiple, card.LongPrompt.OutputMultiple
}

func perMillion(price decimal.Decimal, tokens int) decimal.Decimal {
	if tokens == 0 {
		return decimal.Zero
	}

	return price.Mul(decimal.NewFromInt(int64(tokens))).Div(oneMillion)
}

var oneMillion = decimal.NewFromInt(1_000_000)
