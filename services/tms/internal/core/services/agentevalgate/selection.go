package agentevalgate

import (
	"fmt"
	"math"
	"os"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

const SelectionDepth = 5

type SelectionSuite struct {
	Cases []SelectionCase `yaml:"cases"`
}

type SelectionCase struct {
	Request string   `yaml:"request"`
	Expect  []string `yaml:"expect"`
}

type SelectionFloors struct {
	Cases     int     `json:"cases"`
	RecallAt5 float64 `json:"recallAt5"`
	Top1      float64 `json:"top1"`
}

type SelectionOutcome struct {
	Case   SelectionCase
	Ranked []string
	Hit    int
}

func (o SelectionOutcome) Top1() bool { return o.Hit == 1 }

func (o SelectionOutcome) Recalled() bool { return o.Hit > 0 && o.Hit <= SelectionDepth }

type SelectionReport struct {
	Outcomes  []SelectionOutcome
	RecallAt5 float64
	Top1      float64
}

func LoadSelectionSuite(path string) (*SelectionSuite, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var suite SelectionSuite
	if err = yaml.Unmarshal(raw, &suite); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	return &suite, nil
}

func (k *Kit) UnknownTools(suite *SelectionSuite) []string {
	known := k.Catalog.Names()
	unknown := make([]string, 0, 4)
	for _, selection := range suite.Cases {
		for _, name := range selection.Expect {
			if !slices.Contains(known, name) && !slices.Contains(unknown, name) {
				unknown = append(unknown, name)
			}
		}
	}

	return unknown
}

func (k *Kit) EvaluateSelection(suite *SelectionSuite) SelectionReport {
	report := SelectionReport{Outcomes: make([]SelectionOutcome, 0, len(suite.Cases))}
	if len(suite.Cases) == 0 {
		return report
	}

	recalled, top := 0, 0
	for _, selection := range suite.Cases {
		found := k.Catalog.Find(nil, selection.Request, SelectionDepth)
		ranked := make([]string, 0, len(found))
		for _, descriptor := range found {
			ranked = append(ranked, descriptor.Name)
		}

		outcome := SelectionOutcome{Case: selection, Ranked: ranked}
		for idx, name := range ranked {
			if slices.Contains(selection.Expect, name) {
				outcome.Hit = idx + 1
				break
			}
		}
		if outcome.Recalled() {
			recalled++
		}
		if outcome.Top1() {
			top++
		}
		report.Outcomes = append(report.Outcomes, outcome)
	}

	total := float64(len(suite.Cases))
	report.RecallAt5 = float64(recalled) / total
	report.Top1 = float64(top) / total

	return report
}

func (r SelectionReport) Floors() SelectionFloors {
	return SelectionFloors{
		Cases:     len(r.Outcomes),
		RecallAt5: floorTo(r.RecallAt5),
		Top1:      floorTo(r.Top1),
	}
}

func (r SelectionReport) Misses() string {
	var builder strings.Builder
	for _, outcome := range r.Outcomes {
		if outcome.Top1() {
			continue
		}
		position := "not in the top 5"
		if outcome.Recalled() {
			position = fmt.Sprintf("at %d", outcome.Hit)
		}
		fmt.Fprintf(&builder, "- %q wants %s, found %s; ranked %s\n",
			outcome.Case.Request,
			strings.Join(outcome.Case.Expect, " or "),
			position,
			strings.Join(outcome.Ranked, ", "),
		)
	}

	return builder.String()
}

func floorTo(value float64) float64 {
	return math.Floor(value*100) / 100
}
