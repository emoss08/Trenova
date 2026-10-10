package aiusageports_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestUsagePurposeForRun(t *testing.T) {
	t.Parallel()

	assert.Equal(t, aiusageports.AIUsagePurposeEvaluation, aiusageports.UsagePurposeForRun(pulid.MustNew("aeval_")))
	assert.Equal(t, kernelports.AIUsagePurposeLive, aiusageports.UsagePurposeForRun(pulid.MustNew("ar_")))
	assert.Equal(t, kernelports.AIUsagePurposeLive, aiusageports.UsagePurposeForRun(pulid.Nil))
}

func TestAttributedPurpose_SurvivesALostField(t *testing.T) {
	t.Parallel()

	explicit := &agentports.RunRequest{UsagePurpose: aiusageports.AIUsagePurposeEvaluation, RunID: pulid.MustNew("ar_")}
	restored := &agentports.RunRequest{RunID: pulid.MustNew("aeval_")}
	live := &agentports.RunRequest{RunID: pulid.MustNew("ar_")}

	assert.Equal(t, aiusageports.AIUsagePurposeEvaluation, explicit.AttributedPurpose())
	assert.Equal(t, aiusageports.AIUsagePurposeEvaluation, restored.AttributedPurpose(),
		"a durable run rebuilt without the field still reads its purpose from the run id")
	assert.Equal(t, kernelports.AIUsagePurposeLive, live.AttributedPurpose())
}

func TestAIUsageAttributionEvaluates(t *testing.T) {
	t.Parallel()

	assert.True(t, aiusageports.AIUsageAttribution{Purpose: aiusageports.AIUsagePurposeEvaluation}.Evaluates())
	assert.True(t, aiusageports.AIUsageAttribution{RunID: pulid.MustNew("aeval_")}.Evaluates())
	assert.False(t, aiusageports.AIUsageAttribution{RunID: pulid.MustNew("ar_")}.Evaluates())
	assert.False(t, aiusageports.AIUsageAttribution{}.Evaluates())
}
