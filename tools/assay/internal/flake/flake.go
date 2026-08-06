// Package flake turns the test verdicts assay already produces into evidence
// about test reliability. A flaky test is one whose verdict changed while the
// code did not: the same test, at the same package fingerprint, both passing
// and failing. That definition is what separates a flake from a regression —
// a verdict that changed *across* fingerprints is just the code changing.
package flake

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

type Verdict string

const (
	VerdictPass    Verdict = "pass"
	VerdictFail    Verdict = "fail"
	VerdictTimeout Verdict = "timeout"
)

// Observation is one test verdict at one code state. The fingerprint is the
// package's content fingerprint — sources, dependencies, build tags — which is
// exactly the identity under which a verdict is expected to be reproducible.
type Observation struct {
	Package     string  `json:"package"`
	Test        string  `json:"test"`
	Fingerprint string  `json:"fingerprint"`
	Verdict     Verdict `json:"verdict"`
	Source      string  `json:"source"`
	Detail      string  `json:"detail,omitempty"`
	Unix        int64   `json:"unix"`
}

func (o Observation) failed() bool {
	return o.Verdict == VerdictFail || o.Verdict == VerdictTimeout
}

// Evidence is the conflicting record at one fingerprint: the same code state
// observed both passing and failing.
type Evidence struct {
	Fingerprint string
	Passes      int
	Fails       int
	Sources     []string
	FirstUnix   int64
	LastUnix    int64
	Detail      string
}

// Flake is a test with at least one fingerprint of conflicting verdicts.
type Flake struct {
	Package  string
	Test     string
	Evidence []Evidence
}

func (f Flake) Passes() int {
	total := 0
	for _, e := range f.Evidence {
		total += e.Passes
	}

	return total
}

func (f Flake) Fails() int {
	total := 0
	for _, e := range f.Evidence {
		total += e.Fails
	}

	return total
}

func (f Flake) LastUnix() int64 {
	var last int64
	for _, e := range f.Evidence {
		if e.LastUnix > last {
			last = e.LastUnix
		}
	}

	return last
}

func (f Flake) LastSeen() time.Time { return time.Unix(f.LastUnix(), 0) }

// Detect finds every test with conflicting verdicts at the same fingerprint.
// Only the conflicting fingerprints are reported as evidence: a fingerprint
// where the test always failed is a plain red test, not a flake.
func Detect(observations []Observation) []Flake {
	type key struct{ pkg, test, fp string }
	states := make(map[key]*Evidence)

	for _, o := range observations {
		if o.Package == "" || o.Test == "" || o.Fingerprint == "" {
			continue
		}
		k := key{o.Package, o.Test, o.Fingerprint}
		e := states[k]
		if e == nil {
			e = &Evidence{Fingerprint: o.Fingerprint, FirstUnix: o.Unix, LastUnix: o.Unix}
			states[k] = e
		}
		if o.failed() {
			e.Fails++
			if e.Detail == "" {
				e.Detail = o.Detail
			}
		} else {
			e.Passes++
		}
		if o.Unix < e.FirstUnix {
			e.FirstUnix = o.Unix
		}
		if o.Unix > e.LastUnix {
			e.LastUnix = o.Unix
		}
		e.Sources = appendUnique(e.Sources, o.Source)
	}

	byTest := make(map[[2]string][]Evidence)
	for k, e := range states {
		if e.Passes == 0 || e.Fails == 0 {
			continue
		}
		sort.Strings(e.Sources)
		byTest[[2]string{k.pkg, k.test}] = append(byTest[[2]string{k.pkg, k.test}], *e)
	}

	out := make([]Flake, 0, len(byTest))
	for id, evidence := range byTest {
		sort.Slice(evidence, func(i, j int) bool {
			return evidence[i].LastUnix > evidence[j].LastUnix
		})
		out = append(out, Flake{Package: id[0], Test: id[1], Evidence: evidence})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Package != out[j].Package {
			return out[i].Package < out[j].Package
		}

		return out[i].Test < out[j].Test
	})

	return out
}

func appendUnique(list []string, value string) []string {
	if value == "" {
		return list
	}
	for _, have := range list {
		if have == value {
			return list
		}
	}

	return append(list, value)
}

// ParseVerboseOutput extracts top-level test verdicts from `go test -v` style
// output. Only unindented result lines count: subtest verdicts are indented,
// and a failing subtest already fails its top-level parent, which is the name
// -test.run selects.
func ParseVerboseOutput(output []byte) (passes, fails []string) {
	seenPass := make(map[string]struct{})
	seenFail := make(map[string]struct{})

	for line := range strings.Lines(string(output)) {
		name, failed, ok := parseResultLine(line)
		if !ok {
			continue
		}
		if failed {
			seenFail[name] = struct{}{}
		} else {
			seenPass[name] = struct{}{}
		}
	}

	// A name both passing and failing in one run would be a parse artefact;
	// the failure wins, because losing one is the dangerous direction.
	for name := range seenFail {
		delete(seenPass, name)
	}

	return sortedNames(seenPass), sortedNames(seenFail)
}

func parseResultLine(line string) (name string, failed bool, ok bool) {
	rest, isPass := strings.CutPrefix(line, "--- PASS: ")
	if !isPass {
		rest, ok = strings.CutPrefix(line, "--- FAIL: ")
		if !ok {
			return "", false, false
		}
		failed = true
	}

	name = rest
	if idx := strings.IndexAny(name, " \t("); idx > 0 {
		name = name[:idx]
	}
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsRune(name, '/') {
		return "", false, false
	}

	return name, failed, true
}

func sortedNames(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for name := range set {
		out = append(out, name)
	}
	sort.Strings(out)

	return out
}

func marshalObservation(o Observation) ([]byte, error) {
	return json.Marshal(o)
}

func unmarshalObservation(line []byte) (Observation, error) {
	var o Observation
	err := json.Unmarshal(line, &o)

	return o, err
}
