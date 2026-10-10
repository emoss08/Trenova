package aiusageports

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestUsagePurposeForRun(t *testing.T) {
	t.Parallel()

	assert.Equal(t, AIUsagePurposeEvaluation, UsagePurposeForRun(pulid.MustNew("aeval_")))
	assert.Equal(t, kernelports.AIUsagePurposeLive, UsagePurposeForRun(pulid.MustNew("ar_")))
	assert.Equal(t, kernelports.AIUsagePurposeLive, UsagePurposeForRun(pulid.Nil))
}

func TestAttributedPurpose_SurvivesALostField(t *testing.T) {
	t.Parallel()

	explicit := &agentports.RunRequest{UsagePurpose: AIUsagePurposeEvaluation, RunID: pulid.MustNew("ar_")}
	restored := &agentports.RunRequest{RunID: pulid.MustNew("aeval_")}
	live := &agentports.RunRequest{RunID: pulid.MustNew("ar_")}

	assert.Equal(t, AIUsagePurposeEvaluation, explicit.AttributedPurpose())
	assert.Equal(t, AIUsagePurposeEvaluation, restored.AttributedPurpose(),
		"a durable run rebuilt without the field still reads its purpose from the run id")
	assert.Equal(t, kernelports.AIUsagePurposeLive, live.AttributedPurpose())
}

func TestAIUsageAttributionEvaluates(t *testing.T) {
	t.Parallel()

	assert.True(t, AIUsageAttribution{Purpose: AIUsagePurposeEvaluation}.Evaluates())
	assert.True(t, AIUsageAttribution{RunID: pulid.MustNew("aeval_")}.Evaluates())
	assert.False(t, AIUsageAttribution{RunID: pulid.MustNew("ar_")}.Evaluates())
	assert.False(t, AIUsageAttribution{}.Evaluates())
}
