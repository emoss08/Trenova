package narrator

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/insightservice/detector"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubCompletion struct {
	result   *services.StructuredCompletionResult
	err      error
	calls    int
	lastReq  *services.StructuredCompletionRequest
	requests []*services.StructuredCompletionRequest
}

func (c *stubCompletion) CompleteStructured(
	_ context.Context,
	req *services.StructuredCompletionRequest,
) (*services.StructuredCompletionResult, error) {
	c.calls++
	c.lastReq = req
	c.requests = append(c.requests, req)

	return c.result, c.err
}

func (c *stubCompletion) Diagnose(
	context.Context,
	*services.DiagnoseRequest,
) (*services.DiagnoseResult, error) {
	return nil, errors.New("not used")
}

func (c *stubCompletion) CompleteChat(
	context.Context,
	*services.ChatCompletionRequest,
) (*services.ChatCompletionResult, error) {
	return nil, errors.New("not used")
}

func (c *stubCompletion) StreamChat(
	context.Context,
	*services.ChatCompletionRequest,
	services.ChatStreamSink,
) (*services.ChatCompletionResult, error) {
	return nil, errors.New("not used")
}

func newNarrator(completion services.CompletionService) *Service {
	return &Service{l: zap.NewNop(), completion: completion}
}

func testFinding() detector.Finding {
	baseline := decimal.NewFromFloat(93.1)

	return detector.Finding{
		DedupeKey: "ontime-decline:cus_1",
		Subject:   "Acme Foods",
		Headline:  "On-time delivery for Acme Foods is 82.4%, down from 93.1%",
		Severity:  insight.SeverityWarning,
		Metrics: []insight.Metric{{
			Key:           "onTimePercent",
			Label:         "On-time delivery",
			Value:         decimal.NewFromFloat(82.4),
			Unit:          insight.UnitPercent,
			Direction:     insight.DirectionLowerIsWorse,
			Baseline:      &baseline,
			BaselineLabel: "prior 30 days",
		}},
	}
}

func completionReturning(insights ...map[string]string) *stubCompletion {
	payload := narrationEnvelope{Insights: make([]narratedFinding, 0, len(insights))}
	for _, item := range insights {
		payload.Insights = append(payload.Insights, narratedFinding{
			DedupeKey:      item["dedupeKey"],
			Headline:       item["headline"],
			Narrative:      item["narrative"],
			Recommendation: item["recommendation"],
		})
	}

	encoded, err := sonic.MarshalString(payload)
	if err != nil {
		panic(err)
	}

	return &stubCompletion{result: &services.StructuredCompletionResult{
		Text:            encoded,
		ModelIdentifier: "test-model",
		ProviderID:      pulid.MustNew("aip_"),
	}}
}

func TestNarrate_UsesTheModelsWordingWhenItCitesComputedFigures(t *testing.T) {
	t.Parallel()

	finding := testFinding()
	completion := completionReturning(map[string]string{
		"dedupeKey":      finding.DedupeKey,
		"headline":       "Acme Foods deliveries are arriving late more often",
		"narrative":      "On-time delivery sits at 82.4%, down from 93.1% the prior month.",
		"recommendation": "Review the appointment windows on this customer's lanes.",
	})

	result := newNarrator(completion).Narrate(t.Context(), &NarrateRequest{
		Findings:   []detector.Finding{finding},
		WindowDays: 30,
	})

	narration := result[finding.DedupeKey]
	assert.True(t, narration.Narrated)
	assert.Equal(t, "Acme Foods deliveries are arriving late more often", narration.Headline)
	assert.Contains(t, narration.Narrative, "82.4%")
	assert.Equal(t, "test-model", narration.ModelIdentifier)
}

// The whole reason narration is allowed to run at all. A model that invents a
// figure loses its wording; the finding keeps the number it computed.
func TestNarrate_KeepsTheDetectorWordingWhenTheModelInventsANumber(t *testing.T) {
	t.Parallel()

	finding := testFinding()
	completion := completionReturning(map[string]string{
		"dedupeKey":      finding.DedupeKey,
		"headline":       "Acme Foods service has collapsed",
		"narrative":      "On-time delivery has fallen to 41%, costing an estimated $96,000.",
		"recommendation": "Escalate immediately.",
	})

	result := newNarrator(completion).Narrate(t.Context(), &NarrateRequest{
		Findings:   []detector.Finding{finding},
		WindowDays: 30,
	})

	narration := result[finding.DedupeKey]
	assert.False(t, narration.Narrated)
	assert.Equal(t, finding.Headline, narration.Headline)
	assert.Empty(t, narration.Narrative)
	assert.Empty(t, narration.ModelIdentifier)
}

