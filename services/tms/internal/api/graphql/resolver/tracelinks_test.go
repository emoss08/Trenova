package resolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const linkedTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"

func TestTraceURL_LinksARecordToItsTraceWhenABackendIsConfigured(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	cfg.Monitoring.Tracing.TraceURLTemplate = "https://traces.example.com/trace/{traceId}"
	r := &Resolver{traceURL: traceURLBuilder(cfg)}

	run, err := (&agentRunResolver{r}).TraceURL(
		t.Context(),
		&agent.AgentRun{TraceID: linkedTraceID},
	)
	require.NoError(t, err)
	require.NotNil(t, run)
	assert.Equal(t, "https://traces.example.com/trace/"+linkedTraceID, *run)

	proposal, err := (&agentProposalResolver{r}).TraceURL(t.Context(),
		&agent.AgentProposal{TraceID: linkedTraceID})
	require.NoError(t, err)
	require.NotNil(t, proposal)

	decision, err := (&agentDecisionResolver{r}).TraceURL(t.Context(),
		&agent.AgentDecision{TraceID: linkedTraceID})
	require.NoError(t, err)
	require.NotNil(t, decision)

	untraced, err := (&agentRunResolver{r}).TraceURL(t.Context(), &agent.AgentRun{})
	require.NoError(t, err)
	assert.Nil(t, untraced, "a record from before traces were kept has no link")
}

func TestTraceURL_IsAbsentWithoutABackend(t *testing.T) {
	t.Parallel()

	for _, r := range []*Resolver{
		{traceURL: traceURLBuilder(&config.Config{})},
		{traceURL: traceURLBuilder(nil)},
		{},
	} {
		link, err := (&agentRunResolver{r}).TraceURL(
			t.Context(),
			&agent.AgentRun{TraceID: linkedTraceID},
		)
		require.NoError(t, err)
		assert.Nil(t, link)
	}
}
