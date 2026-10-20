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
	ConfigKeyCodeMode        = "codeMode"
	ConfigKeyScriptSeconds   = "scriptSeconds"
	ConfigKeyScriptReads     = "scriptReads"

	DefaultE2BTemplate        = "code-interpreter-v1"
	DefaultAnalysisRunSeconds = 90
	MinAnalysisRunSeconds     = 15
	MaxAnalysisRunSeconds     = 180
	DefaultDailyAnalyses      = 100

	DefaultScriptSeconds = 60
	MinScriptSeconds     = 15
	MaxScriptSeconds     = 180
	DefaultScriptReads   = 60
	MinScriptReads       = 1
	MaxScriptReads       = 200

	analysisLimitHelp = "The most analyses all agents in this organization may run in a day (UTC). Agents are told the limit is reached rather than retrying."
	runSecondsHelp    = "How long one analysis may run, from 15 to 180 seconds. Code still running then is stopped and the agent is told to narrow it."
	templateHelp      = "The E2B sandbox template analyses run in. Leave it as code-interpreter-v1, which has Python with pandas, numpy and matplotlib, unless you built your own from it."
	codeModeHelp      = "Let agents answer questions that need many reads with one short read-only script. Every read the script makes is the agent's own read, checked as the person; the rows it reads are sent to E2B while it runs. Off until an administrator turns it on."
	scriptSecondsHelp = "How long one script may run, from 15 to 180 seconds."
	scriptReadsHelp   = "How many reads one script may make, from 1 to 200."
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
			{
				Key:      ConfigKeyCodeMode,
				Label:    "Read-only scripts",
				Type:     configspec.FieldTypeBoolean,
				Default:  "false",
				HelpText: codeModeHelp,
			},
			{
				Key:      ConfigKeyScriptSeconds,
				Label:    "Seconds per script",
				Type:     configspec.FieldTypeNumber,
				Default:  strconv.Itoa(DefaultScriptSeconds),
				HelpText: scriptSecondsHelp,
			},
			{
				Key:      ConfigKeyScriptReads,
				Label:    "Reads per script",
				Type:     configspec.FieldTypeNumber,
				Default:  strconv.Itoa(DefaultScriptReads),
				HelpText: scriptReadsHelp,
			},
		},
		Switched:            map[Capability]string{CapabilityCodeMode: ConfigKeyCodeMode},
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
	CodeMode          bool
	ScriptSeconds     int
	ScriptReads       int
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
		CodeMode:          parseSwitch(values, ConfigKeyCodeMode, multiErr),
		ScriptSeconds: parseBoundedInt(boundedIntField{
			values:   values,
			key:      ConfigKeyScriptSeconds,
			fallback: DefaultScriptSeconds,
			min:      MinScriptSeconds,
			max:      MaxScriptSeconds,
			message:  "Seconds per script must be a whole number from 15 to 180",
		}, multiErr),
		ScriptReads: parseBoundedInt(boundedIntField{
			values:   values,
			key:      ConfigKeyScriptReads,
			fallback: DefaultScriptReads,
			min:      MinScriptReads,
			max:      MaxScriptReads,
			message:  "Reads per script must be a whole number from 1 to 200",
		}, multiErr),
	}
}

func parseSwitch(values map[string]string, key string, multiErr *errortypes.MultiError) bool {
	switch strings.TrimSpace(values[key]) {
	case "", "false":
		return false
	case "true":
		return true
	default:
		multiErr.Add("configuration."+key, errortypes.ErrInvalid, "Must be true or false")
		return false
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
