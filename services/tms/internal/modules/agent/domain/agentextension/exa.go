package agentextension

import (
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
			vendor: []configspec.Field{exaSearchDepth.field()},
		}),
		SupportsTestConnect: true,
		Validate: func(values map[string]string, multiErr *errortypes.MultiError) {
			parseExaSettings(values, multiErr)
		},
	}
}

type ExaSettings struct {
	WebResearchSettings
	SearchType string
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
	}
}
