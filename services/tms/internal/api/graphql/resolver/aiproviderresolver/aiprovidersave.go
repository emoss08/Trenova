package aiproviderresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type providerCosts struct {
	input      *decimal.Decimal
	output     *decimal.Decimal
	cacheRead  *decimal.Decimal
	cacheWrite *decimal.Decimal
	monthlyCap *decimal.Decimal
}

func saveRequestFromInput(
	id pulid.ID,
	input *gqlmodel.SaveAIProviderInput,
	tenantInfo pagination.TenantInfo,
) (*aiproviderports.SaveAIProviderRequest, error) {
	costs, err := providerCostsFromInput(input)
	if err != nil {
		return nil, err
	}

	return &aiproviderports.SaveAIProviderRequest{
		ID:                       id,
		Name:                     input.Name,
		Description:              input.Description,
		Kind:                     input.Kind,
		BaseURL:                  input.BaseURL,
		Model:                    input.Model,
		APIKey:                   input.APIKey,
		AllowPrivateNetwork:      input.AllowPrivateNetwork,
		StructuredOutputMode:     input.StructuredOutputMode,
		ReasoningEffort:          input.ReasoningEffort,
		ThinkingStyle:            input.ThinkingStyle,
		AdvisorEnabled:           input.AdvisorEnabled,
		AdvisorModel:             input.AdvisorModel,
		Scripts:                  input.Scripts,
		ScriptWrites:             input.ScriptWrites,
		Platform:                 input.Platform,
		PlatformSettings:         platformSettingsFromInput(input.PlatformSettings),
		ExtraBody:                input.ExtraBody,
		InputCostPerMillion:      costs.input,
		OutputCostPerMillion:     costs.output,
		CacheReadCostPerMillion:  costs.cacheRead,
		CacheWriteCostPerMillion: costs.cacheWrite,
		MaxTokens:                input.MaxTokens,
		ContextWindow:            input.ContextWindow,
		Tasks:                    input.Tasks,
		Priority:                 input.Priority,
		EmbeddingDimensions:      input.EmbeddingDimensions,
		EmbeddingInputStyle:      input.EmbeddingInputStyle,
		Trusted:                  input.Trusted,
		Enabled:                  input.Enabled,
		TimeoutSeconds:           input.TimeoutSeconds,
		MaxConcurrent:            input.MaxConcurrent,
		MonthlyCapUSD:            costs.monthlyCap,
		OnCap:                    input.OnCap,
		KeepPreviousKey:          input.KeepPreviousKey,
		Version:                  int64(input.Version),
		TenantInfo:               tenantInfo,
	}, nil
}

type decimalField struct {
	name string
	raw  *string
	dest **decimal.Decimal
}

func providerCostsFromInput(input *gqlmodel.SaveAIProviderInput) (providerCosts, error) {
	costs := providerCosts{}
	fields := []decimalField{
		{"inputCostPerMillion", input.InputCostPerMillion, &costs.input},
		{"outputCostPerMillion", input.OutputCostPerMillion, &costs.output},
		{"cacheReadCostPerMillion", input.CacheReadCostPerMillion, &costs.cacheRead},
		{"cacheWriteCostPerMillion", input.CacheWriteCostPerMillion, &costs.cacheWrite},
		{"monthlyCapUsd", input.MonthlyCapUsd, &costs.monthlyCap},
	}
	for _, field := range fields {
		parsed, err := gqldecimal.ParseNullDecimalField(field.name, field.raw)
		if err != nil {
			return providerCosts{}, err
		}
		*field.dest = gqldecimal.NullDecimalValue(parsed)
	}

	return costs, nil
}
