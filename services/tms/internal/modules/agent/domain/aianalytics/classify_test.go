package aianalytics_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/stretchr/testify/assert"
)

func TestClassifyToolFailureNamesWhyACallFailed(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		verdict string
		reason  string
		content string
		want    aianalytics.Cause
	}{
		{
			"bad arguments",
			"invalid",
			"its arguments were not valid JSON",
			"",
			aianalytics.CauseWrongArguments,
		},
		{
			"unregistered tool",
			"invalid",
			aianalytics.ReasonUnregistered,
			"",
			aianalytics.CauseToolNotHeld,
		},
		{"tool not held", "denied", aianalytics.ReasonNotHeld, "", aianalytics.CauseToolNotHeld},
		{
			"permission refused",
			"denied",
			"the person may not read shipments",
			"",
			aianalytics.CauseBlockedByRule,
		},
		{
			"repeat",
			"duplicate",
			"the same question was already asked",
			"",
			aianalytics.CauseRepeatedCall,
		},
		{
			"budget",
			"over_budget",
			"the turn's tool budget is spent",
			"",
			aianalytics.CauseBlockedByRule,
		},
		{"deadline", "failed", "context deadline exceeded", "", aianalytics.CauseToolTimedOut},
		{
			"timed out in content",
			"failed",
			"",
			"The carrier API timed out after 30s",
			aianalytics.CauseToolTimedOut,
		},
		{"error", "failed", "record not found", "", aianalytics.CauseToolError},
		{"no verdict", "", "", "", aianalytics.CauseToolError},
	}
	for _, tc := range cases {
		assert.Equal(
			t,
			tc.want,
			aianalytics.ClassifyToolFailure(tc.verdict, tc.reason, tc.content),
			tc.name,
		)
	}
}

func TestCauseBlameSplitsTheModelsMistakesFromTheTools(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(t, []aianalytics.Cause{
		aianalytics.CauseWrongArguments,
		aianalytics.CauseRepeatedCall,
		aianalytics.CauseToolNotHeld,
	}, aianalytics.ModelCauses())
	assert.Equal(t, aianalytics.BlameTool, aianalytics.CauseToolTimedOut.Blame())
	assert.Equal(t, aianalytics.BlameTool, aianalytics.CauseBlockedByRule.Blame())
	assert.Equal(t, aianalytics.BlameTool, aianalytics.CauseToolError.Blame())
}
