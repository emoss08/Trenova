package agentevalgate

import (
	"fmt"
	"math"
	"os"
	"slices"
	"strings"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/core/services/productguideservice"
	"gopkg.in/yaml.v3"
)

const SelectionDepth = 5

type SelectionSuite struct {
	Cases   []SelectionCase `yaml:"cases"`
	Nothing []string        `yaml:"nothing"`
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

type SelectionFloorSet struct {
	Keyword SelectionFloors  `json:"keyword"`
	Hybrid  *SelectionFloors `json:"hybrid,omitempty"`
}

type Finder func(request string) ([]string, error)

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
	Spurious  []SelectionOutcome
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
	report, _ := EvaluateWith(suite, k.KeywordFinder())

	return report
}

func (k *Kit) KeywordFinder() Finder {
	return func(request string) ([]string, error) {
		return descriptorNames(k.Catalog.Find(nil, request, SelectionDepth)), nil
	}
}

func (k *Kit) HybridFinder(fixture *EmbeddingFixture) Finder {
	items := k.Catalog.Items()

	return func(request string) ([]string, error) {
		similarity, err := fixture.Similarities(request, items)
		if err != nil {
			return nil, err
		}

		return descriptorNames(k.Catalog.FindHybrid(agenttoolcatalog.Query{
			Text:     request,
			Limit:    SelectionDepth,
			Semantic: agenttoolcatalog.NewSemantic(similarity),
		})), nil
	}
}

func GuideKeywordFinder(guide *productguideservice.Service) Finder {
	return func(request string) ([]string, error) {
		return guide.RankPaths(request, nil, SelectionDepth), nil
	}
}

func GuideHybridFinder(guide *productguideservice.Service, fixture *EmbeddingFixture) Finder {
	items := GuideItems()

	return func(request string) ([]string, error) {
		similarity, err := fixture.Similarities(request, items)
		if err != nil {
			return nil, err
		}

		return guide.RankPaths(request, similarity, SelectionDepth), nil
	}
}

func Combined(suites ...*SelectionSuite) *SelectionSuite {
	combined := &SelectionSuite{}
	for _, suite := range suites {
		combined.Cases = append(combined.Cases, suite.Cases...)
		combined.Nothing = append(combined.Nothing, suite.Nothing...)
	}

	return combined
}

func (s *SelectionSuite) Requests() []string {
	requests := make([]string, 0, len(s.Cases)+len(s.Nothing))
	for _, selection := range s.Cases {
		requests = append(requests, selection.Request)
	}

	return append(requests, s.Nothing...)
}

func EvaluateWith(suite *SelectionSuite, find Finder) (SelectionReport, error) {
	report := SelectionReport{Outcomes: make([]SelectionOutcome, 0, len(suite.Cases))}

	for _, request := range suite.Nothing {
		ranked, err := find(request)
		if err != nil {
			return report, err
		}
		if len(ranked) > 0 {
			report.Spurious = append(report.Spurious, SelectionOutcome{
				Case:   SelectionCase{Request: request},
				Ranked: ranked,
			})
		}
	}

	if len(suite.Cases) == 0 {
		return report, nil
	}

	recalled, top := 0, 0
	for _, selection := range suite.Cases {
		ranked, err := find(selection.Request)
		if err != nil {
			return report, err
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

	return report, nil
}

func (r SelectionReport) SpuriousMatches() string {
	var builder strings.Builder
	for _, outcome := range r.Spurious {
		fmt.Fprintf(&builder, "- %q should find nothing, found %s\n",
			outcome.Case.Request, strings.Join(outcome.Ranked, ", "))
	}

	return builder.String()
}

func descriptorNames(found []serviceports.AgentToolDescriptor) []string {
	names := make([]string, 0, len(found))
	for _, descriptor := range found {
		names = append(names, descriptor.Name)
	}

	return names
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
