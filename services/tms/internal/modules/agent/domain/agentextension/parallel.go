package agentextension

import (
	"github.com/emoss08/trenova/internal/modules/kernel/domain/configspec"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	ParallelModeFast     = "fast"
	ParallelModeBasic    = "basic"
	ParallelModeAdvanced = "advanced"
)

var parallelSearchDepth = searchDepthField{
	fallback: ParallelModeBasic,
	options:  []string{ParallelModeFast, ParallelModeBasic, ParallelModeAdvanced},
	helpText: "Fast answers quickest and costs least. Basic balances speed and quality. Advanced ranks more carefully and costs the same as basic.",
	message:  "Search depth must be fast, basic or advanced",
}

func parallelSpec() Spec {
	return Spec{
		Capability: CapabilityWebResearch,
		Fields: webResearchFields(&webResearchFieldsParams{
			apiKey: APIKeyField(
				"Your Parallel API key",
				"Create a key at platform.parallel.ai. Trenova stores it encrypted and only sends it to api.parallel.ai.",
			),
			vendor: []configspec.Field{parallelSearchDepth.field()},
		}),
		SupportsTestConnect: true,
		Validate: func(values map[string]string, multiErr *errortypes.MultiError) {
			parseParallelSettings(values, multiErr)
		},
	}
}

type ParallelSettings struct {
	WebResearchSettings
	Mode string
}

func ParseParallelSettings(values map[string]string) (ParallelSettings, error) {
	multiErr := errortypes.NewMultiError()
	settings := parseParallelSettings(values, multiErr)
	if multiErr.HasErrors() {
		return ParallelSettings{}, multiErr
	}

	return settings, nil
}

func parseParallelSettings(
	values map[string]string,
	multiErr *errortypes.MultiError,
) ParallelSettings {
	return ParallelSettings{
		WebResearchSettings: parseWebResearchSettings(values, multiErr),
		Mode:                parallelSearchDepth.parse(values, multiErr),
	}
}
