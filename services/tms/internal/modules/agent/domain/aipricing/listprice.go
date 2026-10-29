package aipricing

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/aimodel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
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
	family  aimodel.ClaudeFamily
	version int
}

type claudeListPrice struct {
	input, output, cacheRead string
	longPrompt               *PriceTier
}

var (
	haikuLongPrompt = &PriceTier{
		AbovePromptTokens: 100_000,
		InputMultiple:     decimal.NewFromInt(5),
		OutputMultiple:    decimal.NewFromInt(5),
	}
	claudeListPrices = map[claudePriceKey]claudeListPrice{
		{aimodel.ClaudeFamilyFable, 51}:  {input: "10", output: "50", cacheRead: "0.25"},
		{aimodel.ClaudeFamilyMythos, 51}: {input: "10", output: "50", cacheRead: "0.25"},
		{aimodel.ClaudeFamilyFable, 50}:  {input: "10", output: "50"},
		{aimodel.ClaudeFamilyOpus, 55}:   {input: "4", output: "20", cacheRead: "0.20"},
		{aimodel.ClaudeFamilyOpus, 50}:   {input: "5", output: "25"},
		{aimodel.ClaudeFamilyOpus, 48}:   {input: "5", output: "25"},
		{aimodel.ClaudeFamilyOpus, 47}:   {input: "5", output: "25"},
		{aimodel.ClaudeFamilyOpus, 46}:   {input: "5", output: "25"},
		{aimodel.ClaudeFamilySonnet, 55}: {input: "2", output: "10", cacheRead: "0.20"},
		{aimodel.ClaudeFamilySonnet, 50}: {input: "2", output: "10"},
		{aimodel.ClaudeFamilySonnet, 46}: {input: "3", output: "15"},
		{aimodel.ClaudeFamilyHaiku, 55}: {
			input:      "0.10",
			output:     "0.50",
			longPrompt: haikuLongPrompt,
		},
		{aimodel.ClaudeFamilyHaiku, 45}: {input: "1", output: "5"},
	}
)

func ListPrice(kind aiprovider.Kind, model string) (PriceCard, bool) {
	if kind != aiprovider.KindAnthropicMessages {
		return PriceCard{}, false
	}
	parsed, ok := aimodel.ParseClaudeModel(model)
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
		CacheReadMultiple:    kind.CacheReadMultiple(),
		CacheWriteMultiple:   kind.CacheWriteMultiple(),
		LongPrompt:           listed.longPrompt,
	}
	if listed.cacheRead != "" {
		card.CacheReadMultiple = decimal.RequireFromString(listed.cacheRead).
			Div(card.InputCostPerMillion)
	}

	return card, true
}
