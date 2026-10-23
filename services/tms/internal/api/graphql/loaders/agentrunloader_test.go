package loaders

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubRunsByIDs struct {
	visible  []*agent.AgentRun
	requests []agentports.ListAgentRunsByIDsRequest
}

func (s *stubRunsByIDs) ListByIDs(
	_ context.Context,
	req agentports.ListAgentRunsByIDsRequest,
) ([]*agent.AgentRun, error) {
	s.requests = append(s.requests, req)

	return s.visible, nil
}

func (s *stubRunsByIDs) ListTranscriptsByIDs(
	ctx context.Context,
	req agentports.ListAgentRunsByIDsRequest,
) ([]*agent.AgentRun, error) {
	return s.ListByIDs(ctx, req)
}

func TestAgentRunLoaders_ReadForThePersonMakingTheRequest(t *testing.T) {
	t.Parallel()

	shown := &agent.AgentRun{ID: pulid.MustNew("ar_")}
	hidden := pulid.MustNew("ar_")

	for name, userID := range map[string]pulid.ID{
		"a person":               pulid.MustNew("usr_"),
		"an agent or an API key": pulid.Nil,
	} {
		tenant := pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: userID,
		}
		runs := &stubRunsByIDs{visible: []*agent.AgentRun{shown}}

		byID := newAgentRunByIDLoader(&AgentRunByIDLoaderFactory{runs: runs}, tenant)
		transcripts := newAgentRunTranscriptByIDLoader(
			&AgentRunTranscriptByIDLoaderFactory{runs: runs},
			tenant,
		)

		for _, loader := range []interface {
			Load(context.Context, string) (*agent.AgentRun, error)
		}{byID, transcripts} {
			got, err := loader.Load(t.Context(), shown.ID.String())
			require.NoError(t, err, name)
			assert.Same(t, shown, got, name)

			_, err = loader.Load(t.Context(), hidden.String())
			assert.True(t, errortypes.IsNotFoundError(err),
				"%s: a run the repository leaves out reads as missing", name)
		}

		require.NotEmpty(t, runs.requests, name)
		for _, req := range runs.requests {
			assert.Equal(t, agentports.PersonRunRead(userID), req.Reader, name)
			assert.Equal(t, tenant, req.TenantInfo, name)
		}
	}
}
