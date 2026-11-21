package play

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/edi-partner-sim/internal/ladder"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/runlog"
	"github.com/emoss08/trenova/edi-partner-sim/internal/sim"
	"github.com/emoss08/trenova/edi-partner-sim/internal/trenovatest"
)

const fixtures = "testdata/partners"

type rig struct {
	fake    *trenovatest.Fake
	sim     *sim.Server
	simURL  string
	options *Options
	out     *bytes.Buffer
}

func newRig(t *testing.T) *rig {
	t.Helper()
	definition, err := partner.Open(filepath.Join(fixtures, "acme"))
	if err != nil {
		t.Fatalf("open the fixture partner: %v", err)
	}
	var handler http.Handler = http.NotFoundHandler()
	var mu sync.RWMutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		current := handler
		mu.RUnlock()
		current.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	simulator, err := sim.New(sim.Options{
		Partners:    []*partner.Definition{definition},
		IdentityDir: t.TempDir(),
		PublicURL:   server.URL,
		Capacity:    500,
		Seed:        1,
		Logger:      slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatalf("new simulator: %v", err)
	}
	mu.Lock()
	handler = simulator.Handler()
	mu.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	ran := make(chan struct{})
	go func() {
		_ = simulator.Run(ctx)
		close(ran)
	}()
	t.Cleanup(func() {
		cancel()
		<-ran
	})
	fake := trenovatest.New(t, trenovatest.Envelope{
		UsQualifier: "ZZ", UsID: "TRENOVA", UsGroup: "TRENOVA",
		ThemQualifier: "ZZ", ThemID: "ACMESHIP", ThemGroup: "ACMESHIP", SCAC: "TRNV",
	})
	out := new(bytes.Buffer)
	return &rig{
		fake:   fake,
		sim:    simulator,
		simURL: server.URL,
		out:    out,
		options: &Options{
			API:         fake.API(),
			Sim:         server.URL,
			Email:       "admin@trenova.app",
			Password:    "admin123!",
			Root:        fixtures,
			Run:         7,
			IdentityDir: t.TempDir(),
			KeepGoing:   true,
			Out:         out,
		},
	}
}

func (r *rig) run(t *testing.T, scenarios ...string) *Report {
	t.Helper()
	r.options.Scenarios = scenarios
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	report, err := Run(ctx, r.options)
	if err != nil {
		t.Fatalf("run: %v\n%s", err, r.out)
	}
	return report
}

func scenarioOf(t *testing.T, report *Report, slug string) *ScenarioReport {
	t.Helper()
	for _, p := range report.Partners {
		for _, s := range p.Scenarios {
			if s.Slug == slug {
				return s
			}
		}
	}
	t.Fatalf("no scenario %s in the report", slug)
	return nil
}

func TestHappyPathPassesAgainstTheFakeTrenova(t *testing.T) {
	r := newRig(t)
	report := r.run(t, "happy")
	s := scenarioOf(t, report, "happy")
	if s.Status != StatusPassed || !report.Passed() {
		t.Fatalf("happy path %s: %+v\n%+v\n%s", s.Status, s.Failures, s.Steps, r.out)
	}
	if len(s.Conversations) != 1 {
		t.Fatalf("conversations: %d", len(s.Conversations))
	}
	c := s.Conversations[0]
	if !strings.HasPrefix(c.Key, "ACM") || c.Seed != s.Seed+1 || s.Seed != SeedOf(7, 2*40) {
		t.Fatalf("the conversation's key and seed: %s %d", c.Key, c.Seed)
	}
	if c.Milestones.Ack == nil || c.Milestones.Answer == nil || c.Milestones.Status == nil ||
		c.Milestones.Invoice == nil {
		t.Fatalf("milestones: %+v", c.Milestones)
	}
	from := map[string]int{}
	for _, entry := range c.Wire {
		from[entry.From]++
		if entry.X12 == "" {
			t.Fatalf("a wire entry without its X12: %+v", entry)
		}
	}
	if from["partner"] == 0 || from["trenova"] < 5 {
		t.Fatalf("the wire log holds both sides: %v", from)
	}
	if len(c.States) == 0 {
		t.Fatalf("state snapshots: %+v", c.States)
	}
	setup := report.Partners[0].Provision
	if setup.Failure != "" || setup.InboundURL == "" || setup.VersionNumber == 0 {
		t.Fatalf("provisioning: %+v", setup)
	}
}

func TestAHeldChangeIsReleased(t *testing.T) {
	r := newRig(t)
	report := r.run(t, "held")
	if s := scenarioOf(t, report, "held"); s.Status != StatusPassed {
		t.Fatalf("held: %+v\n%s", s.Failures, r.out)
	}
}

func TestABatchIsOneConversationPerTender(t *testing.T) {
	r := newRig(t)
	report := r.run(t, "batch")
	s := scenarioOf(t, report, "batch")
	if s.Status != StatusPassed || len(s.Conversations) != 3 {
		t.Fatalf("batch: %s %+v (%d conversations)\n%s", s.Status, s.Failures,
			len(s.Conversations), r.out)
	}
	keys := map[string]bool{}
	for _, c := range s.Conversations {
		keys[c.Key] = true
	}
	if len(keys) != 3 {
		t.Fatalf("the batch's keys: %v", keys)
	}
}

func TestAcceptingMapsThePartnersCodesOnce(t *testing.T) {
	r := newRig(t)
	first := r.run(t, "batch")
	second := r.run(t, "happy")
	for _, s := range []*ScenarioReport{scenarioOf(t, first, "batch"),
		scenarioOf(t, second, "happy")} {
		if s.Status != StatusPassed {
			t.Fatalf("%s: %+v\n%s", s.Slug, s.Failures, r.out)
		}
	}
	partnerID := second.Partners[0].Provision.PartnerID
	mapped := r.fake.Mapped(partnerID)
	if r.fake.Created("CUSTOMER") != 1 || mapped["ServiceType|DEFAULT"] != "st_2" ||
		mapped["FormulaTemplate|DEFAULT"] != "ft_2" ||
		mapped["Customer|KVW-AP"] != second.Partners[0].Provision.CustomerID {
		t.Fatalf("the partner's mappings: %v (%d customers)", mapped,
			r.fake.Created("CUSTOMER"))
	}
	seen := map[string]bool{}
	for _, location := range r.fake.Records("LOCATION") {
		meta, _ := location["meta"].(map[string]any)
		key := fmt.Sprint(location["label"], "|", meta["postalCode"])
		if seen[key] {
			t.Fatalf("location %s was created twice", key)
		}
		seen[key] = true
	}
	if len(seen) == 0 || r.fake.Created("COMMODITY") == 0 {
		t.Fatalf("no location or commodity was created: %d locations, %d commodities",
			len(seen), r.fake.Created("COMMODITY"))
	}
	for key := range mapped {
		if strings.HasPrefix(key, "Location|") || strings.HasPrefix(key, "Commodity|") {
			continue
		}
		if !slices.Contains([]string{"Customer|KVW-AP", "ServiceType|DEFAULT",
			"FormulaTemplate|DEFAULT"}, key) {
			t.Fatalf("an unexpected mapping %s", key)
		}
	}
}

func TestAnAcceptedTenderThatCannotBeBuiltFailsTheDecision(t *testing.T) {
	r := newRig(t)
	r.fake.BuildFail = "shipmentTypeId: This field is required"
	report := r.run(t, "happy")
	s := scenarioOf(t, report, "happy")
	if s.Status != StatusFailed || len(s.Failures) != 1 ||
		!strings.Contains(s.Failures[0].Message, "shipmentTypeId") ||
		!strings.Contains(s.Steps[s.Failures[0].Step].Text, "decide") {
		t.Fatalf("happy with an unbuildable tender: %s %+v", s.Status, s.Failures)
	}
}

func TestAWrongAnswerFailsWithTheElementItSaw(t *testing.T) {
	r := newRig(t)
	report := r.run(t, "wrong-answer")
	s := scenarioOf(t, report, "wrong-answer")
	if s.Status != StatusFailed || len(s.Failures) != 1 {
		t.Fatalf("wrong-answer: %s %+v", s.Status, s.Failures)
	}
	f := s.Failures[0]
	if f.Step != 2 || f.Source != SourceExpect || f.Rule != "B1-04" ||
		!strings.Contains(f.Message, `"D"`) || !strings.Contains(f.Replay, "--run 7") {
		t.Fatalf("the failure: %+v", f)
	}
	if s.Steps[3].Status != StatusSkipped {
		t.Fatalf("the step after a failure runs: %+v", s.Steps[3])
	}
	if report.Passed() || report.Failures != 1 {
		t.Fatalf("the report passes: %d failures", report.Failures)
	}
}

func TestAWaitEndsAtItsDeadline(t *testing.T) {
	r := newRig(t)
	started := time.Now()
	report := r.run(t, "never-escalates")
	s := scenarioOf(t, report, "never-escalates")
	if s.Status != StatusFailed || !strings.Contains(s.Failures[0].Message, "is Open") {
		t.Fatalf("never-escalates: %s %+v", s.Status, s.Failures)
	}
	if took := time.Since(started); took > 20*time.Second {
		t.Fatalf("a 1s wait took %s", took)
	}
}

func TestTheDeadlineScaleShortensEveryWaitAndEachScenarioIsReportedAsItEnds(t *testing.T) {
	r := newRig(t)
	r.options.Scale = 0.25
	var finished []string
	r.options.OnScenario = func(partner string, s *ScenarioReport) {
		finished = append(finished, partner+"/"+s.Slug+" "+s.Status)
	}
	report := r.run(t, "never-escalates")
	s := scenarioOf(t, report, "never-escalates")
	wait := s.Steps[len(s.Steps)-1]
	if s.Status != StatusFailed || time.Duration(wait.Took) >= time.Second {
		t.Fatalf("a 1s wait at ×0.25: %s after %s", s.Status, wait.Took)
	}
	if len(finished) != 1 || !strings.HasSuffix(finished[0], "never-escalates failed") {
		t.Fatalf("the scenarios reported as they ended: %v", finished)
	}
	if report.Scale != 0.25 || report.Waiting.Expected != 1 ||
		report.Waiting.Failed != wait.Took {
		t.Fatalf("the run's waiting: ×%g %+v", report.Scale, report.Waiting)
	}
}

func TestAPartnerRejectionEscalates(t *testing.T) {
	r := newRig(t)
	report := r.run(t, "rejected")
	if s := scenarioOf(t, report, "rejected"); s.Status != StatusPassed {
		t.Fatalf("rejected: %+v\n%s", s.Failures, r.out)
	}
}

func TestAdvanceRunsTheCLIForTheConversationsFlow(t *testing.T) {
	r := newRig(t)
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args")
	script := filepath.Join(dir, "trenova")
	body := "#!/bin/sh\necho \"$@\" > " + argsFile + "\n" +
		`echo '{"instanceId":"x","status":"Escalated","reason":"needs an answer","sent":[]}'` + "\n"
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	r.options.CLI.Binary, r.options.CLI.Dir = script, dir
	report := r.run(t, "advance")
	s := scenarioOf(t, report, "advance")
	if s.Status != StatusPassed || !strings.Contains(s.Steps[2].Note, "Escalated") {
		t.Fatalf("advance: %s %+v %+v", s.Status, s.Failures, s.Steps)
	}
	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	flows := r.fake.Flows()
	if len(flows) != 1 || !strings.Contains(string(args),
		"edi flows advance --instance "+flows[0].ID+" --by 2h0m0s --json") {
		t.Fatalf("the CLI ran with %q", args)
	}
}

func TestAdvanceWithoutACLIStopsBeforeTheRun(t *testing.T) {
	r := newRig(t)
	r.options.Scenarios = []string{"advance"}
	_, err := Run(context.Background(), r.options)
	if err == nil || !strings.Contains(err.Error(), "--trenova-cli") {
		t.Fatalf("a run that advances without a CLI: %v", err)
	}
}

func TestAPartnerThatCannotCertifyStopsWithWhy(t *testing.T) {
	r := newRig(t)
	r.fake.NotReady = true
	report := r.run(t, "happy")
	p := report.Partners[0]
	if len(p.Failures) != 1 || len(p.Scenarios) != 0 ||
		!strings.Contains(p.Failures[0].Message, "1 of 3 tests failing") {
		t.Fatalf("an uncertifiable partner: %+v", p.Failures)
	}
	if len(p.Provision.FailingTests) != 1 ||
		!strings.Contains(p.Provision.FailingTests[0], "expected held") {
		t.Fatalf("the failing tests: %v", p.Provision.FailingTests)
	}
}

func TestTheReportIsWrittenAsJSON(t *testing.T) {
	r := newRig(t)
	report := r.run(t, "wrong-answer")
	path := filepath.Join(t.TempDir(), "report.json")
	if err := report.WriteJSON(path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Run      uint64 `json:"run"`
		Failures int    `json:"failures"`
		Partners []struct {
			Scenarios []struct {
				Seed          uint64 `json:"seed"`
				Took          float64
				Conversations []struct {
					Wire []struct {
						X12 string `json:"x12"`
					} `json:"wire"`
				} `json:"conversations"`
			} `json:"scenarios"`
		} `json:"partners"`
	}
	if err = sonic.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	scenarios := decoded.Partners[0].Scenarios
	if decoded.Run != 7 || decoded.Failures != 1 || scenarios[0].Seed != SeedOf(7, 6*40) ||
		len(scenarios[0].Conversations[0].Wire) == 0 {
		t.Fatalf("the JSON report: %s", raw)
	}
}

func TestARunPostsItsProgressToTheViewer(t *testing.T) {
	r := newRig(t)
	report := r.run(t, "held", "wrong-answer")
	current, ok := r.sim.Runs().Current()
	if !ok || current.Kind != runlog.KindRun || current.Number != 7 ||
		current.Status != runlog.StatusFailed || current.Tally.Passed != 1 ||
		current.Tally.Failed != 1 {
		t.Fatalf("the run in the viewer: %+v", current)
	}
	held := scenarioOf(t, report, "held")
	key := held.Conversations[0].Key
	detail, err := r.sim.Conversation("acme", key)
	if err != nil || detail.Scenario == nil || detail.Scenario.Slug != "held" {
		t.Fatalf("the conversation: %v %+v", err, detail.Scenario)
	}
	if len(detail.Scenario.Steps) != 7 ||
		detail.Scenario.Steps[0].Text != "send 204 original as t1" ||
		detail.Scenario.Steps[2].Text != "send 204 change of t1, total +12%" {
		t.Fatalf("the steps: %+v", detail.Scenario.Steps)
	}
	marked, labels := 0, []string{}
	for _, row := range detail.Rows {
		if row.Kind == ladder.KindMessage && row.Expect == ladder.ToneOK {
			marked++
		}
		if row.Kind == ladder.KindEvent {
			labels = append(labels, row.Label+"/"+row.Tone+"/"+row.Expect)
		}
	}
	if marked < 4 {
		t.Fatalf("only %d messages carry a mark: %+v", marked, detail.Rows)
	}
	want := []string{"Flow held/hold/ok", "Planner released/trenova/", "Flow open/trenova/ok"}
	if strings.Join(labels, ",") != strings.Join(want, ",") {
		t.Fatalf("events %v, want %v", labels, want)
	}
	if detail.Run.ID != current.ID {
		t.Fatalf("the conversation names run %s, not %s", detail.Run.ID, current.ID)
	}
	if !slices.ContainsFunc(detail.Rows, func(row ladder.Row) bool {
		return row.Set == "204" && row.Title == "Change: rate +12%"
	}) {
		t.Fatalf("no row titles the change: %+v", detail.Rows)
	}
	wrong := scenarioOf(t, report, "wrong-answer")
	failedKey := wrong.Conversations[0].Key
	failed, _ := r.sim.Conversation("acme", failedKey)
	step := failed.Scenario.Steps[2]
	if step.Status != runlog.StatusFailed || !strings.Contains(step.Why, "B1-04") ||
		len(step.Matches) != 1 {
		t.Fatalf("the failing step: %+v", step)
	}
	bad := false
	for _, row := range failed.Rows {
		if row.Set == "990" && row.Expect == ladder.ToneBad {
			bad = true
		}
	}
	if !bad {
		t.Fatalf("the wrong 990 carries no failing mark: %+v", failed.Rows)
	}
}
