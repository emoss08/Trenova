package agentextension

import (
	"strconv"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/configspec"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	ExaSearchTypeAuto = "auto"
	ExaSearchTypeFast = "fast"
	ExaSearchTypeDeep = "deep"
)

var exaSearchDepth = searchDepthField{
	fallback: ExaSearchTypeAuto,
	options:  []string{ExaSearchTypeAuto, ExaSearchTypeFast, ExaSearchTypeDeep},
	helpText: "Auto balances speed and quality. Fast answers quickest. Deep reads more of the web and costs more per search.",
	message:  "Search depth must be auto, fast or deep",
}

func exaSpec() Spec {
	return Spec{
		Capability: CapabilityWebResearch,
		Fields: webResearchFields(&webResearchFieldsParams{
			apiKey: APIKeyField(
				"Your Exa API key",
				"Create a key at dashboard.exa.ai. Trenova stores it encrypted and only sends it to api.exa.ai.",
			),
			vendor: []configspec.Field{exaSearchDepth.field(), maxActiveWatchesField()},
		}),
		SupportsTestConnect: true,
		Validate: func(values map[string]string, multiErr *errortypes.MultiError) {
			parseExaSettings(values, multiErr)
		},
	}
}

type ExaSettings struct {
	WebResearchSettings
	SearchType       string
	MaxActiveWatches int
}

func maxActiveWatchesField() configspec.Field {
	return configspec.Field{
		Key:      ConfigKeyMaxActiveWatches,
		Label:    "Web watches at once",
		Type:     configspec.FieldTypeNumber,
		Default:  strconv.Itoa(DefaultMaxActiveWebWatches),
		HelpText: "The most searches agents may keep watching on a schedule at the same time, from 1 to 100. Each check Exa runs costs about $0.015.",
	}
}

func ParseExaSettings(values map[string]string) (ExaSettings, error) {
	multiErr := errortypes.NewMultiError()
	settings := parseExaSettings(values, multiErr)
	if multiErr.HasErrors() {
		return ExaSettings{}, multiErr
	}

	return settings, nil
}

func parseExaSettings(values map[string]string, multiErr *errortypes.MultiError) ExaSettings {
	return ExaSettings{
		WebResearchSettings: parseWebResearchSettings(values, multiErr),
		SearchType:          exaSearchDepth.parse(values, multiErr),
		MaxActiveWatches: parseBoundedInt(boundedIntField{
			values:   values,
			key:      ConfigKeyMaxActiveWatches,
			fallback: DefaultMaxActiveWebWatches,
			min:      MinMaxActiveWebWatches,
			max:      MaxMaxActiveWebWatches,
			message:  "Web watches at once must be a whole number from 1 to 100",
		}, multiErr),
	}
}
