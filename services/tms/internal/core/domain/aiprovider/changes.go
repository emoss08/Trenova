package aiprovider

import (
	"reflect"

	"github.com/emoss08/trenova/internal/core/domain/editchange"
	"github.com/emoss08/trenova/shared/decimalutils"
	"github.com/emoss08/trenova/shared/setutils"
	"github.com/emoss08/trenova/shared/typeutils"
)

// ChangeRules name, in the provider editor's words, the settings a person
// edits there. The API key itself is never compared, since it is not kept in
// a version; its prefix, last four and when it was added are.
var ChangeRules = []editchange.Rule[Provider]{
	{
		Field: "name", Label: "Name",
		Same: func(a, b *Provider) bool { return a.Name == b.Name && a.Description == b.Description },
	},
	{
		Field: "baseUrl", Label: "Connection",
		Same: func(a, b *Provider) bool { return a.Kind == b.Kind && a.BaseURL == b.BaseURL },
	},
	{
		Field: "model", Label: "Model",
		Same: func(a, b *Provider) bool {
			return a.Model == b.Model &&
				a.StructuredOutputMode == b.StructuredOutputMode &&
				a.MaxTokens == b.MaxTokens &&
				typeutils.EqualPtr(a.ContextWindowTokens, b.ContextWindowTokens) &&
				a.ReasoningEffort == b.ReasoningEffort &&
				a.ThinkingStyle == b.ThinkingStyle &&
				sameExtraBody(a.ExtraBody, b.ExtraBody)
		},
	},
	{
		Field: "embeddingDimensions", Label: "Embeddings",
		Same: func(a, b *Provider) bool {
			return typeutils.EqualPtr(a.EmbeddingDimensions, b.EmbeddingDimensions) &&
				a.EmbeddingInputStyle == b.EmbeddingInputStyle
		},
	},
	{
		Field: "tasks", Label: "What it handles",
		Same: func(a, b *Provider) bool { return setutils.SameMembers(a.Tasks, b.Tasks) },
	},
	{
		Field: "priority", Label: "Order",
		Same: func(a, b *Provider) bool { return a.Priority == b.Priority },
	},
	{
		Field: "trusted", Label: "Access",
		Same: func(a, b *Provider) bool {
			return a.Trusted == b.Trusted && a.AllowPrivateNetwork == b.AllowPrivateNetwork
		},
	},
	{
		Field: "inputCostPerMillion", Label: "Price",
		Same: func(a, b *Provider) bool {
			return decimalutils.PtrEqual(a.InputCostPerMillion, b.InputCostPerMillion) &&
				decimalutils.PtrEqual(a.OutputCostPerMillion, b.OutputCostPerMillion)
		},
	},
	{
		Field: "timeoutSeconds", Label: "Limits",
		Same: func(a, b *Provider) bool {
			return a.TimeoutSeconds == b.TimeoutSeconds &&
				a.MaxConcurrent == b.MaxConcurrent &&
				decimalutils.PtrEqual(a.MonthlyCapUSD, b.MonthlyCapUSD) &&
				a.OnCap == b.OnCap
		},
	},
	{
		Field: "apiKey", Label: "API key",
		Same: func(a, b *Provider) bool {
			return a.APIKeyPrefix == b.APIKeyPrefix &&
				a.APIKeyLastFour == b.APIKeyLastFour &&
				typeutils.EqualPtr(a.APIKeyAddedAt, b.APIKeyAddedAt)
		},
	},
	{
		Field: "enabled", Label: "Enabled",
		Same: func(a, b *Provider) bool { return a.Enabled == b.Enabled },
	},
}

func sameExtraBody(a, b map[string]any) bool {
	if len(a) == 0 || len(b) == 0 {
		return len(a) == len(b)
	}
	return reflect.DeepEqual(a, b)
}
