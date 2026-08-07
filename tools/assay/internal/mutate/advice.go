package mutate

import (
	"sort"
	"strings"
	"time"
)

// Advice turns a surviving mutant into a prescription: the covering test whose
// assertions sit closest to the mutated line, and what input would have told
// the original from the mutant. A survivor without advice is a metric; with
// it, it is a to-do item.
type Advice struct {
	// TestPackage and Test name the most focused covering test — the cheapest
	// place to add the missing assertion, because it already executes the line.
	TestPackage string `json:"testPackage"`
	Test        string `json:"test"`
	// Focus is how many lines that test covers in total. Smaller is closer.
	Focus int `json:"focusLines,omitempty"`
	// Duration is the test's indexed cost, so the reader knows the price of
	// iterating on it.
	Duration     time.Duration `json:"durationNanos,omitempty"`
	Prescription string        `json:"prescription"`
}

// TestProfile resolves a covering test's coverage breadth and indexed
// duration. Unknown tests report ok=false and rank last.
type TestProfile func(pkg, test string) (lines int, duration time.Duration, ok bool)

// Advise builds the prescription for one surviving mutant. Nil when nothing
// covers it — an uncovered site is a coverage gap, not an assertion gap, and
// the coverage report already owns that story.
func Advise(m Mutant, profile TestProfile) *Advice {
	type candidate struct {
		pkg, test string
		focus     int
		duration  time.Duration
		profiled  bool
	}

	var candidates []candidate
	for pkg, tests := range m.tests {
		for _, test := range tests {
			c := candidate{pkg: pkg, test: test}
			if profile != nil {
				c.focus, c.duration, c.profiled = profile(pkg, test)
			}
			candidates = append(candidates, c)
		}
	}
	if len(candidates) == 0 {
		return nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		// A test in the mutated package itself is the natural home for the
		// assertion; cross-package tests observe through more indirection.
		aHome, bHome := a.pkg == m.Package, b.pkg == m.Package
		if aHome != bHome {
			return aHome
		}
		if a.profiled != b.profiled {
			return a.profiled
		}
		if a.profiled && a.focus != b.focus {
			return a.focus < b.focus
		}
		if a.duration != b.duration {
			return a.duration < b.duration
		}
		if a.pkg != b.pkg {
			return a.pkg < b.pkg
		}

		return a.test < b.test
	})

	best := candidates[0]

	return &Advice{
		TestPackage:  best.pkg,
		Test:         best.test,
		Focus:        best.focus,
		Duration:     best.duration,
		Prescription: prescription(m),
	}
}

// prescription describes the input that distinguishes the original from the
// mutant — which is exactly the test case that is missing. Every covering test
// ran both versions to the same verdict, so the distinguishing input is, by
// construction, one nobody supplies.
func prescription(m Mutant) string {
	switch m.Kind {
	case KindBoundary:
		return "no test pins the boundary: add an input where both sides of `" + m.Original +
			"` are equal — the only input where `" + m.Original + "` and `" + m.Replacement + "` disagree"
	case KindArithmetic:
		return "assert the exact result of this computation; no covering test notices `" +
			m.Original + "` becoming `" + m.Replacement + "`"
	case KindEquality:
		return "cover both sides of this comparison: one input where it holds and one where it does not"
	case KindConnector:
		return "add a case where one operand is true and the other false — the only case where `" +
			m.Original + "` and `" + m.Replacement + "` disagree"
	case KindBoolean:
		return "nothing observes behaviour that depends on this constant; assert an outcome that changes when it flips"
	case KindBranch:
		if strings.Contains(m.Replacement, "&& false") {
			return "no test observes what happens when this condition is true; add one that makes it true and asserts the effect"
		}

		return "no test observes what happens when this condition is false; add one that makes it false and asserts the effect"
	case KindIncDec:
		return "assert the final value this counter reaches; its direction is unobserved"
	default:
		return "no covering test distinguishes `" + m.Original + "` from `" + m.Replacement + "`; assert the behaviour that depends on it"
	}
}

// SurvivorGroup clusters a report's survivors by the function they mutate.
type SurvivorGroup struct {
	Package   string
	Function  string
	File      string
	Survivors []Result
}

// GroupSurvivors ranks survivors worst-first: the function with the most
// survivors is the biggest assertion gap, and scattering its mutants through a
// flat list hid that. Within a group, source order; groups of equal size in
// deterministic package/function order.
func GroupSurvivors(survivors []Result) []SurvivorGroup {
	type key struct{ pkg, fn, file string }
	order := make([]key, 0, len(survivors))
	grouped := make(map[key][]Result, len(survivors))

	for _, r := range survivors {
		k := key{r.Mutant.Package, r.Mutant.Function, r.Mutant.File}
		if _, seen := grouped[k]; !seen {
			order = append(order, k)
		}
		grouped[k] = append(grouped[k], r)
	}

	out := make([]SurvivorGroup, 0, len(order))
	for _, k := range order {
		members := grouped[k]
		sort.Slice(members, func(i, j int) bool { return members[i].Mutant.Line < members[j].Mutant.Line })
		out = append(out, SurvivorGroup{Package: k.pkg, Function: k.fn, File: k.file, Survivors: members})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if len(out[i].Survivors) != len(out[j].Survivors) {
			return len(out[i].Survivors) > len(out[j].Survivors)
		}
		if out[i].Package != out[j].Package {
			return out[i].Package < out[j].Package
		}

		return out[i].Function < out[j].Function
	})

	return out
}
