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

func (s *stubCompletion) Diagnose(
	_ context.Context,
	_ *serviceports.DiagnoseRequest,
) (*serviceports.DiagnoseResult, error) {
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
		tenant(),
		"Write me a Python script to export loads",
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
				tenant(),
				"Which driver is on load 12345?",
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
			decision := newGuard(t, stub).Evaluate(t.Context(), tenant(), "Tell me a joke")

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
	decision := newGuard(t, stub).Evaluate(t.Context(), tenant(), "Which driver is on load 1?")

	require.True(t, decision.Allowed)
	assert.Equal(t, agentguard.StageUnavailable, decision.Stage)
}

// A configured classifier that breaks is different: a control that silently
// stops working is how guardrails rot, so the request is refused instead.
func TestEvaluate_RefusesWhenConfiguredClassifierFails(t *testing.T) {
	t.Parallel()

	stub := &stubCompletion{err: errors.New("upstream timeout")}
	decision := newGuard(t, stub).Evaluate(t.Context(), tenant(), "Which driver is on load 1?")

	require.False(t, decision.Allowed)
	assert.Equal(t, agentguard.StageUnavailable, decision.Stage)
	assert.Equal(t, agentguard.ReasonClassifierUnavailable, decision.Reason)
}

// The message being classified must reach the model as data. If it were trusted
// context, a request could argue its own way into scope.
func TestEvaluate_SendsRequestAsUntrustedContext(t *testing.T) {
	t.Parallel()

	stub := &stubCompletion{category: string(agentguard.CategoryTransportationOperations)}
	newGuard(t, stub).Evaluate(t.Context(), tenant(), "Which driver is on load 12345?")

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
