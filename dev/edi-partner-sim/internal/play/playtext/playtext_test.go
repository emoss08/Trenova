package playtext_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/play"
	"github.com/emoss08/trenova/edi-partner-sim/internal/play/playtext"
	"github.com/emoss08/trenova/edi-partner-sim/internal/provision"
)

func TestTheSummaryNamesFailuresWaitsAndTheScale(t *testing.T) {
	t.Parallel()

	failed := &play.ScenarioReport{
		Slug:   "happy-path",
		Seed:   12_040,
		Status: play.StatusFailed,
		Took:   play.Duration(3 * time.Minute),
		Steps: []*play.StepReport{
			{Kind: "send", Text: "send 204 original as t1", Status: play.StatusPassed},
			{
				Kind:   "expect",
				Text:   "expect flow t1 Open",
				Status: play.StatusFailed,
				Took:   play.Duration(54 * time.Second),
			},
		},
		Failures: []play.Failure{{
			Step: 1, Source: play.SourceExpect, Rule: "flow", Message: "the flow is Held",
		}},
		Replay: "edi-sim run --partner retailer --scenario happy-path --run 12",
	}
	report := &play.Report{
		Run:   12,
		Scale: 0.3,
		Partners: []*play.PartnerReport{{
			Name:      "retailer",
			Role:      "shipper",
			Provision: &provision.Result{Code: "EDISIM-RETAILER", FailingTests: []string{"x"}},
			Scenarios: []*play.ScenarioReport{failed},
		}},
		Failures: 1,
		Waiting: play.Waiting{
			Expectations: play.Duration(54 * time.Second),
			Expected:     1,
			Failed:       play.Duration(54 * time.Second),
		},
	}
	var summary bytes.Buffer
	playtext.Summary(&summary, report)
	for _, want := range []string{
		"failing test: x",
		"FAIL happy-path",
		"step 2 expect flow t1 Open",
		"replay: edi-sim run --partner retailer",
		"waited 54s on 1 expectations",
		"54s of it in steps that failed",
		"every deadline ×0.3",
		"FAIL: 1 failures",
	} {
		if !strings.Contains(summary.String(), want) {
			t.Errorf("the summary has no %q:\n%s", want, summary.String())
		}
	}

	var line bytes.Buffer
	playtext.Line(&line, failed)
	if !strings.HasPrefix(line.String(), "  FAIL happy-path") ||
		!strings.Contains(line.String(), "seed 12040") {
		t.Errorf("the scenario's line:\n%s", line.String())
	}
}
