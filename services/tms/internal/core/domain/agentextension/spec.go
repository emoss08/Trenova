package agentextension

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/configspec"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	ToolWebSearch = "web_search"
	ToolWebRead   = "web_read"

	ConfigKeyAPIKey            = "apiKey"
	ConfigKeyDailyRequestLimit = "dailyRequestLimit"
	ConfigKeySearchType        = "searchType"
	ConfigKeyResultsPerSearch  = "resultsPerSearch"
	ConfigKeyExcludedDomains   = "excludedDomains"

	ExaSearchTypeAuto = "auto"
	ExaSearchTypeFast = "fast"
	ExaSearchTypeDeep = "deep"

	DefaultDailyRequestLimit = 200
	MinDailyRequestLimit     = 1
	MaxDailyRequestLimit     = 10000
	DefaultResultsPerSearch  = 5
	MinResultsPerSearch      = 1
	MaxResultsPerSearch      = 10
	MaxExcludedDomains       = 100
)

var hostnamePattern = regexp.MustCompile(`^(\*\.)?([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$`)

type Spec struct {
	Fields                 []configspec.Field
	SupportsTestConnect    bool
	Tools                  []string
	ReturnsExternalContent bool
}

func (s Spec) Configured(configuration map[string]any) bool {
	return configspec.HasRequired(configuration, s.Fields)
}

func (s Spec) HasTool(name string) bool {
	for _, tool := range s.Tools {
		if tool == name {
			return true
		}
	}

	return false
}

var specs = map[Type]Spec{
	TypeExa: exaSpec(),
}

func SpecFor(t Type) (Spec, bool) {
	spec, ok := specs[t]
	return spec, ok
}

func ExtensionForTool(name string) (Type, bool) {
	for _, t := range AllTypes() {
		if specs[t].HasTool(name) {
			return t, true
		}
	}

	return "", false
}

func exaSpec() Spec {
	return Spec{
		Fields: []configspec.Field{
			{
				Key:         ConfigKeyAPIKey,
				Label:       "API key",
				Type:        configspec.FieldTypePassword,
				Required:    true,
				Sensitive:   true,
				Placeholder: "Your Exa API key",
				HelpText:    "Create a key at dashboard.exa.ai. Trenova stores it encrypted and only sends it to api.exa.ai.",
			},
			{
				Key:      ConfigKeySearchType,
				Label:    "Search depth",
				Type:     configspec.FieldTypeSelect,
				Default:  ExaSearchTypeAuto,
				Options:  []string{ExaSearchTypeAuto, ExaSearchTypeFast, ExaSearchTypeDeep},
				HelpText: "Auto balances speed and quality. Fast answers quickest. Deep reads more of the web and costs more per search.",
			},
			{
				Key:      ConfigKeyResultsPerSearch,
				Label:    "Results per search",
				Type:     configspec.FieldTypeNumber,
				Default:  strconv.Itoa(DefaultResultsPerSearch),
				HelpText: "How many pages one search returns to the agent, from 1 to 10. More results cost more and take more of the agent's attention.",
			},
			{
				Key:      ConfigKeyDailyRequestLimit,
				Label:    "Daily request limit",
				Type:     configspec.FieldTypeNumber,
				Default:  strconv.Itoa(DefaultDailyRequestLimit),
				HelpText: "The most searches and page reads all agents in this organization may make in a day (UTC). Agents are told the limit is reached rather than retrying.",
			},
			{
				Key:         ConfigKeyExcludedDomains,
				Label:       "Excluded sites",
				Type:        configspec.FieldTypeString,
				Placeholder: "example.com, *.example.org",
				HelpText:    "Sites agents never receive results from, separated by commas. Use *.example.com to exclude every subdomain.",
			},
		},
		SupportsTestConnect:    true,
		Tools:                  []string{ToolWebSearch, ToolWebRead},
		ReturnsExternalContent: true,
	}
}

type ExaSettings struct {
	APIKey            string
	SearchType        string
	ResultsPerSearch  int
	DailyRequestLimit int
	ExcludedDomains   []string
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
	settings := ExaSettings{
		APIKey:     strings.TrimSpace(values[ConfigKeyAPIKey]),
		SearchType: strings.TrimSpace(values[ConfigKeySearchType]),
	}
	if settings.SearchType == "" {
		settings.SearchType = ExaSearchTypeAuto
	}
	switch settings.SearchType {
	case ExaSearchTypeAuto, ExaSearchTypeFast, ExaSearchTypeDeep:
	default:
		multiErr.Add(
			"configuration."+ConfigKeySearchType,
			errortypes.ErrInvalid,
			"Search depth must be auto, fast or deep",
		)
	}

	settings.ResultsPerSearch = parseBoundedInt(boundedIntField{
		values:   values,
		key:      ConfigKeyResultsPerSearch,
		fallback: DefaultResultsPerSearch,
		min:      MinResultsPerSearch,
		max:      MaxResultsPerSearch,
		message:  "Results per search must be a whole number from 1 to 10",
	}, multiErr)
	settings.DailyRequestLimit = DailyRequestLimit(values, multiErr)
	settings.ExcludedDomains = ParseDomainList(values[ConfigKeyExcludedDomains], multiErr)

	return settings
}

func DailyRequestLimit(values map[string]string, multiErr *errortypes.MultiError) int {
	return parseBoundedInt(boundedIntField{
		values:   values,
		key:      ConfigKeyDailyRequestLimit,
		fallback: DefaultDailyRequestLimit,
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

func ParseDomainList(raw string, multiErr *errortypes.MultiError) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == ' ' || r == '\t' || r == ';'
	})
	domains := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		domain := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(part)), ".")
		if domain == "" {
			continue
		}
		if !hostnamePattern.MatchString(domain) {
			multiErr.Add(
				"configuration."+ConfigKeyExcludedDomains,
				errortypes.ErrInvalid,
				"Excluded sites must be hostnames such as example.com or *.example.com; "+
					"\""+part+"\" is not",
			)
			continue
		}
		if _, dup := seen[domain]; dup {
			continue
		}
		seen[domain] = struct{}{}
		domains = append(domains, domain)
	}
	if len(domains) > MaxExcludedDomains {
		multiErr.Add(
			"configuration."+ConfigKeyExcludedDomains,
			errortypes.ErrInvalid,
			"No more than 100 sites can be excluded",
		)
		return domains[:MaxExcludedDomains]
	}

	return domains
}

func ValidateSettings(t Type, values map[string]string, multiErr *errortypes.MultiError) {
	switch t {
	case TypeExa:
		parseExaSettings(values, multiErr)
	default:
		DailyRequestLimit(values, multiErr)
	}
}
