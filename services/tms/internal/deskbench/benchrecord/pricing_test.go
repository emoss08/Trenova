package benchrecord

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aipricing"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeProviders struct {
	aiproviderports.AIProviderRepository
	provider *aiprovider.Provider
}

func (f *fakeProviders) GetByID(
	context.Context,
	aiproviderports.GetAIProviderByIDRequest,
) (*aiprovider.Provider, error) {
	return f.provider, nil
}

type fakeReference struct{}

func (fakeReference) Prices(
	_ context.Context,
	_ aiprovider.Kind,
	ids []string,
) map[string]aiproviderports.ReferencePrice {
	output := decimal.RequireFromString("8")
	read := decimal.RequireFromString("0.1")

	return map[string]aiproviderports.ReferencePrice{ids[0]: {
		InputCostPerMillion:     decimal.RequireFromString("1"),
		OutputCostPerMillion:    &output,
		CacheReadCostPerMillion: &read,
	}}
}

func TestPricer_PricesAnUnpricedProviderFromTheListThenTheReference(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	haiku := &aiprovider.Provider{
		ID: pulid.MustNew(
			"aiprv_",
		), Kind: aiprovider.KindAnthropicMessages, Model: "claude-haiku-5-5",
	}
	listed := NewPricer(&fakeProviders{provider: haiku}, fakeReference{})
	price := listed.Price(t.Context(), tenant, &tokenCount{
		provider: haiku.ID, model: "claude-haiku-5-5",
		raw: aipricing.TokenUsage{Input: 1_000_000, CacheRead: 1_000_000, Output: 1_000_000},
	})
	require.NotNil(t, price.cost)
	assert.Equal(t, PriceSourceListPrice, price.source)
	assert.True(t, price.cost.Equal(decimal.RequireFromString("3.05")), price.cost.String(),
		"above 100K prompt tokens Haiku 5.5 is priced five times over")

	luna := &aiprovider.Provider{
		ID: pulid.MustNew("aiprv_"), Kind: aiprovider.KindOpenAIResponses, Model: "gpt-6-luna",
	}
	referenced := NewPricer(&fakeProviders{provider: luna}, fakeReference{})
	price = referenced.Price(t.Context(), tenant, &tokenCount{
		provider: luna.ID, model: "gpt-6-luna",
		raw: aipricing.TokenUsage{Input: 1_000_000, CacheRead: 500_000, Output: 100_000},
	})
	require.NotNil(t, price.cost)
	assert.Equal(t, PriceSourceReference, price.source)
	assert.True(t, price.cost.Equal(decimal.RequireFromString("1.35")), price.cost.String())

	charged := decimal.RequireFromString("0.5")
	price = referenced.Price(t.Context(), tenant, &tokenCount{
		provider: luna.ID, model: "gpt-6-luna", charged: &charged,
	})
	assert.Equal(t, PriceSourceProvider, price.source)
}

func TestPricer_PricesEachModelAProviderServedOnItsOwn(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	gateway := &aiprovider.Provider{
		ID: pulid.MustNew(
			"aiprv_",
		), Kind: aiprovider.KindAnthropicMessages, Model: "claude-haiku-5-5",
	}
	pricer := NewPricer(&fakeProviders{provider: gateway}, fakeReference{})
	usage := aipricing.TokenUsage{Input: 1_000, Output: 1_000}

	haiku := pricer.Price(t.Context(), tenant, &tokenCount{
		provider: gateway.ID, model: "claude-haiku-5-5", raw: usage,
	})
	opus := pricer.Price(t.Context(), tenant, &tokenCount{
		provider: gateway.ID, model: "claude-opus-5-5", raw: usage,
	})
	require.NotNil(t, haiku.cost)
	require.NotNil(t, opus.cost)
	assert.True(t, opus.cost.GreaterThan(*haiku.cost),
		"a failover to another model on the same provider is priced at that model's rate")
}

func TestPricer_PricesTheAdvisorsTokensAtTheAdvisorsRate(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	haiku := &aiprovider.Provider{
		ID: pulid.MustNew(
			"aiprv_",
		), Kind: aiprovider.KindAnthropicMessages, Model: "claude-haiku-5-5",
	}
	pricer := NewPricer(&fakeProviders{provider: haiku}, fakeReference{})
	tokens := tokenCount{
		provider: haiku.ID, model: "claude-haiku-5-5",
		raw: aipricing.TokenUsage{Input: 10_000, Output: 10_000},
		advisor: &aipricing.AdvisorUsage{
			Model:  "claude-opus-5-5",
			Calls:  1,
			Tokens: aipricing.TokenUsage{Input: 1_000_000, Output: 100_000},
		},
	}

	price := pricer.Price(t.Context(), tenant, &tokens)
	require.NotNil(t, price.cost)
	assert.True(t, price.cost.Equal(decimal.RequireFromString("6.006")), price.cost.String(),
		"Haiku 5.5 at $0.10/$0.50 plus Opus 5.5 at $4/$20")

	tokens.advisor.Model = "house-advisor"
	unpriced := pricer.Price(t.Context(), tenant, &tokens)
	assert.Nil(t, unpriced.cost, "an advisor without a price leaves the call unpriced")

	charged := decimal.RequireFromString("1.25")
	tokens.charged = &charged
	assert.True(t, pricer.Price(t.Context(), tenant, &tokens).cost.Equal(charged),
		"the router's charge already holds the advisor")
}

func TestPricer_PricesEachExecutorIterationOnItsOwnTier(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	haiku := &aiprovider.Provider{
		ID: pulid.MustNew(
			"aiprv_",
		), Kind: aiprovider.KindAnthropicMessages, Model: "claude-haiku-5-5",
	}
	pricer := NewPricer(&fakeProviders{provider: haiku}, fakeReference{})
	price := pricer.Price(t.Context(), tenant, &tokenCount{
		provider: haiku.ID, model: "claude-haiku-5-5",
		raw: aipricing.TokenUsage{Input: 120_000, Output: 2_000},
		advisor: &aipricing.AdvisorUsage{
			Model:  "claude-opus-5-5",
			Calls:  1,
			Tokens: aipricing.TokenUsage{Input: 10_000, Output: 1_000},
			Executor: []aipricing.TokenUsage{
				{Input: 60_000, Output: 1_000},
				{Input: 60_000, Output: 1_000},
			},
		},
	})

	require.NotNil(t, price.cost)
	assert.True(
		t,
		price.cost.Equal(decimal.RequireFromString("0.073")),
		price.cost.String(),
		"each 60K-prompt iteration at Haiku 5.5's short rates, not the summed 120K at the long ones",
	)
}
