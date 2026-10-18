package modeladapter

import "github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"

// anthropicModel is what a Claude model takes for thinking. The models differ
// sharply: the ones from Opus 4.6 and Sonnet 4.6 on think by effort and refuse
// a token budget, some cannot be told not to think at all, and the newest bind
// every thinking block to the exact conversation that produced it.
type anthropicModel struct {
	// adaptive models think by effort (thinking.type "adaptive" with
	// output_config.effort). A budget_tokens request is a 400 on most of them.
	adaptive bool
	// thinksAlways models refuse thinking.type "disabled" at every effort.
	thinksAlways bool
	// thinksUnasked models think adaptively when a request says nothing
	// about thinking, and leave the summary out unless it is asked for.
	thinksUnasked bool
	// disableRefused models refuse "disabled"; their own off switch takes no
	// other field, so the least thinking is adaptive at low effort.
	disableRefused bool
	// disableNeedsLowEffort models accept "disabled" only at effort high or
	// below, which is every effort this system sends.
	disableNeedsLowEffort bool
	// bindsPrefix models check that a replayed thinking block's system
	// prompt, tools and earlier messages are unchanged since it was written.
	bindsPrefix bool
	// declared models think by effort because the operator said so, behind
	// an id this cannot read. Which effort model it is is unknown, so None
	// asks for the least every one of them accepts.
	declared bool
}

// providerTraits is what the configured provider's model takes: read from its
// id unless the operator said how it thinks.
func providerTraits(provider *aiprovider.Provider) anthropicModel {
	switch provider.ThinkingStyle {
	case aiprovider.ThinkingStyleBudget:
		return anthropicModel{}
	case aiprovider.ThinkingStyleEffort:
		traits := anthropicTraits(provider.Model)
		if traits.adaptive {
			return traits
		}
		return anthropicModel{adaptive: true, declared: true}
	case aiprovider.ThinkingStyleAuto:
		return anthropicTraits(provider.Model)
	default:
		return anthropicTraits(provider.Model)
	}
}

// anthropicTraits reads what a configured model id takes, in any form
// aiprovider.ParseClaudeModel reads. An id it cannot read gets no traits,
// which is the budget behaviour every model took before adaptive thinking, so
// nothing that works today changes.
//
// A version past the newest named here is read as that newest one: the model
// table moves faster than this code, and the newest constraints are the safe
// ones to assume.
func anthropicTraits(model string) anthropicModel {
	claude, ok := aiprovider.ParseClaudeModel(model)
	if !ok {
		return anthropicModel{}
	}
	version := claude.Version

	traits := anthropicModel{adaptive: version >= 46}
	switch claude.Family {
	case aiprovider.ClaudeFamilyOpus:
		traits.thinksAlways = version >= 55
		traits.thinksUnasked = version >= 50
		traits.bindsPrefix = version >= 55
		traits.disableNeedsLowEffort = version == 50
	case aiprovider.ClaudeFamilySonnet:
		traits.disableRefused = version >= 55
		traits.thinksUnasked = version >= 50
		traits.bindsPrefix = version >= 55
	case aiprovider.ClaudeFamilyFable:
		traits.thinksAlways = true
		traits.thinksUnasked = true
		traits.bindsPrefix = version >= 51
	case aiprovider.ClaudeFamilyMythos:
		traits.thinksAlways = true
		traits.thinksUnasked = true
	case aiprovider.ClaudeFamilyHaiku:
		traits.thinksUnasked = version >= 55
		traits.disableNeedsLowEffort = version >= 55
	}

	return traits
}
