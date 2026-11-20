package reply_test

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner/reply"
	"github.com/emoss08/trenova/pkg/lading"
)

const (
	carrierDir    = "../../../partners/carrier"
	conversations = 200
	runAt         = "2026-03-01T09:30-07:00"
)

var update = flag.Bool("update", false, "rewrite the carrier's generated Lading samples")

var mapFor = map[string]string{
	"990": "responseIn",
	"214": "statusIn",
	"210": "invoiceIn",
}

func openCarrier(t *testing.T) *reply.Carrier {
	t.Helper()
	definition, err := partner.Open(carrierDir)
	if err != nil {
		t.Fatalf("open the carrier:\n%v", err)
	}
	carrier, err := reply.Open(definition)
	if err != nil {
		t.Fatalf("open the carrier's replies:\n%v", err)
	}
	return carrier
}

func encodeCarrier(t *testing.T, carrier *reply.Carrier) *lading.Encoded {
	t.Helper()
	files, samples, err := carrier.Definition.LadingProject()
	if err != nil {
		t.Fatal(err)
	}
	project := &lading.Project{}
	for _, file := range files {
		project.Files = append(project.Files, lading.File{Path: file.Path, Source: file.Content})
	}
	for _, sample := range samples {
		project.Samples = append(
			project.Samples,
			lading.Sample{Path: sample.Path, Content: sample.Content},
		)
	}
	encoded, err := lading.Encode(project, &lading.Context{Org: lading.Org{
		Name: "Trenova",
		SCAC: carrier.Definition.Partner.Trenova.SCAC,
		Zone: carrier.Definition.Partner.Zone,
	}})
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestCarrierProjectChecksAndPasses(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	carrier := openCarrier(t)
	project := encodeCarrier(t, carrier)
	check, err := lading.Default().Check(ctx, project)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range check.Files {
		for _, diagnostic := range file.Diagnostics {
			if diagnostic.Severity == lading.SeverityError {
				t.Errorf("%s %v: %s: %s", file.Path, diagnostic.Range, diagnostic.Code,
					diagnostic.Message)
			}
		}
	}
	if check.Errors > 0 {
		t.FailNow()
	}
	assertCarrierIdentity(t, carrier.Definition, check)
	for _, set := range []string{"990", "214", "210"} {
		if _, clause := check.Receiving(set); clause == nil {
			t.Errorf("the project receives no %s", set)
		}
	}
	if _, clause := check.Sending("204", "TenderOffered"); clause == nil {
		t.Error("the project sends no 204 when a load is offered")
	}

	report, err := lading.Default().Test(ctx, project)
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range report.Tests {
		if !result.Passed {
			t.Errorf("%s: %q failed: %+v %+v", result.Path, result.Name, result.Failures,
				result.Counterexample)
		}
	}
	t.Logf("%d Lading tests, readiness %+v", len(report.Tests), report.Readiness)
	if report.Readiness.UncoveredFlowClauses > 0 {
		t.Errorf("%d flow clauses no test reaches", report.Readiness.UncoveredFlowClauses)
	}
	if !report.Readiness.Ready && !t.Failed() {
		t.Errorf("the project is not ready to certify: %+v", report.Readiness)
	}
}

func assertCarrierIdentity(t *testing.T, definition *partner.Definition, check *lading.Check) {
	t.Helper()
	p := definition.Partner
	if check.Partner == nil || check.Partner.Interchange == nil {
		t.Fatal("the project has no partner interchange")
	}
	interchange := check.Partner.Interchange
	got := []string{
		interchange.Them.Qualifier, interchange.Them.ID,
		interchange.Us.Qualifier, interchange.Us.ID,
		interchange.Group.Theirs, interchange.Group.Ours,
		interchange.Version, check.Partner.Zone,
		strings.Join(interchange.Separators, ""),
	}
	separators := p.Envelope.Separators
	written := separators.Element + separators.Segment + separators.Component
	if p.Envelope.Version >= "004020" {
		written += separators.Repetition
	}
	want := []string{
		p.Them.Qualifier, p.Them.ID,
		p.Trenova.Qualifier, p.Trenova.ID,
		p.Them.Group, p.Trenova.Group,
		p.Envelope.Version, p.Zone,
		written,
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("the Lading partner file says %v, partner.yaml says %v", got, want)
			return
		}
	}
	usage := map[string]string{"T": "test", "P": "production"}[p.Envelope.Usage]
	if interchange.Usage != usage {
		t.Errorf("the Lading partner file's usage is %q, partner.yaml says %q",
			interchange.Usage, usage)
	}
}

