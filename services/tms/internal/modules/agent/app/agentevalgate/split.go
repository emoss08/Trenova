package agentevalgate

import (
	"cmp"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/evalsplit"
)

const unheldDesk = "No desk"

type SelectionSplit struct {
	HeldOut map[string]struct{}
	Ledger  evalsplit.Ledger
}

func DeskStrata() map[string]evalsplit.Stratum {
	templates := agentdefinition.AllTemplates()
	slices.SortFunc(templates, func(a, b agentdefinition.Template) int {
		return cmp.Or(
			cmp.Compare(len(a.StarterTools()), len(b.StarterTools())),
			cmp.Compare(a.Label(), b.Label()),
		)
	})

	strata := make(map[string]evalsplit.Stratum, 512)
	for _, template := range templates {
		mode := evalsplit.Chat
		if template.StarterTrigger() != agentdefinition.TriggerChat {
			mode = evalsplit.Unattended
		}
		for _, name := range template.StarterTools() {
			if _, held := strata[name]; !held {
				strata[name] = evalsplit.Stratum{Desk: template.Label(), Mode: mode}
			}
		}
	}

	return strata
}

func CaseStratum(selection SelectionCase, strata map[string]evalsplit.Stratum) evalsplit.Stratum {
	for _, name := range selection.Expect {
		if stratum, ok := strata[name]; ok {
			return stratum
		}
	}

	return evalsplit.Stratum{Desk: unheldDesk, Mode: evalsplit.Chat}
}

func SplitSelection(suite *SelectionSuite) SelectionSplit {
	strata := DeskStrata()
	items := make([]evalsplit.Item, 0, len(suite.Cases))
	for _, selection := range suite.Cases {
		items = append(items, evalsplit.Item{
			Key:     selection.Request,
			Stratum: CaseStratum(selection, strata),
		})
	}
	splits := evalsplit.Assign(items)

	split := SelectionSplit{
		HeldOut: make(map[string]struct{}, len(items)/4+1),
		Ledger:  evalsplit.LedgerOf(items, splits),
	}
	for idx := range items {
		if splits[idx] == evalsplit.HeldOut {
			split.HeldOut[items[idx].Key] = struct{}{}
		}
	}

	return split
}

func (s SelectionSplit) Partition(suite *SelectionSuite) (development, heldOut *SelectionSuite) {
	development = &SelectionSuite{Nothing: suite.Nothing}
	heldOut = &SelectionSuite{}
	for _, selection := range suite.Cases {
		if _, held := s.HeldOut[selection.Request]; held {
			heldOut.Cases = append(heldOut.Cases, selection)
			continue
		}
		development.Cases = append(development.Cases, selection)
	}

	return development, heldOut
}
