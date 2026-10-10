package modeladapter

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
)

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

var (
	// claudeID is family, major and an optional minor, as in claude-opus-5-5
	// or claude-haiku-4-5. The older claude-3-7-sonnet shape does not match
	// and keeps the budget behaviour every model of that age takes.
	claudeID = regexp.MustCompile(`^claude-([a-z]+)-(\d+)(?:-(\d))?$`)
	// dateSuffix is a snapshot date: claude-haiku-4-5-20251001.
	dateSuffix = regexp.MustCompile(`-\d{8}$`)
	// bedrockSuffix is Bedrock's model version: ...-v1:0.
	bedrockSuffix = regexp.MustCompile(`-v\d+(?::\d+)?$`)
)

// anthropicTraits reads what a configured model id takes. The id comes in the
// form of whatever endpoint serves it: the Claude API's plain id, Bedrock's
// anthropic.-prefixed one (with a region in front for an inference profile),
// Vertex's @date one, a dated snapshot, or an alias a gateway made up. An id
// this cannot read gets no traits, which is the budget behaviour every model
// took before adaptive thinking, so nothing that works today changes.
//
// A version past the newest named here is read as that newest one: the model
// table moves faster than this code, and the newest constraints are the safe
// ones to assume.
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

func anthropicTraits(model string) anthropicModel {
	id := strings.ToLower(strings.TrimSpace(model))
	if at := strings.IndexByte(id, '@'); at >= 0 {
		id = id[:at]
	}
	start := strings.Index(id, "claude-")
	if start < 0 {
		return anthropicModel{}
	}
	id = bedrockSuffix.ReplaceAllString(id[start:], "")
	id = dateSuffix.ReplaceAllString(id, "")

	match := claudeID.FindStringSubmatch(id)
	if match == nil {
		return anthropicModel{}
	}
	major, err := strconv.Atoi(match[2])
	if err != nil {
		return anthropicModel{}
	}
	minor := 0
	if match[3] != "" {
		minor, _ = strconv.Atoi(match[3])
	}
	version := major*10 + minor

	traits := anthropicModel{adaptive: version >= 46}
	switch match[1] {
	case "opus":
		traits.thinksAlways = version >= 55
		traits.bindsPrefix = version >= 55
		traits.disableNeedsLowEffort = version == 50
	case "sonnet":
		traits.disableRefused = version >= 55
		traits.bindsPrefix = version >= 55
	case "fable":
		traits.thinksAlways = true
		traits.bindsPrefix = version >= 51
	case "mythos":
		traits.thinksAlways = true
	case "haiku":
		traits.disableNeedsLowEffort = version >= 55
	}

	return traits
}
