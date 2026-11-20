package partner_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/pkg/lading"
)

const generatedTenders = 200

func encodeProject(t *testing.T, definition *partner.Definition) *lading.Encoded {
	t.Helper()
	files, samples, err := definition.LadingProject()
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
		SCAC: definition.Partner.Trenova.SCAC,
		Zone: definition.Partner.Zone,
	}})
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestLadingProjectsCheckAndPass(t *testing.T) {
	t.Parallel()

	for _, definition := range openAll(t) {
		t.Run(definition.Name(), func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			project := encodeProject(t, definition)

			check, err := lading.Default().Check(ctx, project)
			if err != nil {
				t.Fatal(err)
			}
			for _, file := range check.Files {
				for _, diagnostic := range file.Diagnostics {
					if diagnostic.Severity == lading.SeverityError {
						t.Errorf(
							"%s %v: %s: %s",
							file.Path,
							diagnostic.Range,
							diagnostic.Code,
							diagnostic.Message,
						)
					}
				}
			}
			if check.Errors > 0 {
				t.FailNow()
			}
			assertIdentity(t, definition, check)

			report, err := lading.Default().Test(ctx, project)
			if err != nil {
				t.Fatal(err)
			}
			for _, result := range report.Tests {
				if !result.Passed {
					t.Errorf(
						"%s: %q failed: %+v %+v",
						result.Path,
						result.Name,
						result.Failures,
						result.Counterexample,
					)
				}
			}
			t.Logf("%d Lading tests, readiness %+v", len(report.Tests), report.Readiness)
			if report.Readiness.UncoveredFlowClauses > 0 {
				t.Errorf("%d flow clauses no test reaches", report.Readiness.UncoveredFlowClauses)
			}
			if !report.Readiness.Ready && !t.Failed() {
				t.Errorf("the project is not ready to certify: %+v", report.Readiness)
			}
		})
	}
}

func assertIdentity(t *testing.T, definition *partner.Definition, check *lading.Check) {
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
	}
	want := []string{
		p.Them.Qualifier, p.Them.ID,
		p.Trenova.Qualifier, p.Trenova.ID,
		p.Them.Group, p.Trenova.Group,
		p.Envelope.Version, p.Zone,
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("the Lading partner file says %v, partner.yaml says %v", got, want)
			return
		}
	}
	separators := p.Envelope.Separators
	wantSeparators := []string{
		separators.Element,
		separators.Segment,
		separators.Component,
		separators.Repetition,
	}
	written := len(wantSeparators)
	if p.Envelope.Version < "004020" {
		written = 3
	}
	if len(interchange.Separators) != written {
		t.Errorf("the Lading partner file names separators %q", interchange.Separators)
	}
	for index, separator := range interchange.Separators {
		if index < len(wantSeparators) && wantSeparators[index] != "" &&
			separator != wantSeparators[index] {
			t.Errorf(
				"the Lading partner file's separators are %q, partner.yaml says %q",
				interchange.Separators,
				wantSeparators,
			)
			break
		}
	}
	terminator := string(p.Envelope.Terminator)
	if terminator == "" {
		terminator = string(partner.TerminatorNone)
	}
	if !strings.EqualFold(string(interchange.LineEnding), terminator) {
		t.Errorf(
			"the Lading partner file's line ending is %q, partner.yaml's terminator is %q",
			interchange.LineEnding,
			terminator,
		)
	}
	usage := map[string]string{"T": "test", "P": "production"}[p.Envelope.Usage]
	if interchange.Usage != usage {
		t.Errorf(
			"the Lading partner file's usage is %q, partner.yaml says %q",
			interchange.Usage,
			usage,
		)
	}
}

