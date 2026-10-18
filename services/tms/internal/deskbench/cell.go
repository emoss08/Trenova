package deskbench

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
)

const effortSeparator = "@"

type Cell struct {
	Provider *aiprovider.Provider
	Effort   aiprovider.ReasoningEffort
}

func (c Cell) Label() string {
	label := ProviderLabel(c.Provider)
	if c.Effort != "" {
		label += " " + effortSeparator + strings.ToLower(string(c.Effort))
	}

	return label
}

func (c Cell) pin() ProviderPin {
	return ProviderPin{ProviderID: providerID(c.Provider), Effort: c.Effort}
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

func (s *Session) Cell(ctx context.Context, ref string) (Cell, error) {
	name, effort, err := splitEffort(strings.TrimSpace(ref))
	if err != nil {
		return Cell{}, err
	}
	provider, err := s.Provider(ctx, name)
	if err != nil {
		return Cell{}, err
	}

	return Cell{Provider: provider, Effort: effort}, nil
}
