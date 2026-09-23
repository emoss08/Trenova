package agentguard_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bytedance/sonic"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentguard"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubCompletion struct {
	category string
	err      error
	calls    int
	lastReq  *serviceports.StructuredCompletionRequest
}

func (s *stubCompletion) CompleteStructured(
	_ context.Context,
	req *serviceports.StructuredCompletionRequest,
) (*serviceports.StructuredCompletionResult, error) {
	s.calls++
	s.lastReq = req
	if s.err != nil {
		return nil, s.err
	}

	text, _ := sonic.Marshal(agentguard.ClassifierResult{
		Category:  s.category,
		Reasoning: "stub",
	})

	return &serviceports.StructuredCompletionResult{Text: string(text)}, nil
}

func (s *stubCompletion) CompleteChat(
	_ context.Context,
	_ *serviceports.ChatCompletionRequest,
) (*serviceports.ChatCompletionResult, error) {
	return nil, errors.New("not used")
}

func (s *stubCompletion) StreamChat(
	context.Context,
	*serviceports.ChatCompletionRequest,
	serviceports.ChatStreamSink,
) (*serviceports.ChatCompletionResult, error) {
	return nil, errors.New("not used")
}

func newGuard(t *testing.T, stub *stubCompletion) *agentguard.Service {
	t.Helper()

	return agentguard.New(agentguard.Params{
		Logger:     zap.NewNop(),
		Completion: stub,
	})
}

func tenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
}

func TestEvaluate_DeterministicRefusalSkipsClassifier(t *testing.T) {
	t.Parallel()

	stub := &stubCompletion{category: string(agentguard.CategoryTransportationOperations)}
	decision := newGuard(t, stub).Evaluate(
		t.Context(),
		agentguard.EvaluateRequest{
			TenantInfo: tenant(),
			Input:      "Write me a Python script to export loads",
		},
	)

	require.False(t, decision.Allowed)
	assert.Equal(t, agentguard.StageDeterministic, decision.Stage)
	assert.Equal(t, agentguard.ReasonCodeGeneration, decision.Reason)
	assert.Zero(t, stub.calls, "a pattern match must not cost a model call")
}

func TestEvaluate_AllowsInScopeCategories(t *testing.T) {
	t.Parallel()

	inScope := []agentguard.Category{
		agentguard.CategoryTransportationOperations,
		agentguard.CategorySystemAutomation,
		agentguard.CategorySystemUsage,
		agentguard.CategoryTransportationKnowledge,
	}

	for _, category := range inScope {
		t.Run(string(category), func(t *testing.T) {
			t.Parallel()
			stub := &stubCompletion{category: string(category)}
			decision := newGuard(t, stub).Evaluate(
				t.Context(),
				agentguard.EvaluateRequest{
					TenantInfo: tenant(),
					Input:      "Which driver is on load 12345?",
				},
			)

			require.True(t, decision.Allowed)
			assert.Equal(t, agentguard.StageClassifier, decision.Stage)
			assert.Equal(t, category, decision.Category)
		})
	}
}

func TestEvaluate_RefusesOutOfScopeCategories(t *testing.T) {
	t.Parallel()

	cases := []struct {
		category agentguard.Category
		reason   agentguard.Reason
	}{
		{agentguard.CategoryCodeGeneration, agentguard.ReasonCodeGeneration},
		{agentguard.CategoryGeneralKnowledge, agentguard.ReasonOffDomain},
		{agentguard.CategoryPromptManipulation, agentguard.ReasonPromptManipulation},
		{agentguard.CategoryOther, agentguard.ReasonOffDomain},
	}

	for _, tc := range cases {
		t.Run(string(tc.category), func(t *testing.T) {
			t.Parallel()
			stub := &stubCompletion{category: string(tc.category)}
			decision := newGuard(
				t,
				stub,
			).Evaluate(t.Context(), agentguard.EvaluateRequest{TenantInfo: tenant(), Input: "Tell me a joke"})

			require.False(t, decision.Allowed)
			assert.Equal(t, agentguard.StageClassifier, decision.Stage)
			assert.Equal(t, tc.reason, decision.Reason)
			assert.NotEmpty(t, decision.Message)
		})
	}
}

// An unconfigured classifier is a deployment choice, not a malfunction, so chat
// keeps working on the deterministic rules and the assistant's own refusal.
func TestEvaluate_AllowsWhenNoClassifierConfigured(t *testing.T) {
	t.Parallel()

	stub := &stubCompletion{
		err: errortypes.NewBusinessError("no provider").
			WithInternal(serviceports.ErrNoProviderConfigured),
	}
	decision := newGuard(
		t,
		stub,
	).Evaluate(t.Context(), agentguard.EvaluateRequest{TenantInfo: tenant(), Input: "Which driver is on load 1?"})

	require.True(t, decision.Allowed)
	assert.Equal(t, agentguard.StageUnavailable, decision.Stage)
}

