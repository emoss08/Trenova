package aiprovider

import (
	"github.com/shopspring/decimal"
)

type PriceTier struct {
	AbovePromptTokens int
	InputMultiple     decimal.Decimal
	OutputMultiple    decimal.Decimal
}

type PriceCard struct {
	InputCostPerMillion  decimal.Decimal
	OutputCostPerMillion decimal.Decimal
	CacheReadMultiple    decimal.Decimal
	CacheWriteMultiple   decimal.Decimal
	LongPrompt           *PriceTier
}

func (c *PriceCard) CacheReadCostPerMillion() decimal.Decimal {
	return c.InputCostPerMillion.Mul(c.CacheReadMultiple)
}

func (c *PriceCard) CacheWriteCostPerMillion() decimal.Decimal {
	return c.InputCostPerMillion.Mul(c.CacheWriteMultiple)
}

type claudePriceKey struct {
	family  ClaudeFamily
	version int
}

type claudeListPrice struct {
	input, output, cacheRead string
	longPrompt               *PriceTier
}

var (
	claudeCacheWriteMultiple = anthropicCacheWriteMultiple
	haikuLongPrompt          = &PriceTier{
		AbovePromptTokens: 100_000,
		InputMultiple:     decimal.NewFromInt(5),
		OutputMultiple:    decimal.NewFromInt(5),
	}
	claudeListPrices = map[claudePriceKey]claudeListPrice{
		{ClaudeFamilyFable, 51}:  {input: "10", output: "50", cacheRead: "0.25"},
		{ClaudeFamilyMythos, 51}: {input: "10", output: "50", cacheRead: "0.25"},
		{ClaudeFamilyFable, 50}:  {input: "10", output: "50"},
		{ClaudeFamilyOpus, 55}:   {input: "4", output: "20", cacheRead: "0.20"},
		{ClaudeFamilyOpus, 50}:   {input: "5", output: "25"},
		{ClaudeFamilyOpus, 48}:   {input: "5", output: "25"},
		{ClaudeFamilyOpus, 47}:   {input: "5", output: "25"},
		{ClaudeFamilyOpus, 46}:   {input: "5", output: "25"},
		{ClaudeFamilySonnet, 55}: {input: "2", output: "10", cacheRead: "0.20"},
		{ClaudeFamilySonnet, 50}: {input: "2", output: "10"},
		{ClaudeFamilySonnet, 46}: {input: "3", output: "15"},
		{ClaudeFamilyHaiku, 55}:  {input: "0.10", output: "0.50", longPrompt: haikuLongPrompt},
		{ClaudeFamilyHaiku, 45}:  {input: "1", output: "5"},
	}
)

func ListPrice(kind Kind, model string) (PriceCard, bool) {
	if kind != KindAnthropicMessages {
		return PriceCard{}, false
	}
	parsed, ok := ParseClaudeModel(model)
	if !ok {
		return PriceCard{}, false
	}
	listed, ok := claudeListPrices[claudePriceKey{family: parsed.Family, version: parsed.Version}]
	if !ok {
		return PriceCard{}, false
	}

	card := PriceCard{
		InputCostPerMillion:  decimal.RequireFromString(listed.input),
		OutputCostPerMillion: decimal.RequireFromString(listed.output),
		CacheReadMultiple:    cacheReadMultiple,
		CacheWriteMultiple:   claudeCacheWriteMultiple,
		LongPrompt:           listed.longPrompt,
	}
	if listed.cacheRead != "" {
		card.CacheReadMultiple = decimal.RequireFromString(listed.cacheRead).
			Div(card.InputCostPerMillion)
	}

	return card, true
}

func (u TokenUsage) PromptTokens(kind Kind) int {
	prompt := max(u.Input, 0)
	if kind.CachesOutsideInput() {
		prompt += max(u.CacheRead, 0) + max(u.CacheWrite, 0)
	}

	return prompt
}

func (p *Provider) listPrice() (PriceCard, bool) {
	return ListPrice(p.Kind, p.Model)
}

func (p *Provider) tierFor(usage TokenUsage) (input, output decimal.Decimal) {
	one := decimal.NewFromInt(1)
	card, ok := p.listPrice()
	if !ok || card.LongPrompt == nil {
		return one, one
	}
	if usage.PromptTokens(p.Kind) <= card.LongPrompt.AbovePromptTokens {
		return one, one
	}

	return card.LongPrompt.InputMultiple, card.LongPrompt.OutputMultiple
}
