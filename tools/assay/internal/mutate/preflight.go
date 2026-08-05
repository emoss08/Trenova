package mutate

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Preflight runs the tests that mutants will be judged by, against unmutated
// code, and reports any that already fail.
//
// This is not optional rigour. A covering test that fails for reasons unrelated
// to a mutant marks that mutant killed, and every such false kill inflates the
// mutation score. Silently reporting a flattering number is worse than reporting
// none, so a red suite has to be surfaced before any mutant is judged.
type Preflight struct {
	Failing  map[string][]string
	Duration time.Duration
}

func (p Preflight) Clean() bool {
	for _, tests := range p.Failing {
		if len(tests) > 0 {
			return false
		}
	}

	return true
}

func (p Preflight) Names() []string {
	var out []string
	for pkg, tests := range p.Failing {
		for _, test := range tests {
			out = append(out, pkg+"."+test)
		}
	}
	sort.Strings(out)

	return out
}

func RunPreflight(ctx context.Context, mutants []Mutant, opts ExecuteOptions) (Preflight, error) {
	started := time.Now()

	wanted := make(map[string]map[string]struct{})
	for _, m := range mutants {
		for pkg, tests := range m.tests {
			set := wanted[pkg]
			if set == nil {
				set = make(map[string]struct{})
				wanted[pkg] = set
			}
			for _, test := range tests {
				set[test] = struct{}{}
			}
		}
	}

	result := Preflight{Failing: make(map[string][]string)}

	for _, pkg := range sortedSetKeys(wanted) {
		tests := setToSorted(wanted[pkg])
		if len(tests) == 0 {
			continue
		}

		failing, err := failingTests(ctx, opts, pkg, tests)
		if err != nil {
			return Preflight{}, err
		}
		if len(failing) > 0 {
			result.Failing[pkg] = failing
		}
	}

	result.Duration = time.Since(started)

	return result, nil
}

func failingTests(ctx context.Context, opts ExecuteOptions, pkg string, tests []string) ([]string, error) {
	args := []string{"test", "-count=1", "-run", RunPattern(tests)}
	if len(opts.Tags) > 0 {
		args = append(args, "-tags", strings.Join(opts.Tags, ","))
	}
	args = append(args, pkg)

	out, _, err := run(ctx, opts.Root, opts.Env, 0, "go", args...)
	if err == nil {
		return nil, nil
	}

	failing := parseFailures(out)
	if len(failing) == 0 {
		// The package did not build, or failed outside any test. Either way the
		// mutants judged by it cannot be trusted.
		return nil, fmt.Errorf("preflight of %s failed without naming a test: %w", pkg, err)
	}

	return failing, nil
}

func parseFailures(out []byte) []string {
	seen := make(map[string]struct{})

	for line := range strings.SplitSeq(string(out), "\n") {
		trimmed := strings.TrimSpace(line)
		rest, ok := strings.CutPrefix(trimmed, "--- FAIL:")
		if !ok {
			continue
		}
		name := strings.TrimSpace(rest)
		if idx := strings.IndexAny(name, " \t("); idx > 0 {
			name = name[:idx]
		}
		// Subtest failures are reported as Parent/Sub; the parent is what -run selects.
		if slash := strings.IndexByte(name, '/'); slash > 0 {
			name = name[:slash]
		}
		if name != "" {
			seen[name] = struct{}{}
		}
	}

	return setToSorted(seen)
}

// Exclude drops known-failing tests from every mutant's plan so the remaining
// judgements stay meaningful. A mutant left with no tests becomes no-coverage
// rather than being silently counted as killed.
func Exclude(mutants []Mutant, failing map[string][]string) []Mutant {
	if len(failing) == 0 {
		return mutants
	}

	blocked := make(map[string]map[string]struct{}, len(failing))
	for pkg, tests := range failing {
		set := make(map[string]struct{}, len(tests))
		for _, test := range tests {
			set[test] = struct{}{}
		}
		blocked[pkg] = set
	}

	out := make([]Mutant, 0, len(mutants))
	for _, m := range mutants {
		plan := make(TestPlan, len(m.tests))
		for pkg, tests := range m.tests {
			kept := make([]string, 0, len(tests))
			for _, test := range tests {
				if _, bad := blocked[pkg][test]; bad {
					continue
				}
				kept = append(kept, test)
			}
			if len(kept) > 0 {
				plan[pkg] = kept
			}
		}
		m.tests = plan
		out = append(out, m)
	}

	return out
}

func sortedSetKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for key := range m {
		out = append(out, key)
	}
	sort.Strings(out)

	return out
}

func setToSorted(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)

	return out
}