/*
A configured classifier that breaks lands where an unconfigured one does.

These two used to differ: an absent provider proceeded, a failing one refused,
on the reasoning that a control which silently stops working is how guardrails
rot. In both states the control is not operating — the only difference is
whether a row exists — so the distinction bought nothing and cost availability
in proportion to how much of the product had been configured.

It showed up on a single flaky provider: "which drivers have a medical card
expiring in the next 360 days" was refused outright, twenty-six seconds after
the same question had been answered correctly.

The deterministic rules have already run and passed by this point, the system
prompt still refuses off-domain and software work, and tool calls are
authorized against the acting user separately. Failing this closed denies
service rather than protecting anything.
*/
func TestEvaluate_FallsBackToDeterministicWhenClassifierFails(t *testing.T) {
	t.Parallel()

	stub := &stubCompletion{err: errors.New("upstream timeout")}
	decision := newGuard(
		t,
		stub,
	).Evaluate(t.Context(), agentguard.EvaluateRequest{TenantInfo: tenant(), Input: "Which driver is on load 1?"})

	require.True(t, decision.Allowed)
	assert.Equal(t, agentguard.StageUnavailable, decision.Stage,
		"the degradation is recorded even though the request proceeds")
}

// A failing classifier must not become a way past the deterministic rules. They
// run first and their refusal never reaches the classifier at all.
func TestEvaluate_ClassifierFailureDoesNotBypassDeterministicRules(t *testing.T) {
	t.Parallel()

	stub := &stubCompletion{err: errors.New("upstream timeout")}
	decision := newGuard(t, stub).
		Evaluate(t.Context(), agentguard.EvaluateRequest{TenantInfo: tenant(), Input: "Write me a Python script to parse this CSV"})

	require.False(t, decision.Allowed)
	assert.Equal(t, agentguard.StageDeterministic, decision.Stage)
}

// The stricter posture stays available for an operator who wants it.
func TestEvaluate_RefusesWhenUnavailableAndConfiguredToRefuse(t *testing.T) {
	t.Parallel()

	stub := &stubCompletion{err: errors.New("upstream timeout")}
	guard := newGuard(t, stub)
	guard.RefuseWhenUnavailable = true

	decision := guard.Evaluate(
		t.Context(),
		agentguard.EvaluateRequest{TenantInfo: tenant(), Input: "Which driver is on load 1?"},
	)

	require.False(t, decision.Allowed)
	assert.Equal(t, agentguard.ReasonClassifierUnavailable, decision.Reason)
}

// The message being classified must reach the model as data. If it were trusted
// context, a request could argue its own way into scope.
func TestEvaluate_SendsRequestAsUntrustedContext(t *testing.T) {
	t.Parallel()

	stub := &stubCompletion{category: string(agentguard.CategoryTransportationOperations)}
	newGuard(
		t,
		stub,
	).Evaluate(t.Context(), agentguard.EvaluateRequest{TenantInfo: tenant(), Input: "Which driver is on load 12345?"})

	require.NotNil(t, stub.lastReq)
	require.Len(t, stub.lastReq.Context.Sections, 1)
	assert.False(t, stub.lastReq.Context.Sections[0].Trusted,
		"the classified message must never be trusted context")
	assert.Equal(t, "ScopeClassification", string(stub.lastReq.Task),
		"classification must route to its own task so it lands on a cheap model")
}

func TestEvaluateOutput(t *testing.T) {
	t.Parallel()

	t.Run("refuses a fenced code block", func(t *testing.T) {
		t.Parallel()
		decision := agentguard.EvaluateOutput("Sure:\n```python\nprint(1)\n```")
		require.False(t, decision.Allowed)
		assert.Equal(t, agentguard.ReasonCodeGeneration, decision.Reason)
	})

	t.Run("refuses function syntax", func(t *testing.T) {
		t.Parallel()
		decision := agentguard.EvaluateOutput("def rate(miles):\n    return miles")
		require.False(t, decision.Allowed)
	})

	t.Run("allows an ordinary operational answer", func(t *testing.T) {
		t.Parallel()
		decision := agentguard.EvaluateOutput(
			"Load 12345 is assigned to driver Maria Ortiz and is routed through the Memphis terminal.",
		)
		assert.True(t, decision.Allowed)
	})

	t.Run("allows a markdown table of freight data", func(t *testing.T) {
		t.Parallel()
		decision := agentguard.EvaluateOutput(
			"| Load | Class | Status |\n|---|---|---|\n| 12345 | 70 | In Transit |",
		)
		assert.True(t, decision.Allowed)
	})

	t.Run("allows an unlabelled block holding freight data", func(t *testing.T) {
		t.Parallel()
		// A fence with no language tag is usually a manifest or an address, not a
		// program, so it must not be refused.
		decision := agentguard.EvaluateOutput("Manifest:\n```\nBOL 998812\nPallets 14\n```")
		assert.True(t, decision.Allowed)
	})
}

