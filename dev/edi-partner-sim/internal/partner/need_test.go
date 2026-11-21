package partner_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/play"
	"github.com/emoss08/trenova/edi-partner-sim/internal/scenario"
	"github.com/emoss08/trenova/pkg/lading"
)

const (
	needSeeds    = 200
	scenarioRuns = 12
	statusHeld   = "Held"
)

func TestANeedNarrowsTheDrawAndNothingElse(t *testing.T) {
	t.Parallel()

	definition := openPartner(t, "retailer")
	original := partner.Variation{Kind: partner.KindOriginal}
	roomy := partner.Variation{
		Kind: partner.KindOriginal,
		Need: &partner.LoadNeed{MaxWeight: 80_000},
	}
	for seed := uint64(1); seed <= needSeeds; seed++ {
		plain, _, err := definition.Transaction("204", seed, original)
		if err != nil {
			t.Fatal(err)
		}
		narrowed, _, err := definition.Transaction("204", seed, roomy)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(plain, narrowed) {
			t.Fatalf("seed %d: a need every load meets changed the tender", seed)
		}
	}
	hazmat, noHazmat := true, false
	cases := []struct {
		partner string
		need    partner.LoadNeed
		holds   func(*partner.Load) bool
	}{
		{"retailer", partner.LoadNeed{MaxWeight: 43_500}, func(l *partner.Load) bool {
			return l.Weight() <= 43_500
		}},
		{"retailer", partner.LoadNeed{MinWeight: 44_001}, func(l *partner.Load) bool {
			return l.Weight() >= 44_001
		}},
		{"manufacturer", partner.LoadNeed{Stops: 2, Hazmat: &hazmat}, func(l *partner.Load) bool {
			return len(l.Stops) == 2 && carriesHazmat(l)
		}},
		{"manufacturer", partner.LoadNeed{Hazmat: &noHazmat}, func(l *partner.Load) bool {
			return !carriesHazmat(l)
		}},
		{"grocer", partner.LoadNeed{OneRange: true}, func(l *partner.Load) bool {
			return !l.TemperatureConflict()
		}},
	}
	for _, c := range cases {
		definition := openPartner(t, c.partner)
		variation := partner.Variation{Kind: partner.KindOriginal, Need: &c.need}
		for seed := uint64(1); seed <= needSeeds; seed++ {
			_, load, err := definition.Transaction("204", seed, variation)
			if err != nil {
				t.Fatalf("%s %s: %v", c.partner, c.need.String(), err)
			}
			if !c.holds(load) {
				t.Fatalf("%s seed %d drew a load that is not %s", c.partner, seed, c.need.String())
			}
		}
	}
}

func TestANeedNoLoadCanMeetIsRefused(t *testing.T) {
	t.Parallel()

	hazmat := true
	cases := []struct {
		partner string
		need    partner.LoadNeed
	}{
		{"retailer", partner.LoadNeed{MaxWeight: 5_000}},
		{"retailer", partner.LoadNeed{MinWeight: 40_000, MaxWeight: 30_000}},
		{"retailer", partner.LoadNeed{Stops: 9}},
		{"grocer", partner.LoadNeed{Hazmat: &hazmat}},
		{"manufacturer", partner.LoadNeed{OneRange: true}},
	}
	for _, c := range cases {
		err := openPartner(t, c.partner).CheckNeed("204", &c.need)
		if err == nil {
			t.Errorf("%s accepted a need for %s", c.partner, c.need.String())
		}
	}
	if err := openPartner(t, "carrier").CheckNeed("204", &partner.LoadNeed{}); err == nil {
		t.Error("a carrier, which sends no 204, accepted a load need")
	}
}

