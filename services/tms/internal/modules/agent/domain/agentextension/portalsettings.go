package agentextension

import (
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/configspec"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	ConfigKeyMaxSteps          = "maxSteps"
	ConfigKeyRecordSessions    = "recordSessions"
	ConfigKeyAllowScreenshots  = "allowScreenshots"
	ConfigKeyBrowserModel      = "browserModel"
	ConfigKeyProxyCountry      = "proxyCountry"
	DefaultPortalMaxSteps      = 30
	MinPortalMaxSteps          = 5
	MaxPortalMaxSteps          = 100
	DefaultPortalDailyTasks    = 50
	defaultRecordSessions      = true
	defaultAllowScreenshots    = false
	portalTaskLimitHelp        = "The most portal tasks all agents in this organization may run in a day (UTC). Agents are told the limit is reached rather than retrying."
	portalMaxStepsHelp         = "The most actions the browser may take on one task, from 5 to 100. A task that needs more stops and says so; most lookups take under 20."
	portalRecordSessionsHelp   = "Keep a video of every portal task so a person can see exactly what the browser did. Turn it off only if the vendor account runs with zero data retention."
	portalAllowScreenshotsHelp = "Let the browser's model see screenshots of the page. It reads the page's text either way; screenshots help with portals drawn as images but can show it values a sign-in field would otherwise hide."
)

type PortalAutomationSettings struct {
	APIKey            string
	MaxSteps          int
	DailyRequestLimit int
	RecordSessions    bool
	AllowScreenshots  bool
}

type portalFieldsParams struct {
	apiKey configspec.Field
	vendor []configspec.Field
}

func portalAutomationFields(params *portalFieldsParams) []configspec.Field {
	fields := make([]configspec.Field, 0, len(params.vendor)+5)
	fields = append(fields, params.apiKey)
	fields = append(fields, params.vendor...)
	return append(fields,
		configspec.Field{
			Key:      ConfigKeyMaxSteps,
			Label:    "Steps per task",
			Type:     configspec.FieldTypeNumber,
			Default:  strconv.Itoa(DefaultPortalMaxSteps),
			HelpText: portalMaxStepsHelp,
		},
		dailyRequestLimitField(portalTaskLimitHelp, DefaultPortalDailyTasks),
		configspec.Field{
			Key:      ConfigKeyRecordSessions,
			Label:    "Record every task",
			Type:     configspec.FieldTypeBoolean,
			Default:  strconv.FormatBool(defaultRecordSessions),
			HelpText: portalRecordSessionsHelp,
		},
		configspec.Field{
			Key:      ConfigKeyAllowScreenshots,
			Label:    "Let the browser see screenshots",
			Type:     configspec.FieldTypeBoolean,
			Default:  strconv.FormatBool(defaultAllowScreenshots),
			HelpText: portalAllowScreenshotsHelp,
		},
	)
}

func ParsePortalAutomationSettings(values map[string]string) (PortalAutomationSettings, error) {
	multiErr := errortypes.NewMultiError()
	settings := parsePortalAutomationSettings(values, multiErr)
	if multiErr.HasErrors() {
		return PortalAutomationSettings{}, multiErr
	}

	return settings, nil
}

func parsePortalAutomationSettings(
	values map[string]string,
	multiErr *errortypes.MultiError,
) PortalAutomationSettings {
	return PortalAutomationSettings{
		APIKey: strings.TrimSpace(values[ConfigKeyAPIKey]),
		MaxSteps: parseBoundedInt(boundedIntField{
			values:   values,
			key:      ConfigKeyMaxSteps,
			fallback: DefaultPortalMaxSteps,
			min:      MinPortalMaxSteps,
			max:      MaxPortalMaxSteps,
			message:  "Steps per task must be a whole number from 5 to 100",
		}, multiErr),
		DailyRequestLimit: dailyRequestLimitFrom(values, DefaultPortalDailyTasks, multiErr),
		RecordSessions: parseFlag(&flagField{
			values:   values,
			key:      ConfigKeyRecordSessions,
			fallback: defaultRecordSessions,
			message:  "Record every task must be true or false",
		}, multiErr),
		AllowScreenshots: parseFlag(&flagField{
			values:   values,
			key:      ConfigKeyAllowScreenshots,
			fallback: defaultAllowScreenshots,
			message:  "Let the browser see screenshots must be true or false",
		}, multiErr),
	}
}

type flagField struct {
	values   map[string]string
	key      string
	fallback bool
	message  string
}

func parseFlag(field *flagField, multiErr *errortypes.MultiError) bool {
	raw := strings.TrimSpace(field.values[field.key])
	if raw == "" {
		return field.fallback
	}

	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		multiErr.Add("configuration."+field.key, errortypes.ErrInvalid, field.message)
		return field.fallback
	}

	return parsed
}
