package scenario

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
)

func TestEveryShippedScenarioIsValid(t *testing.T) {
	t.Parallel()
	definitions, err := partner.OpenAll("../../partners")
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, def := range definitions {
		scenarios, loadErr := Load(def.Dir)
		if loadErr != nil {
			t.Fatalf("%s: %v", def.Name(), loadErr)
		}
		if len(scenarios) == 0 {
			t.Fatalf("%s ships no scenarios", def.Name())
		}
		for _, s := range scenarios {
			for index := range s.Steps {
				step := &s.Steps[index]
				if step.Send == nil {
					continue
				}
				kind := partner.Kind(cmpOr(step.Send.Variation, "original"))
				if _, ok := def.Templates["204"].Purposes[kind]; !ok {
					t.Errorf("%s: steps[%d] sends a %s %s never sends", s.File, index, kind,
						def.Name())
				}
				if def.Partner.Role == partner.RoleCarrier {
					t.Errorf("%s: a carrier partner does not send tenders", s.File)
				}
			}
		}
		total += len(scenarios)
	}
	if total < 20 {
		t.Fatalf("only %d scenarios ship", total)
	}
}

func TestSeedsAreAssignedAndKeptApart(t *testing.T) {
	t.Parallel()
	s := &Scenario{Name: "seeds", Steps: []Step{
		{Send: &Send{Set: "204", As: "a"}},
		{Send: &Send{Set: "204", As: "b", Count: 3}},
		{Send: &Send{Set: "204", As: "c", Seed: 9}},
		{Send: &Send{Set: "204", As: "d"}},
	}}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	got := []uint64{s.Steps[0].Send.Seed, s.Steps[1].Send.Seed, s.Steps[2].Send.Seed,
		s.Steps[3].Send.Seed}
	want := []uint64{1, 2, 9, 5}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("seeds %v, want %v", got, want)
		}
	}
}

func TestProblemsNameTheStep(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		yaml string
		want string
	}{
		{
			name: "unknown conversation",
			yaml: "name: x\nsteps:\n  - expect: { flow: t9, status: Held }\n",
			want: `steps[0].expect.flow: "t9" names no conversation`,
		},
		{
			name: "two kinds in one step",
			yaml: "name: x\nsteps:\n  - send: { set: \"204\", as: t1 }\n    advance: { about: t1, by: 1h }\n",
			want: "it takes exactly one",
		},
		{
			name: "a bad element",
			yaml: "name: x\nsteps:\n  - send: { set: \"204\", as: t1 }\n  - expect: { wire: \"990\", about: t1, elements: { b1: A } }\n",
			want: `"b1" is not a segment and position`,
		},
		{
			name: "statuses on a 990",
			yaml: "name: x\nsteps:\n  - send: { set: \"204\", as: t1 }\n  - expect: { wire: \"990\", about: t1, statuses: [X3] }\n",
			want: "statuses check 214s",
		},
		{
			name: "a change without of",
			yaml: "name: x\nsteps:\n  - send: { set: \"204\", variation: change }\n",
			want: "a change names the conversation it varies",
		},
		{
			name: "an unknown key",
			yaml: "name: x\nsteps:\n  - send: { set: \"204\", as: t1, colour: red }\n",
			want: "colour",
		},
		{
			name: "a reused seed",
			yaml: "name: x\nsteps:\n  - send: { set: \"204\", as: a, seed: 3 }\n  - send: { set: \"204\", as: b, seed: 3 }\n",
			want: "seed 3 is already a's",
		},
		{
			name: "a bad total",
			yaml: "name: x\nsteps:\n  - send: { set: \"204\", as: t1 }\n  - expect: { wire: \"210\", about: t1, total: lots }\n",
			want: `"lots" is not tendered`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "s.yaml")
			if err := os.WriteFile(path, []byte(tt.yaml), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := LoadFile(path)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %v, want %q", err, tt.want)
			}
		})
	}
}
