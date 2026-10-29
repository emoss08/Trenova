package agentextension

import (
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/configspec"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/urlutils"
)

const (
	ConfigKeySearchType       = "searchType"
	ConfigKeyResultsPerSearch = "resultsPerSearch"
	ConfigKeyExcludedDomains  = "excludedDomains"

	DefaultResultsPerSearch = 5
	MinResultsPerSearch     = 1
	MaxResultsPerSearch     = 10
	MaxExcludedDomains      = 100
)

type WebResearchSettings struct {
	APIKey            string
	ResultsPerSearch  int
	DailyRequestLimit int
	ExcludedDomains   []string
}

type searchDepthField struct {
	fallback string
	options  []string
	helpText string
	message  string
}

func (f searchDepthField) field() configspec.Field {
	return configspec.Field{
		Key:      ConfigKeySearchType,
		Label:    "Search depth",
		Type:     configspec.FieldTypeSelect,
		Default:  f.fallback,
		Options:  f.options,
		HelpText: f.helpText,
	}
}

func (f searchDepthField) parse(values map[string]string, multiErr *errortypes.MultiError) string {
	return parseChoice(&choiceField{
		values:   values,
		key:      ConfigKeySearchType,
		fallback: f.fallback,
		options:  f.options,
		message:  f.message,
	}, multiErr)
}

type webResearchFieldsParams struct {
	apiKey configspec.Field
	vendor []configspec.Field
}

func webResearchFields(params *webResearchFieldsParams) []configspec.Field {
	fields := make([]configspec.Field, 0, len(params.vendor)+4)
	fields = append(fields, params.apiKey)
	fields = append(fields, params.vendor...)

	return append(fields,
		configspec.Field{
			Key:      ConfigKeyResultsPerSearch,
			Label:    "Results per search",
			Type:     configspec.FieldTypeNumber,
			Default:  strconv.Itoa(DefaultResultsPerSearch),
			HelpText: "How many pages one search returns to the agent, from 1 to 10. More results cost more and take more of the agent's attention.",
		},
		DailyRequestLimitField(
			"The most searches and page reads all agents in this organization may make in a day (UTC). Agents are told the limit is reached rather than retrying.",
		),
		configspec.Field{
			Key:         ConfigKeyExcludedDomains,
			Label:       "Excluded sites",
			Type:        configspec.FieldTypeString,
			Placeholder: "example.com, *.example.org",
			HelpText:    "Sites agents never receive results from, separated by commas. Use *.example.com to exclude every subdomain.",
		},
	)
}

func ParseWebResearchSettings(values map[string]string) (WebResearchSettings, error) {
	multiErr := errortypes.NewMultiError()
	settings := parseWebResearchSettings(values, multiErr)
	if multiErr.HasErrors() {
		return WebResearchSettings{}, multiErr
	}

	return settings, nil
}

func parseWebResearchSettings(
	values map[string]string,
	multiErr *errortypes.MultiError,
) WebResearchSettings {
	return WebResearchSettings{
		APIKey: strings.TrimSpace(values[ConfigKeyAPIKey]),
		ResultsPerSearch: parseBoundedInt(boundedIntField{
			values:   values,
			key:      ConfigKeyResultsPerSearch,
			fallback: DefaultResultsPerSearch,
			min:      MinResultsPerSearch,
			max:      MaxResultsPerSearch,
			message:  "Results per search must be a whole number from 1 to 10",
		}, multiErr),
		DailyRequestLimit: DailyRequestLimit(values, multiErr),
		ExcludedDomains:   ParseDomainList(values[ConfigKeyExcludedDomains], multiErr),
	}
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
		if !urlutils.IsHostPattern(domain) {
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

func NormalizeHostnames(raw []string) []string {
	out := make([]string, 0, len(raw))
	for _, value := range raw {
		host := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), ".")
		if host == "" || slices.Contains(out, host) {
			continue
		}
		out = append(out, host)
	}

	return out
}
