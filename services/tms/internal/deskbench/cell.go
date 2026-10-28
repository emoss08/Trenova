package deskbench

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
)

const (
	effortSeparator = "@"
	suffixSeparator = "+"
	valueSeparator  = "="
	advisorSuffix   = "advisor"
)

type Cell struct {
	Provider *aiprovider.Provider
	Effort   aiprovider.ReasoningEffort
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

func (c Cell) pin() ProviderPin {
	return ProviderPin{ProviderID: providerID(c.Provider), Effort: c.Effort, Advisor: c.Advisor}
}

type cellRef struct {
	name    string
	effort  aiprovider.ReasoningEffort
	advisor *llmports.AdvisorSpec
}

func parseCellRef(ref string) (cellRef, error) {
	parts := strings.Split(strings.TrimSpace(ref), suffixSeparator)
	name, effort, err := splitEffort(strings.TrimSpace(parts[0]))
	if err != nil {
		return cellRef{}, err
	}
	parsed := cellRef{name: name, effort: effort}
	for _, suffix := range parts[1:] {
		if err = parsed.applySuffix(strings.TrimSpace(suffix)); err != nil {
			return cellRef{}, err
		}
	}

	return parsed, nil
}

func (r *cellRef) applySuffix(suffix string) error {
	key, value, valued := strings.Cut(suffix, valueSeparator)
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if !strings.EqualFold(key, advisorSuffix) {
		return fmt.Errorf("%q is not a cell suffix (one of %s%s)", suffixSeparator+suffix,
			suffixSeparator, advisorSuffix)
	}
	if r.advisor != nil {
		return fmt.Errorf("%s%s is given twice", suffixSeparator, advisorSuffix)
	}
	if valued && value == "" {
		return fmt.Errorf("%s%s%s names no model", suffixSeparator, advisorSuffix, valueSeparator)
	}
	r.advisor = llmports.DefaultAdvisor(value)

	return nil
}

func splitEffort(ref string) (string, aiprovider.ReasoningEffort, error) {
	at := strings.LastIndex(ref, effortSeparator)
	if at < 0 {
		return ref, "", nil
	}

	name := strings.TrimSpace(ref[:at])
	wanted := strings.TrimSpace(ref[at+1:])
	for _, effort := range aiprovider.AllReasoningEfforts() {
		if strings.EqualFold(string(effort), wanted) {
			return name, effort, nil
		}
	}

	return "", "", fmt.Errorf("%q is not a reasoning effort (one of %s)", wanted,
		effortNames())
}

func effortNames() string {
	efforts := aiprovider.AllReasoningEfforts()
	names := make([]string, 0, len(efforts))
	for _, effort := range efforts {
		names = append(names, strings.ToLower(string(effort)))
	}

	return strings.Join(names, ", ")
}

var errAdvisorNeedsAnthropic = errors.New(
	suffixSeparator + advisorSuffix + " needs a named provider that speaks the Anthropic Messages API",
)

func checkAdvisor(provider *aiprovider.Provider, advisor *llmports.AdvisorSpec) error {
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

func (s *Session) Cell(ctx context.Context, ref string) (Cell, error) {
	parsed, err := parseCellRef(ref)
	if err != nil {
		return Cell{}, err
	}
	provider, err := s.Provider(ctx, parsed.name)
	if err != nil {
		return Cell{}, err
	}
	if err = checkAdvisor(provider, parsed.advisor); err != nil {
		return Cell{}, fmt.Errorf("%s: %w", ProviderLabel(provider), err)
	}

	return Cell{Provider: provider, Effort: parsed.effort, Advisor: parsed.advisor}, nil
}
