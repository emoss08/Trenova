package agentextension

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/configspec"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	ConfigKeyAPIKey            = "apiKey"
	ConfigKeyDailyRequestLimit = "dailyRequestLimit"

	DefaultDailyRequestLimit = 200
	MinDailyRequestLimit     = 1
	MaxDailyRequestLimit     = 10000
)

var hostnamePattern = regexp.MustCompile(
	`^(\*\.)?([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$`,
)

type SettingsValidator func(values map[string]string, multiErr *errortypes.MultiError)

type Spec struct {
	Capability          Capability
	Fields              []configspec.Field
	SupportsTestConnect bool
	Validate            SettingsValidator
	DefaultDailyLimit   int
}

func (s Spec) DailyLimit(values map[string]string, multiErr *errortypes.MultiError) int {
	return dailyRequestLimitFrom(values, s.dailyLimitFallback(), multiErr)
}

func (s Spec) dailyLimitFallback() int {
	if s.DefaultDailyLimit > 0 {
		return s.DefaultDailyLimit
	}

	return DefaultDailyRequestLimit
}

func (s Spec) Configured(configuration map[string]any) bool {
	return configspec.HasRequired(configuration, s.Fields)
}

func (s Spec) Tools() []string {
	return capabilities[s.Capability].Tools
}

func (s Spec) HasTool(name string) bool {
	return capabilities[s.Capability].HasTool(name)
}

var specs = map[Type]Spec{
	TypeExa:        exaSpec(),
	TypeParallel:   parallelSpec(),
	TypeBrowserUse: browserUseSpec(),
}

func SpecFor(t Type) (Spec, bool) {
	spec, ok := specs[t]
	return spec, ok
}

func TypesFor(c Capability) []Type {
	types := make([]Type, 0, 1)
	for _, typ := range AllTypes() {
		if specs[typ].Capability == c {
			types = append(types, typ)
		}
	}

	return types
}

func ValidateSettings(t Type, values map[string]string, multiErr *errortypes.MultiError) {
	spec, ok := specs[t]
	if !ok || spec.Validate == nil {
		DailyRequestLimit(values, multiErr)
		return
	}

	spec.Validate(values, multiErr)
}

func APIKeyField(placeholder, helpText string) configspec.Field {
	return configspec.Field{
		Key:         ConfigKeyAPIKey,
		Label:       "API key",
		Type:        configspec.FieldTypePassword,
		Required:    true,
		Sensitive:   true,
		Placeholder: placeholder,
		HelpText:    helpText,
	}
}

func DailyRequestLimitField(helpText string) configspec.Field {
	return dailyRequestLimitField(helpText, DefaultDailyRequestLimit)
}

func dailyRequestLimitField(helpText string, fallback int) configspec.Field {
	return configspec.Field{
		Key:      ConfigKeyDailyRequestLimit,
		Label:    "Daily request limit",
		Type:     configspec.FieldTypeNumber,
		Default:  strconv.Itoa(fallback),
		HelpText: helpText,
	}
}

func DailyRequestLimit(values map[string]string, multiErr *errortypes.MultiError) int {
	return dailyRequestLimitFrom(values, DefaultDailyRequestLimit, multiErr)
}

func dailyRequestLimitFrom(
	values map[string]string,
	fallback int,
	multiErr *errortypes.MultiError,
) int {
	return parseBoundedInt(boundedIntField{
		values:   values,
		key:      ConfigKeyDailyRequestLimit,
		fallback: fallback,
		min:      MinDailyRequestLimit,
		max:      MaxDailyRequestLimit,
		message:  "Daily request limit must be a whole number from 1 to 10,000",
	}, multiErr)
}

type boundedIntField struct {
	values   map[string]string
	key      string
	fallback int
	min      int
	max      int
	message  string
}

func parseBoundedInt(field boundedIntField, multiErr *errortypes.MultiError) int {
	raw := strings.TrimSpace(field.values[field.key])
	if raw == "" {
		return field.fallback
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < field.min || parsed > field.max {
		multiErr.Add("configuration."+field.key, errortypes.ErrInvalid, field.message)
		return field.fallback
	}

	return parsed
}

type choiceField struct {
	values   map[string]string
	key      string
	fallback string
	options  []string
	message  string
}

func parseChoice(field *choiceField, multiErr *errortypes.MultiError) string {
	value := strings.TrimSpace(field.values[field.key])
	if value == "" {
		return field.fallback
	}
	for _, option := range field.options {
		if value == option {
			return value
		}
	}

	multiErr.Add("configuration."+field.key, errortypes.ErrInvalid, field.message)

	return field.fallback
}