// One bad narration must not cost the others theirs, or a single hallucination
// would blank a whole refresh.
func TestNarrate_RejectsOnlyTheFindingThatWasFabricated(t *testing.T) {
	t.Parallel()

	honest := testFinding()
	fabricated := testFinding()
	fabricated.DedupeKey = "ontime-decline:cus_2"
	fabricated.Subject = "Borden Freight"

	completion := completionReturning(
		map[string]string{
			"dedupeKey": honest.DedupeKey,
			"headline":  "Acme Foods is arriving late more often",
			"narrative": "On-time delivery is 82.4%.",
		},
		map[string]string{
			"dedupeKey": fabricated.DedupeKey,
			"headline":  "Borden Freight is costing $250,000 a quarter",
			"narrative": "Losses have reached $250,000.",
		},
	)

	result := newNarrator(completion).Narrate(t.Context(), &NarrateRequest{
		Findings:   []detector.Finding{honest, fabricated},
		WindowDays: 30,
	})

	assert.True(t, result[honest.DedupeKey].Narrated)
	assert.False(t, result[fabricated.DedupeKey].Narrated)
	assert.Equal(t, fabricated.Headline, result[fabricated.DedupeKey].Headline)
}

// An insight has to survive its AI being switched off. This is the case a
// deployment with no provider configured hits on every refresh.
func TestNarrate_FallsBackWhenNoProviderIsConfigured(t *testing.T) {
	t.Parallel()

	finding := testFinding()
	completion := &stubCompletion{err: services.ErrNoProviderConfigured}

	result := newNarrator(completion).Narrate(t.Context(), &NarrateRequest{
		Findings: []detector.Finding{finding},
	})

	narration := result[finding.DedupeKey]
	assert.False(t, narration.Narrated)
	assert.Equal(t, finding.Headline, narration.Headline)
}

func TestNarrate_FallsBackWhenTheProviderFails(t *testing.T) {
	t.Parallel()

	finding := testFinding()
	completion := &stubCompletion{err: errors.New("upstream timeout")}

	result := newNarrator(completion).Narrate(t.Context(), &NarrateRequest{
		Findings: []detector.Finding{finding},
	})

	assert.False(t, result[finding.DedupeKey].Narrated)
	assert.Equal(t, finding.Headline, result[finding.DedupeKey].Headline)
}

func TestNarrate_FallsBackWhenTheModelReturnsUnparseableOutput(t *testing.T) {
	t.Parallel()

	finding := testFinding()
	completion := &stubCompletion{result: &services.StructuredCompletionResult{
		Text:            "I'm afraid I can't help with that.",
		ModelIdentifier: "test-model",
	}}

	result := newNarrator(completion).Narrate(t.Context(), &NarrateRequest{
		Findings: []detector.Finding{finding},
	})

	assert.False(t, result[finding.DedupeKey].Narrated)
	assert.Equal(t, finding.Headline, result[finding.DedupeKey].Headline)
}

// A model that answers about a finding nobody asked about must not be able to
// put a card on someone's home screen.
func TestNarrate_IgnoresNarrationForAFindingThatWasNotSent(t *testing.T) {
	t.Parallel()

	finding := testFinding()
	completion := completionReturning(map[string]string{
		"dedupeKey": "something-nobody-detected",
		"headline":  "A problem that does not exist",
	})

	result := newNarrator(completion).Narrate(t.Context(), &NarrateRequest{
		Findings: []detector.Finding{finding},
	})

	require.Len(t, result, 1)
	assert.False(t, result[finding.DedupeKey].Narrated)
	_, invented := result["something-nobody-detected"]
	assert.False(t, invented)
}

// Every finding appears in the result whatever happened, so a caller never has
// to tell "no narration" apart from "finding lost".
func TestNarrate_ReturnsAnEntryForEveryFinding(t *testing.T) {
	t.Parallel()

	findings := make([]detector.Finding, 0, 5)
	for index := range 5 {
		finding := testFinding()
		finding.DedupeKey = "ontime-decline:cus_" + string(rune('a'+index))
		findings = append(findings, finding)
	}

	completion := &stubCompletion{err: errors.New("down")}

	result := newNarrator(completion).Narrate(t.Context(), &NarrateRequest{Findings: findings})

	assert.Len(t, result, 5)
	for _, finding := range findings {
		assert.Equal(t, finding.Headline, result[finding.DedupeKey].Headline)
	}
}

