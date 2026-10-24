package agentextension

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/configspec"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	ConfigKeyRunSeconds      = "runSeconds"
	ConfigKeySandboxTemplate = "sandboxTemplate"

	DefaultE2BTemplate        = "code-interpreter-v1"
	DefaultAnalysisRunSeconds = 90
	MinAnalysisRunSeconds     = 15
	MaxAnalysisRunSeconds     = 180
	DefaultDailyAnalyses      = 100

	analysisLimitHelp = "The most analyses all agents in this organization may run in a day (UTC). Agents are told the limit is reached rather than retrying."
	runSecondsHelp    = "How long one analysis may run, from 15 to 180 seconds. Code still running then is stopped and the agent is told to narrow it."
	templateHelp      = "The E2B sandbox template analyses run in. Leave it as code-interpreter-v1, which has Python with pandas, numpy and matplotlib, unless you built your own from it."
)

var sandboxTemplatePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,62}(:[a-z0-9._-]{1,64})?$`)

func e2bSpec() Spec {
	return Spec{
		Capability: CapabilityDataAnalysis,
		Fields: []configspec.Field{
			APIKeyField(
				"Your E2B API key",
				"Create a key at e2b.dev/dashboard. Trenova stores it encrypted and only sends it to E2B.",
			),
			{
				Key:      ConfigKeyRunSeconds,
				Label:    "Seconds per analysis",
				Type:     configspec.FieldTypeNumber,
				Default:  strconv.Itoa(DefaultAnalysisRunSeconds),
				HelpText: runSecondsHelp,
			},
			dailyRequestLimitField(analysisLimitHelp, DefaultDailyAnalyses),
			{
				Key:      ConfigKeySandboxTemplate,
				Label:    "Sandbox template",
				Type:     configspec.FieldTypeString,
				Default:  DefaultE2BTemplate,
				HelpText: templateHelp,
			},
		},
		SupportsTestConnect: true,
		DefaultDailyLimit:   DefaultDailyAnalyses,
		Validate: func(values map[string]string, multiErr *errortypes.MultiError) {
			parseE2BSettings(values, multiErr)
		},
	}
}

type E2BSettings struct {
	APIKey            string
	RunSeconds        int
	DailyRequestLimit int
	Template          string
}

func ParseE2BSettings(values map[string]string) (E2BSettings, error) {
	multiErr := errortypes.NewMultiError()
	settings := parseE2BSettings(values, multiErr)
	if multiErr.HasErrors() {
		return E2BSettings{}, multiErr
	}

	return settings, nil
}

func parseE2BSettings(values map[string]string, multiErr *errortypes.MultiError) E2BSettings {
	return E2BSettings{
		APIKey: strings.TrimSpace(values[ConfigKeyAPIKey]),
		RunSeconds: parseBoundedInt(boundedIntField{
			values:   values,
			key:      ConfigKeyRunSeconds,
			fallback: DefaultAnalysisRunSeconds,
			min:      MinAnalysisRunSeconds,
			max:      MaxAnalysisRunSeconds,
			message:  "Seconds per analysis must be a whole number from 15 to 180",
		}, multiErr),
		DailyRequestLimit: dailyRequestLimitFrom(values, DefaultDailyAnalyses, multiErr),
		Template:          parseSandboxTemplate(values, multiErr),
	}
}

func parseSandboxTemplate(values map[string]string, multiErr *errortypes.MultiError) string {
	template := strings.TrimSpace(values[ConfigKeySandboxTemplate])
	if template == "" {
		return DefaultE2BTemplate
	}
	if !sandboxTemplatePattern.MatchString(template) {
		multiErr.Add("configuration."+ConfigKeySandboxTemplate, errortypes.ErrInvalid,
			"Sandbox template must be an E2B template name such as code-interpreter-v1")
		return DefaultE2BTemplate
	}

	return template
}
