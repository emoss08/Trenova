package agentflow

import (
	"context"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow/agentflowcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type countingObserver struct {
	calls int
}

func (o *countingObserver) ObserveTool(
	context.Context,
	RunContext,
	*agentruntimeports.ToolObservation,
	kernelports.AssistantStreamEmitter,
) (*agentruntimeports.ShownArtifact, []*assistantartifact.Artifact, error) {
	o.calls++

	return nil, nil, nil
}

func TestRunContext_CarriesAnInlineReplyAsData(t *testing.T) {
	t.Parallel()

	run := NewRunContext(&agentruntimeports.RunRequest{
		Definition: &agentdefinition.Definition{ID: pulid.MustNew("agdef_")},
		ThreadID:   pulid.MustNew("athr_"),
		Inline:     true,
	}, agentflowcontract.PriorityInteractive)

	assert.True(t, run.Inline)
	assert.True(t, run.request().Inline)

	encoded, err := sonic.Marshal(run)
	require.NoError(t, err)
	var decoded RunContext
	require.NoError(t, sonic.Unmarshal(encoded, &decoded))
	assert.True(t, decoded.Inline)

	var older RunContext
	require.NoError(t, sonic.Unmarshal([]byte(`{"threadId":"athr_1"}`), &older))
	assert.False(t, older.Inline,
		"a run recorded before inline replies existed keeps its artifacts")
}

// A palette question's tool results are not kept as artifacts: nothing beside
// the answer could show them, and the model would be told to point to them.
func TestObserving_KeepsNothingForAnInlineReply(t *testing.T) {
	t.Parallel()

	observer := &countingObserver{}
	a := NewActivities(ActivitiesParams{Logger: zap.NewNop(), Observer: observer})

	observe, kept, done := a.observing(t.Context(), &RunContext{
		ThreadID: pulid.MustNew("athr_"),
		Inline:   true,
	})
	defer done()

	assert.Nil(t, observe)
	assert.Empty(t, *kept)
	assert.Zero(t, observer.calls)
}
