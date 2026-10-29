package benchcell

import (
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/deskbench/benchrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aireasoning"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	effortSeparator = "@"
	suffixSeparator = "+"
	valueSeparator  = "="
	advisorSuffix   = "advisor"
)

type Cell struct {
	Provider *aiprovider.Provider
	Effort   aireasoning.ReasoningEffort
	Advisor  *llmports.AdvisorSpec
}

func (c Cell) Label() string {
	label := ProviderLabel(c.Provider)
	if c.Effort != "" {
		label += " " + effortSeparator + strings.ToLower(string(c.Effort))
	}
	if c.Advisor != nil {
		label += " " + advisorLabel(c.Advisor)
	}

	return label
}

func advisorLabel(advisor *llmports.AdvisorSpec) string {
	if advisor.Model == llmports.DefaultAdvisorModel {
		return suffixSeparator + advisorSuffix
	}

	return suffixSeparator + advisorSuffix + valueSeparator + advisor.Model
}

func (c Cell) Pin() benchrecord.ProviderPin {
	return benchrecord.ProviderPin{
		ProviderID: ProviderID(c.Provider),
		Effort:     c.Effort,
		Advisor:    c.Advisor,
	}
}

type Ref struct {
	Name    string
	Effort  aireasoning.ReasoningEffort
	Advisor *llmports.AdvisorSpec
}

func ParseRef(ref string) (Ref, error) {
	parts := strings.Split(strings.TrimSpace(ref), suffixSeparator)
	name, effort, err := SplitEffort(strings.TrimSpace(parts[0]))
	if err != nil {
		return Ref{}, err
	}
	parsed := Ref{Name: name, Effort: effort}
	for _, suffix := range parts[1:] {
		if err = parsed.applySuffix(strings.TrimSpace(suffix)); err != nil {
			return Ref{}, err
		}
	}

	return parsed, nil
}

func (r *Ref) applySuffix(suffix string) error {
	key, value, valued := strings.Cut(suffix, valueSeparator)
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if !strings.EqualFold(key, advisorSuffix) {
		return fmt.Errorf("%q is not a cell suffix (one of %s%s)", suffixSeparator+suffix,
			suffixSeparator, advisorSuffix)
	}
	if r.Advisor != nil {
		return fmt.Errorf("%s%s is given twice", suffixSeparator, advisorSuffix)
	}
	if valued && value == "" {
		return fmt.Errorf("%s%s%s names no model", suffixSeparator, advisorSuffix, valueSeparator)
	}
	r.Advisor = llmports.DefaultAdvisor(value)

	return nil
}

func SplitEffort(ref string) (string, aireasoning.ReasoningEffort, error) {
	at := strings.LastIndex(ref, effortSeparator)
	if at < 0 {
		return ref, "", nil
	}

	name := strings.TrimSpace(ref[:at])
	wanted := strings.TrimSpace(ref[at+1:])
	for _, effort := range aireasoning.AllReasoningEfforts() {
		if strings.EqualFold(string(effort), wanted) {
			return name, effort, nil
		}
	}

	return "", "", fmt.Errorf("%q is not a reasoning effort (one of %s)", wanted,
		effortNames())
}

func effortNames() string {
	efforts := aireasoning.AllReasoningEfforts()
	names := make([]string, 0, len(efforts))
	for _, effort := range efforts {
		names = append(names, strings.ToLower(string(effort)))
	}

	return strings.Join(names, ", ")
}

var errAdvisorNeedsAnthropic = errors.New(
	suffixSeparator + advisorSuffix + " needs a named provider that speaks the Anthropic Messages API",
)

func CheckAdvisor(provider *aiprovider.Provider, advisor *llmports.AdvisorSpec) error {
	if advisor == nil {
		return nil
	}
	err := provider.CheckAdvisor()
	switch {
	case err == nil:
		return nil
	case errors.Is(err, aiprovider.ErrAdvisorNeedsAnthropic):
		return errAdvisorNeedsAnthropic
	default:
		return fmt.Errorf("%s%s: %w", suffixSeparator, advisorSuffix, err)
	}
}

const DefaultProviderLabel = "default"

func ProviderLabel(provider *aiprovider.Provider) string {
	if provider == nil {
		return DefaultProviderLabel
	}

	return provider.Name + " (" + provider.Model + ")"
}

func ProviderID(provider *aiprovider.Provider) pulid.ID {
	if provider == nil {
		return pulid.Nil
	}

	return provider.ID
}
