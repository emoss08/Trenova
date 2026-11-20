package conform

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
)

const partnersRoot = "../../partners"

func copyDir(t *testing.T, from, to string) {
	t.Helper()
	err := filepath.WalkDir(from, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}
		target := filepath.Join(to, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func edit(t *testing.T, path, old, replacement string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, old) {
		t.Fatalf("%s does not contain %q", path, old)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(text, old, replacement, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
}

func run(t *testing.T, options Options) *Report {
	t.Helper()
	report, err := Run(context.Background(), &options)
	if err != nil {
		t.Fatal(err)
	}
	return report
}

func failuresOf(report *Report, step string) []Failure {
	var out []Failure
	for _, p := range report.Partners {
		for _, c := range p.Cases {
			for _, f := range c.Failures {
				if f.Step == step {
					out = append(out, f)
				}
			}
		}
	}
	return out
}

func TestABrokenResponseMapIsNamedByTheOracle(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dir := filepath.Join(root, "manufacturer")
	copyDir(t, filepath.Join(partnersRoot, "manufacturer"), dir)
	edit(t, filepath.Join(dir, "lading/maps/response-out.lading"),
		"  04 <- decision via decisions\n", "")
	edit(t, filepath.Join(dir, "lading/guides/kestrel-990.lading"),
		"04 required codes", "04 codes")

	report := run(t, Options{Root: root, Seed: 7, Count: 3, Workers: 1})
	if report.Passed() {
		t.Fatal("conform passed a 990 with no B1-04")
	}
	failures := failuresOf(report, "990 accept")
	if len(failures) == 0 {
		t.Fatalf("no failure on the 990; report: %+v", report.Partners[0].Failures)
	}
	named := false
	for _, f := range failures {
		t.Logf("%s %s [%s]: %s", f.Source, f.Rule, f.X12Code, f.Message)
		if f.Source == SourceOracle && f.Rule == "B1-04 usage M" && f.X12Code == "AK403 1" &&
			strings.Contains(f.Message, "mandatory element B1-04 is missing") &&
			strings.HasPrefix(f.Replay, "manufacturer:") {
			named = true
		}
	}
	if !named {
		t.Fatalf("the oracle did not name B1-04 mandatory missing: %+v", failures)
	}
}

func TestTheOracleCatchesWhatTheCrateLetsThrough(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dir := filepath.Join(root, "manufacturer")
	copyDir(t, filepath.Join(partnersRoot, "manufacturer"), dir)
	edit(t, filepath.Join(dir, "lading/maps/response-out.lading"),
		"  03 <- respondedAt\n  04 <- decision via decisions\n", "")
	edit(t, filepath.Join(dir, "lading/guides/kestrel-990.lading"),
		"03 required; 04 required codes", "04 codes")

	report := run(t, Options{Root: root, Seed: 7, Count: 2, Workers: 1})
	failures := failuresOf(report, "990 accept")
	rules := map[string]bool{}
	for _, f := range failures {
		t.Logf("%s %s [%s]: %s", f.Source, f.Rule, f.X12Code, f.Message)
		rules[f.Source+" "+f.Rule] = true
		if f.Source == SourceLading {
			t.Errorf("the crate caught it too, so this proves nothing: %s", f.Message)
		}
	}
	for _, want := range []string{"oracle B1-03 usage M", "oracle B1-04 usage M"} {
		if !rules[want] {
			t.Errorf("the oracle did not name %s", want)
		}
	}
}

func TestTheManufacturerConforms(t *testing.T) {
	t.Parallel()

	report := run(t, Options{Root: partnersRoot, Partners: []string{"manufacturer"}, Count: 6})
	p := report.Partners[0]
	if len(p.Failures) > 0 {
		t.Fatalf("the project fails: %+v", p.Failures)
	}
	if len(p.Cases) != 6 {
		t.Fatalf("ran %d cases, want 6", len(p.Cases))
	}
	kinds := map[string]bool{}
	for _, c := range p.Cases {
		kinds[strings.Fields(c.Variation)[0]] = true
		for _, f := range c.Failures {
			t.Errorf("seed %d %s: %s %s: %s", c.Seed, f.Step, f.Source, f.Rule, f.Message)
		}
	}
	for _, kind := range partner.Kinds {
		if !kinds[string(kind)] {
			t.Errorf("no case tendered a %s", kind)
		}
	}
}

func TestTheSameSeedReplaysTheSameCase(t *testing.T) {
	t.Parallel()

	def, err := partner.Open(filepath.Join(partnersRoot, "manufacturer"))
	if err != nil {
		t.Fatal(err)
	}
	for seed := uint64(1); seed < 40; seed++ {
		a, _ := Variation(def, seed)
		b, _ := Variation(def, seed)
		if a.String() != b.String() {
			t.Fatalf("seed %d: %s then %s", seed, a, b)
		}
	}
}

func TestAStatusTheMapDeclaresNotSentGoesNowhere(t *testing.T) {
	t.Parallel()

	report := run(t, Options{Root: partnersRoot, Partners: []string{"legacy"}, Count: 1})
	c := report.Partners[0].Cases[0]
	if len(c.Failures) > 0 {
		t.Fatalf("seed %d failed: %+v", c.Seed, c.Failures)
	}
	notSent := 0
	for _, step := range c.Steps {
		if step.Status != StatusNotSent {
			continue
		}
		notSent++
		if step.X12 != "" || step.Ack != nil || len(step.Oracle) > 0 {
			t.Errorf("%s was not sent yet carries wire, an ack or findings: %+v", step.Name, step)
		}
	}
	if notSent == 0 {
		t.Fatalf("seed %d: no 214 the legacy map declares not sent", c.Seed)
	}
}