func TestEveryGeneratedWindowOpensAfterTheOneBefore(t *testing.T) {
	t.Parallel()

	for _, definition := range openAll(t) {
		if definition.Partner.Role == partner.RoleCarrier {
			continue
		}
		for seed := uint64(1); seed <= needSeeds; seed++ {
			load, err := definition.Load("204", seed)
			if err != nil {
				t.Fatal(err)
			}
			for index := range load.Stops {
				window := load.Stops[index].Window
				if !window.End.After(window.Start) {
					t.Errorf("%s seed %d stop %d: the window %s-%s is empty", definition.Name(),
						seed, index+1, window.Start, window.End)
				}
				if index > 0 && !window.Start.After(load.Stops[index-1].Window.Start) {
					t.Errorf("%s seed %d stop %d opens at %s, not after stop %d at %s",
						definition.Name(), seed, index+1, window.Start, index,
						load.Stops[index-1].Window.Start)
				}
			}
		}
	}
}

func TestShippedScenariosTenderLoadsTheirExpectationsHoldFor(t *testing.T) {
	t.Parallel()

	for _, definition := range openAll(t) {
		if definition.Partner.Role == partner.RoleCarrier {
			continue
		}
		t.Run(definition.Name(), func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			project := encodeProject(t, definition)
			scenarios, err := scenario.Load(definition.Dir)
			if err != nil {
				t.Fatal(err)
			}
			for index, s := range scenarios {
				for step := range s.Steps {
					send := s.Steps[step].Send
					if send == nil || send.Of != "" {
						continue
					}
					held := expectsHeld(s, step, send.As)
					for run := uint64(1); run <= scenarioRuns; run++ {
						base := play.SeedOf(run, uint64(index)*scenario.MaxSeedsPerCase)
						for offset := range uint64(max(send.Count, 1)) {
							checkTender(ctx, t, project, definition, s, tenderCase{
								seed: base + send.Seed + offset,
								need: send.Load,
								held: held,
							})
						}
					}
				}
			}
		})
	}
}

type tenderCase struct {
	seed uint64
	need *partner.LoadNeed
	held bool
}

func checkTender(
	ctx context.Context,
	t *testing.T,
	project *lading.Encoded,
	definition *partner.Definition,
	s *scenario.Scenario,
	c tenderCase,
) {
	t.Helper()
	written, _, err := definition.Interchange("204", c.seed,
		partner.Variation{Kind: partner.KindOriginal, Need: c.need}, controlsAt(int64(c.seed)))
	if err != nil {
		t.Fatalf("%s seed %d: %v", s.Slug, c.seed, err)
	}
	result, err := runTenders(ctx, project, "tenderIn", written.Bytes)
	if err != nil || len(result.Transactions) != 1 {
		t.Fatalf("%s seed %d: %v", s.Slug, c.seed, err)
	}
	var stopped []string
	for _, outcome := range result.Transactions[0].Outcomes {
		switch outcome.Kind {
		case lading.OutcomeHold, lading.OutcomeDecline, lading.OutcomeReject:
			stopped = append(stopped, outcome.Kind+": "+outcome.Message)
		}
	}
	switch {
	case c.held && len(stopped) == 0:
		t.Errorf("%s seed %d: the scenario expects its tender held, and the rules took it",
			s.Slug, c.seed)
	case !c.held && len(stopped) > 0:
		t.Errorf("%s seed %d: the scenario expects its tender taken, and the rules %v; "+
			"state what the scenario needs from the load (send.load)", s.Slug, c.seed, stopped)
	}
}

func expectsHeld(s *scenario.Scenario, from int, name string) bool {
	for _, step := range s.Steps[from+1:] {
		if step.Send != nil && step.Send.Of == name {
			return false
		}
		if step.Expect != nil && step.Expect.Flow == name {
			return step.Expect.Status == statusHeld
		}
	}
	return false
}

func carriesHazmat(load *partner.Load) bool {
	for _, stop := range load.Stops {
		for _, line := range stop.Lines {
			if line.Hazmat != nil {
				return true
			}
		}
	}
	return false
}