func TestNarrate_BatchesRatherThanCallingOncePerFinding(t *testing.T) {
	t.Parallel()

	findings := make([]detector.Finding, 0, maxFindingsPerCall+3)
	for index := range maxFindingsPerCall + 3 {
		finding := testFinding()
		finding.DedupeKey = "ontime-decline:cus_" + string(rune('a'+index))
		findings = append(findings, finding)
	}

	completion := completionReturning()

	newNarrator(completion).Narrate(t.Context(), &NarrateRequest{Findings: findings})

	require.Equal(t, 2, completion.calls, "11 findings must not be 11 round trips")

	// The split has to cover every finding exactly once: a batching bug that
	// dropped the tail would silently leave those cards unnarrated forever.
	described := 0
	for _, request := range completion.requests {
		for _, finding := range findings {
			if strings.Contains(request.Context.Sections[0].Content, finding.DedupeKey) {
				described++
			}
		}
	}
	assert.Equal(t, len(findings), described)
}

func TestNarrate_RoutesToTheInsightsTask(t *testing.T) {
	t.Parallel()

	completion := completionReturning()

	newNarrator(completion).Narrate(t.Context(), &NarrateRequest{
		Findings: []detector.Finding{testFinding()},
	})

	require.NotNil(t, completion.lastReq)
	assert.Equal(t, aiprovider.TaskOperationalInsights, completion.lastReq.Task)
}

// Subject names are customer-supplied text passing through a prompt. Marking the
// section untrusted is what tells the adapter to fence it.
func TestNarrate_SendsFindingsAsUntrustedContext(t *testing.T) {
	t.Parallel()

	completion := completionReturning()

	newNarrator(completion).Narrate(t.Context(), &NarrateRequest{
		Findings: []detector.Finding{testFinding()},
	})

	sections := completion.lastReq.Context.Sections
	require.Len(t, sections, 1)
	assert.False(t, sections[0].Trusted)
	assert.Contains(t, sections[0].Content, "Acme Foods")
}

// The prompt has to show the model exactly the figures the guard will accept
// back, or every narration is rejected and nothing explains why.
func TestNarrate_ShowsTheModelTheFiguresTheGuardAccepts(t *testing.T) {
	t.Parallel()

	completion := completionReturning()

	newNarrator(completion).Narrate(t.Context(), &NarrateRequest{
		Findings:   []detector.Finding{testFinding()},
		WindowDays: 30,
	})

	content := completion.lastReq.Context.Sections[0].Content
	assert.Contains(t, content, "82.4")
	assert.Contains(t, content, "93.1")
	assert.Contains(t, content, "prior 30 days")
}

func TestNarrate_TrimsOverlongProseRatherThanRejectingIt(t *testing.T) {
	t.Parallel()

	finding := testFinding()
	completion := completionReturning(map[string]string{
		"dedupeKey": finding.DedupeKey,
		"headline":  strings.Repeat("a", insight.MaxHeadlineLength+50),
		"narrative": strings.Repeat("b", insight.MaxNarrativeLength+50),
	})

	result := newNarrator(completion).Narrate(t.Context(), &NarrateRequest{
		Findings: []detector.Finding{finding},
	})

	narration := result[finding.DedupeKey]
	require.True(t, narration.Narrated)
	assert.LessOrEqual(t, len([]rune(narration.Headline)), insight.MaxHeadlineLength)
	assert.LessOrEqual(t, len([]rune(narration.Narrative)), insight.MaxNarrativeLength)
}

// An empty headline from the model would leave a blank card, so the detector's
// wording fills it rather than the model's silence winning.
func TestNarrate_KeepsTheDetectorHeadlineWhenTheModelReturnsNone(t *testing.T) {
	t.Parallel()

	finding := testFinding()
	completion := completionReturning(map[string]string{
		"dedupeKey": finding.DedupeKey,
		"headline":  "   ",
		"narrative": "On-time delivery is 82.4%.",
	})

	result := newNarrator(completion).Narrate(t.Context(), &NarrateRequest{
		Findings: []detector.Finding{finding},
	})

	assert.Equal(t, finding.Headline, result[finding.DedupeKey].Headline)
}
