package aiproviderchange

import (
	"reflect"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/editchange"
	"github.com/emoss08/trenova/shared/decimalutils"
	"github.com/emoss08/trenova/shared/httpsafe"
	"github.com/emoss08/trenova/shared/setutils"
	"github.com/emoss08/trenova/shared/typeutils"
)

// ChangeRules name, in the provider editor's words, the settings a person
// edits there. The API key itself is never compared, since it is not kept in
// a version; its prefix, last four and when it was added are.
var ChangeRules = []editchange.Rule[aiprovider.Provider]{
	{
		Field: "name",
		Label: "Name",
		Same:  func(a, b *aiprovider.Provider) bool { return a.Name == b.Name && a.Description == b.Description },
	},
	{
		Field: "baseUrl", Label: "Connection",
		Same: func(a, b *aiprovider.Provider) bool {
			return a.Kind == b.Kind && a.BaseURL == b.BaseURL &&
				a.ResolvedPlatform() == b.ResolvedPlatform() &&
				a.ResolvedSettings() == b.ResolvedSettings()
		},
	},
	{
		Field: "model", Label: "Model",
		Same: func(a, b *aiprovider.Provider) bool {
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
		Field: "advisorEnabled", Label: "Advisor",
		Same: func(a, b *aiprovider.Provider) bool {
			return a.AdvisorEnabled == b.AdvisorEnabled &&
				(!a.AdvisorEnabled || a.AdvisorModelName() == b.AdvisorModelName())
		},
	},
	{
		Field: "scripts",
		Label: "Scripts",
		Same:  func(a, b *aiprovider.Provider) bool { return a.ScriptSettings() == b.ScriptSettings() },
	},
	{
		Field: "embeddingDimensions", Label: "Embeddings",
		Same: func(a, b *aiprovider.Provider) bool {
			return typeutils.EqualPtr(a.EmbeddingDimensions, b.EmbeddingDimensions) &&
				a.EmbeddingInputStyle == b.EmbeddingInputStyle
		},
	},
	{
		Field: "tasks",
		Label: "What it handles",
		Same:  func(a, b *aiprovider.Provider) bool { return setutils.SameMembers(a.Tasks, b.Tasks) },
	},
	{
		Field: "priority", Label: "Order",
		Same: func(a, b *aiprovider.Provider) bool { return a.Priority == b.Priority },
	},
	{
		Field: "trusted", Label: "Access",
		Same: func(a, b *aiprovider.Provider) bool {
			return a.Trusted == b.Trusted && a.AllowPrivateNetwork == b.AllowPrivateNetwork
		},
	},
	{
		Field: "inputCostPerMillion", Label: "Price",
		Same: func(a, b *aiprovider.Provider) bool {
			return decimalutils.PtrEqual(a.InputCostPerMillion, b.InputCostPerMillion) &&
				decimalutils.PtrEqual(a.OutputCostPerMillion, b.OutputCostPerMillion) &&
				decimalutils.PtrEqual(a.CacheReadCostPerMillion, b.CacheReadCostPerMillion) &&
				decimalutils.PtrEqual(a.CacheWriteCostPerMillion, b.CacheWriteCostPerMillion)
		},
	},
	{
		Field: "timeoutSeconds", Label: "Limits",
		Same: func(a, b *aiprovider.Provider) bool {
			return a.TimeoutSeconds == b.TimeoutSeconds &&
				a.MaxConcurrent == b.MaxConcurrent &&
				decimalutils.PtrEqual(a.MonthlyCapUSD, b.MonthlyCapUSD) &&
				a.OnCap == b.OnCap
		},
	},
	{
		Field: "apiKey", Label: "API key",
		Same: func(a, b *aiprovider.Provider) bool {
			return a.APIKeyPrefix == b.APIKeyPrefix &&
				a.APIKeyLastFour == b.APIKeyLastFour &&
				typeutils.EqualPtr(a.APIKeyAddedAt, b.APIKeyAddedAt)
		},
	},
	{
		Field: "enabled", Label: "Enabled",
		Same: func(a, b *aiprovider.Provider) bool { return a.Enabled == b.Enabled },
	},
}

func sameExtraBody(a, b map[string]any) bool {
	if len(a) == 0 || len(b) == 0 {
		return len(a) == len(b)
	}
	return reflect.DeepEqual(a, b)
}

// AdvisorSettingsChanged reports a change that makes an earlier refusal stale:
// the advisor turned on, another advisor, or another model it advises.
func AdvisorSettingsChanged(before, after *aiprovider.Provider) bool {
	return before.AdvisorEnabled != after.AdvisorEnabled ||
		!strings.EqualFold(before.AdvisorModelName(), after.AdvisorModelName()) ||
		!strings.EqualFold(before.Model, after.Model) ||
		before.Kind != after.Kind ||
		before.ResolvedPlatform() != after.ResolvedPlatform()
}

// SameCredentialTarget reports whether two configurations send a stored
// credential to the same place, as the same identity. A credential entered
// for one is never carried to the other: kept across a change of platform,
// region, project, workspace, tenant or address, it would be sent, decrypted,
// somewhere it was never meant for.
func SameCredentialTarget(a, b *aiprovider.Provider) bool {
	if a.Kind != b.Kind || a.ResolvedPlatform() != b.ResolvedPlatform() {
		return false
	}
	if a.ResolvedSettings() != b.ResolvedSettings() {
		return false
	}

	return httpsafe.SameOrigin(a.ResolvedBaseURL(), b.ResolvedBaseURL())
}