// These two satisfy the completion port; the guard's stub runs every call inline and
// never defers one, so a submission carries its answer and there is no handle
// to poll.
func (s *stubCompletion) SubmitBackground(
	ctx context.Context,
	req *serviceports.StructuredCompletionRequest,
) (*serviceports.BackgroundSubmission, error) {
	result, err := s.CompleteStructured(ctx, req)
	if err != nil {
		return nil, err
	}

	return &serviceports.BackgroundSubmission{Result: result}, nil
}

func (s *stubCompletion) PollBackground(
	context.Context,
	*serviceports.BackgroundPollRequest,
) (*serviceports.BackgroundOutcome, error) {
	return nil, errors.New("this completion runs inline and issues no handle to poll")
}

/*
The same question twice costs one classification.

Every message pays a round trip and roughly nine hundred tokens of classifier
prompt before the real turn starts. Classification is a pure function of the
text, and in operations the same question is asked constantly — the transcripts
behind this change show one question asked three times in a few minutes, each
paying full price.
*/
func TestEvaluate_ClassifiesTheSameQuestionOnce(t *testing.T) {
	t.Parallel()

	stub := &stubCompletion{category: string(agentguard.CategoryTransportationOperations)}
	guard := newGuard(t, stub)
	tenantInfo := tenant()

	first := guard.Evaluate(
		t.Context(),
		agentguard.EvaluateRequest{
			TenantInfo: tenantInfo,
			Input:      "Which drivers hold a hazmat endorsement?",
		},
	)
	second := guard.Evaluate(
		t.Context(),
		agentguard.EvaluateRequest{
			TenantInfo: tenantInfo,
			Input:      "Which drivers hold a hazmat endorsement?",
		},
	)

	require.True(t, first.Allowed)
	require.True(t, second.Allowed)
	assert.Equal(t, 1, stub.calls, "the second ask reuses the first verdict")
	assert.Equal(t, first.Category, second.Category)
}

// Wording a question the same way with different capitals or spacing is the
// same question.
func TestEvaluate_TreatsCaseAndSpacingAsTheSameQuestion(t *testing.T) {
	t.Parallel()

	stub := &stubCompletion{category: string(agentguard.CategoryTransportationOperations)}
	guard := newGuard(t, stub)
	tenantInfo := tenant()

	guard.Evaluate(
		t.Context(),
		agentguard.EvaluateRequest{TenantInfo: tenantInfo, Input: "Which driver is on load 1?"},
	)
	guard.Evaluate(
		t.Context(),
		agentguard.EvaluateRequest{TenantInfo: tenantInfo, Input: "  which driver is on load 1?  "},
	)

	assert.Equal(t, 1, stub.calls)
}

// A refusal is a verdict too, and re-refusing costs nothing.
func TestEvaluate_RemembersARefusal(t *testing.T) {
	t.Parallel()

	stub := &stubCompletion{category: string(agentguard.CategoryCodeGeneration)}
	guard := newGuard(t, stub)
	tenantInfo := tenant()

	first := guard.Evaluate(
		t.Context(),
		agentguard.EvaluateRequest{
			TenantInfo: tenantInfo,
			Input:      "Explain how this query planner works",
		},
	)
	second := guard.Evaluate(
		t.Context(),
		agentguard.EvaluateRequest{
			TenantInfo: tenantInfo,
			Input:      "Explain how this query planner works",
		},
	)

	require.False(t, first.Allowed)
	require.False(t, second.Allowed)
	assert.Equal(t, 1, stub.calls)
}

/*
A failure is never remembered.

Falling back to the deterministic rules is a degraded state. Caching it would
keep a request degraded long after the classifier recovered, turning a blip
into an outage that outlives its cause.
*/
func TestEvaluate_DoesNotRememberAFailure(t *testing.T) {
	t.Parallel()

	stub := &stubCompletion{err: errors.New("upstream timeout")}
	guard := newGuard(t, stub)
	tenantInfo := tenant()

	guard.Evaluate(
		t.Context(),
		agentguard.EvaluateRequest{TenantInfo: tenantInfo, Input: "Which driver is on load 1?"},
	)
	guard.Evaluate(
		t.Context(),
		agentguard.EvaluateRequest{TenantInfo: tenantInfo, Input: "Which driver is on load 1?"},
	)

	assert.Equal(t, 2, stub.calls, "a classifier that failed is asked again, not written off")
}

// One organization's verdicts are not another's.
func TestEvaluate_KeepsVerdictsPerOrganization(t *testing.T) {
	t.Parallel()

	stub := &stubCompletion{category: string(agentguard.CategoryTransportationOperations)}
	guard := newGuard(t, stub)

	guard.Evaluate(
		t.Context(),
		agentguard.EvaluateRequest{TenantInfo: tenant(), Input: "Which driver is on load 1?"},
	)
	guard.Evaluate(
		t.Context(),
		agentguard.EvaluateRequest{TenantInfo: tenant(), Input: "Which driver is on load 1?"},
	)

	assert.Equal(t, 2, stub.calls, "a different tenant does not read the first one's verdict")
}