func TestGeneratedTendersReadWithoutRejects(t *testing.T) {
	t.Parallel()

	for _, definition := range openAll(t) {
		template, ok := definition.Templates["204"]
		if !ok {
			continue
		}
		t.Run(definition.Name(), func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			project := encodeProject(t, definition)
			check, err := lading.Default().Check(ctx, project)
			if err != nil {
				t.Fatal(err)
			}
			_, receive := check.Receiving("204")
			if receive == nil {
				t.Fatal("the project receives no 204")
			}
			kinds := make([]partner.Kind, 0, len(template.Purposes))
			for _, kind := range partner.Kinds {
				if _, ok := template.Purposes[kind]; ok {
					kinds = append(kinds, kind)
				}
			}
			rng := rand.New(rand.NewPCG(2026, 11))
			batch := uint64(min(definition.Partner.Batching.TransactionsPerGroup, 5))
			for start := uint64(1); start <= generatedTenders; start += batch {
				var sends []partner.Send
				for seed := start; seed < start+batch && seed <= generatedTenders; seed++ {
					sends = append(sends, partner.Send{
						Seed:      seed,
						Variation: randomVariation(rng, kinds[int(seed)%len(kinds)]),
					})
				}
				controls := partner.Controls{
					Interchange: partner.NewCounter(int64(start)),
					Group:       partner.NewCounter(int64(start)),
				}
				written, _, err := definition.Batch("204", sends, controls)
				if err != nil {
					t.Fatalf("seeds %d..: %v", start, err)
				}
				for _, interchange := range written {
					count := 0
					for _, group := range interchange.Groups {
						count += len(group.Transactions)
					}
					problem := readTenders(ctx, project, receive.Map, interchange.Bytes, count)
					if problem != "" {
						t.Errorf("seeds %d..: %s\n%s", start, problem, interchange.Bytes)
					}
				}
			}
		})
	}
}

func randomVariation(rng *rand.Rand, kind partner.Kind) partner.Variation {
	variation := partner.Variation{Kind: kind, Sparse: rng.IntN(4) == 0}
	if kind != partner.KindChange && kind != partner.KindReplace {
		return variation
	}
	switch rng.IntN(4) {
	case 0:
		variation.Delta.RatePercent = float64(rng.IntN(31) - 10)
	case 1:
		variation.Delta.AddStop = true
	case 2:
		variation.Delta.Move = &partner.Move{By: time.Duration(1+rng.IntN(24)) * time.Hour}
	}
	return variation
}

func readTender(ctx context.Context, project *lading.Encoded, mapName string, x12 []byte) string {
	return readTenders(ctx, project, mapName, x12, 1)
}

func runTenders(
	ctx context.Context,
	project *lading.Encoded,
	mapName string,
	x12 []byte,
) (*lading.RunResult, error) {
	request := &lading.RunRequest{Map: mapName, Input: lading.X12Input(string(x12)), Rules: true}
	result, err := lading.Default().Run(ctx, project, request)
	if err != nil || len(result.Needs) == 0 {
		return result, err
	}
	for _, need := range result.Needs {
		request.Lookups = append(request.Lookups, lading.LookupAnswer{
			Kind:  need.Kind,
			Code:  need.Code,
			Found: &lading.Found{ID: "id-" + need.Code, Label: need.Code},
		})
	}
	return lading.Default().Run(ctx, project, request)
}

func readTenders(
	ctx context.Context,
	project *lading.Encoded,
	mapName string,
	x12 []byte,
	want int,
) string {
	result, err := runTenders(ctx, project, mapName, x12)
	if err != nil {
		return err.Error()
	}
	if result.Error != nil {
		return fmt.Sprintf("the run failed: %v", result.Error)
	}
	if len(result.Transactions) != want {
		return fmt.Sprintf("read %d transactions, want %d", len(result.Transactions), want)
	}
	var problems []string
	for index, transaction := range result.Transactions {
		for _, finding := range transaction.Findings {
			if finding.IsError() {
				problems = append(problems, fmt.Sprintf("[%d] finding %+v", index, finding))
			}
		}
		for _, outcome := range transaction.Outcomes {
			if outcome.Kind == lading.OutcomeReject {
				problems = append(
					problems,
					fmt.Sprintf("[%d] rejected: %s", index, outcome.Message),
				)
			}
		}
		if transaction.Rejected() && len(problems) == 0 {
			problems = append(problems, fmt.Sprintf("[%d] the transaction was rejected", index))
		}
	}
	return strings.Join(problems, "; ")
}

func TestABrokenTenderIsCaught(t *testing.T) {
	t.Parallel()

	definition, err := partner.Open(partnersRoot + "/manufacturer")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	project := encodeProject(t, definition)
	controls := partner.Controls{Interchange: partner.NewCounter(9), Group: partner.NewCounter(9)}
	written, load, err := definition.Interchange(
		"204",
		37,
		partner.Variation{Kind: partner.KindOriginal},
		controls,
	)
	if err != nil {
		t.Fatal(err)
	}
	broken := strings.Replace(string(written.Bytes), "**"+load.Number+"**", "****", 1)
	if problem := readTender(ctx, project, "tenderIn", []byte(broken)); problem == "" {
		t.Fatalf("a tender with no load number read clean:\n%s", broken)
	}
}