func writeTender(
	ctx context.Context,
	project *lading.Encoded,
	offer *reply.OfferedLoad,
	control int64,
) (string, error) {
	document, err := offer.JSON()
	if err != nil {
		return "", err
	}
	result, err := lading.Default().Run(ctx, project, &lading.RunRequest{
		Map:   "tenderOut",
		Input: lading.DocumentInput(document),
		Rules: true,
		Now:   runAt,
		Control: lading.Control{
			Interchange: uint64(control),
			Group:       uint64(control),
			Transaction: 1,
		},
	})
	if err != nil {
		return "", err
	}
	if result.Error != nil {
		return "", fmt.Errorf("tenderOut failed: %v", result.Error)
	}
	if result.Output == nil {
		return "", fmt.Errorf("tenderOut wrote nothing for %s", offer.Key)
	}
	if result.Output.Blocked() {
		return "", fmt.Errorf(
			"tenderOut blocked %s: findings %+v validation %+v outcomes %+v",
			offer.Key,
			result.Output.Findings,
			result.Output.Validation,
			result.Output.Outcomes,
		)
	}
	return result.Output.Text, nil
}

func TestCarrierSamplesAreTheBuildersOutput(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	carrier := openCarrier(t)
	specs, err := carrier.SampleSpecs()
	if err != nil {
		t.Fatal(err)
	}
	project := encodeCarrier(t, carrier)
	tender, err := writeTender(ctx, project, reply.Offer(specs.Tender.Seed), specs.Tender.Control)
	if err != nil {
		t.Fatal(err)
	}
	generated := []partner.GeneratedSample{{
		Path:    carrier.TenderSamplePath(specs),
		Content: []byte(tender),
	}}
	replies, err := carrier.GenerateReplies(specs, []byte(tender))
	if err != nil {
		t.Fatal(err)
	}
	generated = append(generated, replies...)
	for _, sample := range generated {
		if *update {
			if err := os.WriteFile(sample.Path, sample.Content, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		onDisk, err := os.ReadFile(sample.Path)
		if err != nil {
			t.Fatalf(
				"%v: run go test ./internal/partner/reply -run TestCarrierSamples -update",
				err,
			)
		}
		if !bytes.Equal(onDisk, sample.Content) {
			t.Errorf(
				"%s differs from what tenderOut and the carrier's builders write; run go test ./internal/partner/reply -run TestCarrierSamples -update",
				filepath.Base(sample.Path),
			)
		}
	}
}

func TestCarrierRepliesAreDeterministic(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	carrier := openCarrier(t)
	project := encodeCarrier(t, carrier)
	raw, err := writeTender(ctx, project, reply.Offer(41), 41)
	if err != nil {
		t.Fatal(err)
	}
	tenders, _, err := carrier.Receive([]byte(raw))
	if err != nil || len(tenders) != 1 {
		t.Fatalf("read %d tenders: %v", len(tenders), err)
	}
	scenario := reply.Scenario{Seed: 41, Delay: 0.5, DeltaCents: 12_500, Stray: true}
	first, err := carrier.Converse(tenders[0], scenario)
	if err != nil {
		t.Fatal(err)
	}
	second, err := carrier.Converse(tenders[0], scenario)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != len(second) {
		t.Fatalf("%d replies, then %d", len(first), len(second))
	}
	for index := range first {
		a := writeReply(t, carrier, first[index], int64(index+1))
		b := writeReply(t, carrier, second[index], int64(index+1))
		if !bytes.Equal(a, b) {
			t.Errorf("reply %d differs between two runs:\n%s\n%s", index, a, b)
		}
	}
}

func writeReply(t *testing.T, carrier *reply.Carrier, r reply.Reply, control int64) []byte {
	t.Helper()
	written, err := carrier.Write(r, partner.Controls{
		Interchange: partner.NewCounter(control),
		Group:       partner.NewCounter(control),
	})
	if err != nil {
		t.Fatal(err)
	}
	return written.Bytes
}

type tally struct {
	accepted, declined, statuses, delays, held, rejected int
	matching, disagreeing                                int
}

func TestCarrierConversationsReadThroughTheProject(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	carrier := openCarrier(t)
	project := encodeCarrier(t, carrier)
	var counts tally
	for seed := uint64(1); seed <= conversations; seed++ {
		offer := reply.Offer(seed)
		raw, err := writeTender(ctx, project, offer, int64(seed))
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		tenders, judgement, err := carrier.Receive([]byte(raw))
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		if !judgement.Accepted() {
			t.Errorf("seed %d: the carrier's guide rejects the tender: %s\n%s", seed,
				findings(judgement), raw)
			continue
		}
		tender := tenders[0]
		state, err := offeredState(ctx, project, offer)
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		if tender.Key != offer.Key || len(tender.Stops) != len(offer.Stops) {
			t.Errorf("seed %d: read %s with %d stops from %s with %d", seed, tender.Key,
				len(tender.Stops), offer.Key, len(offer.Stops))
		}
		scenario := reply.DrawScenario(seed)
		replies, err := carrier.Converse(tender, scenario)
		if err != nil {
			t.Fatalf("seed %d %s: %v", seed, scenario, err)
		}
		controls := partner.Controls{
			Interchange: partner.NewCounter(int64(seed) * 100),
			Group:       partner.NewCounter(int64(seed) * 100),
		}
		for _, r := range replies {
			written, err := carrier.Write(r, controls)
			if err != nil {
				t.Fatalf("seed %d: %v", seed, err)
			}
			got, document, problem := readReply(
				ctx,
				project,
				r.Set,
				written.Bytes,
				tender.Key,
				state,
			)
			if problem != "" {
				t.Errorf("seed %d %s %s: %s\n%s", seed, scenario, r.Set, problem, written.Bytes)
				continue
			}
			if got != r.Want {
				t.Errorf("seed %d %s %s: the project %s it, want %s (%s)\n%s", seed, scenario,
					r.Set, got, r.Want, r.Note, written.Bytes)
				continue
			}
			if r.Want != reply.OutcomeRejected {
				if key, _ := document["key"].(string); key != tender.Key {
					t.Errorf("seed %d %s: read key %q, tendered %q", seed, r.Set, key, tender.Key)
				}
			}
			if problem := counts.add(r, tender, scenario, document); problem != "" {
				t.Errorf("seed %d %s: %s", seed, scenario, problem)
			}
		}
	}
	t.Logf("%+v", counts)
	if counts.accepted == 0 || counts.declined == 0 || counts.delays == 0 ||
		counts.held == 0 || counts.rejected == 0 || counts.matching == 0 ||
		counts.disagreeing == 0 {
		t.Errorf("the conversations miss a case: %+v", counts)
	}
}

func (c *tally) add(
	r reply.Reply,
	tender *reply.Tender,
	scenario reply.Scenario,
	document map[string]any,
) string {
	switch r.Want {
	case reply.OutcomeHeld:
		c.held++
	case reply.OutcomeRejected:
		c.rejected++
	}
	switch r.Set {
	case "990":
		if document["decision"] == "Accept" {
			c.accepted++
		} else {
			c.declined++
		}
	case "214":
		c.statuses++
		if reply.StatusCode(r) == reply.StatusDelayed {
			c.delays++
		}
	case "210":
		if scenario.Misfooted {
			return ""
		}
		billed := invoiceCents(document)
		if billed-tender.TotalCents != scenario.DeltaCents {
			return fmt.Sprintf("the invoice reads %d cents against a tendered %d, want %+d",
				billed, tender.TotalCents, scenario.DeltaCents)
		}
		if scenario.DeltaCents == 0 {
			c.matching++
		} else {
			c.disagreeing++
		}
	}
	return ""
}

func invoiceCents(document map[string]any) int64 {
	total, _ := document["total"].(map[string]any)
	amount := fmt.Sprint(total["amount"])
	whole, fraction, _ := strings.Cut(amount, ".")
	fraction = (fraction + "00")[:2]
	dollars, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return -1
	}
	cents, err := strconv.ParseInt(fraction, 10, 64)
	if err != nil {
		return -1
	}
	return dollars*100 + cents
}

func readReply(
	ctx context.Context,
	project *lading.Encoded,
	set string,
	x12 []byte,
	tendered string,
	state []byte,
) (reply.Outcome, map[string]any, string) {
	result, err := lading.Default().Run(ctx, project, &lading.RunRequest{
		Map:   mapFor[set],
		Input: lading.X12Input(string(x12)),
		Rules: true,
		Now:   runAt,
	})
	if err != nil {
		return "", nil, err.Error()
	}
	if result.Error != nil {
		return "", nil, fmt.Sprintf("the run failed: %v", result.Error)
	}
	if len(result.Needs) > 0 {
		return "", nil, fmt.Sprintf("the run asks for lookups: %+v", result.Needs)
	}
	if len(result.Transactions) != 1 {
		return "", nil, fmt.Sprintf("read %d transactions, want 1", len(result.Transactions))
	}
	transaction := result.Transactions[0]
	var problems []string
	for _, finding := range transaction.Findings {
		if finding.IsError() {
			problems = append(problems, fmt.Sprintf("finding %+v", finding))
		}
	}
	if len(problems) > 0 {
		return "", nil, strings.Join(problems, "; ")
	}
	document := map[string]any{}
	if len(transaction.Document) > 0 {
		if err := sonic.Unmarshal(transaction.Document, &document); err != nil {
			return "", nil, err.Error()
		}
	}
	if transaction.Rejected() {
		return reply.OutcomeRejected, document, ""
	}
	if key, _ := document["key"].(string); key != tendered {
		state = []byte(`{}`)
	}
	flowed, problem := flowOutcome(ctx, project, set, &transaction, state)
	if problem != "" {
		return "", nil, problem
	}
	if flowed == reply.OutcomeRejected {
		return reply.OutcomeRejected, document, ""
	}
	for _, outcome := range transaction.Outcomes {
		if outcome.Kind == lading.OutcomeHold {
			return reply.OutcomeHeld, document, ""
		}
	}
	return flowed, document, ""
}

func findings(judgement *oracle.Judgement) string {
	var out []string
	for _, finding := range judgement.Findings {
		out = append(out, finding.Message)
	}
	for _, report := range judgement.Reports() {
		for _, finding := range report.Findings {
			out = append(out, fmt.Sprintf("%s%02d %q: %s (%s)", finding.SegmentID,
				finding.ElementPos, finding.Value, finding.Message, finding.Rule))
		}
	}
	return strings.Join(out, "\n")
}

func offeredState(
	ctx context.Context,
	project *lading.Encoded,
	offer *reply.OfferedLoad,
) ([]byte, error) {
	document, err := offer.JSON()
	if err != nil {
		return nil, err
	}
	step, err := stepFlow(ctx, project, []byte(`{}`), lading.FlowEvent{
		Kind:     lading.EventTrenova,
		Event:    "TenderOffered",
		Document: document,
	})
	if err != nil {
		return nil, err
	}
	state := step.State
	for _, action := range step.Actions {
		if action.Kind != lading.ActionSend {
			continue
		}
		sent, err := stepFlow(ctx, project, state, lading.FlowEvent{
			Kind:    lading.EventSent,
			ID:      action.ID,
			Control: "0001",
		})
		if err != nil {
			return nil, err
		}
		state = sent.State
	}
	return state, nil
}

func stepFlow(
	ctx context.Context,
	project *lading.Encoded,
	state []byte,
	event lading.FlowEvent,
) (*lading.FlowStep, error) {
	step, err := lading.Default().FlowStep(ctx, project, &lading.FlowRequest{
		Flow:  "carrierTender",
		State: state,
		Event: event,
		Now:   runAt,
	})
	if err != nil {
		return nil, err
	}
	if step.Error != nil {
		return nil, fmt.Errorf("the flow step failed: %v", step.Error)
	}
	return step, nil
}

func flowOutcome(
	ctx context.Context,
	project *lading.Encoded,
	set string,
	transaction *lading.TransactionResult,
	state []byte,
) (reply.Outcome, string) {
	step, err := stepFlow(ctx, project, state, lading.FlowEvent{
		Kind:     lading.EventReceived,
		Set:      set,
		Document: transaction.Document,
	})
	if err != nil {
		return "", err.Error()
	}
	outcome := reply.OutcomeApplied
	for _, action := range step.Actions {
		switch {
		case action.Kind == lading.ActionAcknowledge && action.Rejected:
			return reply.OutcomeRejected, ""
		case action.Kind == lading.ActionHold:
			outcome = reply.OutcomeHeld
		}
	}
	return outcome, ""
}
