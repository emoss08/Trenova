package agentresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type transcriptRuns struct {
	repositories.AgentRunRepository

	runs     map[pulid.ID]*agent.AgentRun
	requests []repositories.ListAgentRunsByIDsRequest
}

func (f *transcriptRuns) ListTranscriptsByIDs(
	_ context.Context,
	req repositories.ListAgentRunsByIDsRequest,
) ([]*agent.AgentRun, error) {
	f.requests = append(f.requests, req)
	found := make([]*agent.AgentRun, 0, len(req.IDs))
	for _, id := range req.IDs {
		if run, ok := f.runs[id]; ok {
			found = append(found, run)
		}
	}

	return found, nil
}

type transcriptFixture struct {
	engine *resolvertest.GrantingPermissionEngine
	runs   *transcriptRuns
	tenant pagination.TenantInfo
}

func newTranscriptFixture(granted bool) *transcriptFixture {
	engine := &resolvertest.GrantingPermissionEngine{Granted: map[string]bool{}}
	if granted {
		engine.Granted[permission.ResourceAgentRun.String()+"|"+string(permission.OpRead)] = true
	}

	return &transcriptFixture{
		engine: engine,
		runs:   &transcriptRuns{runs: map[pulid.ID]*agent.AgentRun{}},
		tenant: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
	}
}

func (f *transcriptFixture) resolver() *AgentRunResolver {
	return &AgentRunResolver{Deps: &Deps{Core: &base.Core{PermissionEngine: f.engine}}}
}

func (f *transcriptFixture) ctx(t *testing.T) context.Context {
	t.Helper()

	factory := loaders.NewAgentRunTranscriptByIDLoaderFactory(
		loaders.AgentRunTranscriptByIDLoaderFactoryParams{Runs: f.runs},
	)
	ctx := loaders.WithLoaders(t.Context(), &loaders.Loaders{
		AgentRunTranscriptByID: factory.NewForTenant(f.tenant),
	})

	return gqlctx.WithAuthContext(ctx, &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    f.tenant.UserID,
		UserID:         f.tenant.UserID,
		OrganizationID: f.tenant.OrgID,
		BusinessUnitID: f.tenant.BuID,
	})
}

func TestAgentRunTranscript_ReadsTheTranscriptThroughTheLoader(t *testing.T) {
	t.Parallel()

	fixture := newTranscriptFixture(true)
	runID := pulid.MustNew("arun_")
	kept := &agent.RunTranscript{
		Messages:        []agent.TranscriptMessage{{Role: "Assistant", Content: "Done."}},
		OmittedMessages: 2,
		OmittedAt:       1,
	}
	fixture.runs.runs[runID] = &agent.AgentRun{ID: runID, Transcript: kept}

	transcript, err := fixture.resolver().Transcript(fixture.ctx(t), &agent.AgentRun{ID: runID})
	require.NoError(t, err)

	assert.Equal(t, kept, transcript)
	require.Len(t, fixture.runs.requests, 1)
	assert.Equal(t, fixture.tenant.OrgID, fixture.runs.requests[0].TenantInfo.OrgID)
	assert.Equal(t, fixture.tenant.BuID, fixture.runs.requests[0].TenantInfo.BuID)
}

func TestAgentRunTranscript_IsAbsentForARunWithoutOne(t *testing.T) {
	t.Parallel()

	fixture := newTranscriptFixture(true)
	runID := pulid.MustNew("arun_")
	fixture.runs.runs[runID] = &agent.AgentRun{ID: runID}

	transcript, err := fixture.resolver().Transcript(fixture.ctx(t), &agent.AgentRun{ID: runID})
	require.NoError(t, err)
	assert.Nil(t, transcript)

	gone, err := fixture.resolver().Transcript(
		fixture.ctx(t),
		&agent.AgentRun{ID: pulid.MustNew("arun_")},
	)
	require.NoError(t, err)
	assert.Nil(t, gone, "a run that is not the tenant's reads as having no transcript")
}

func TestAgentRunTranscript_NeedsTheRunReadPermission(t *testing.T) {
	t.Parallel()

	fixture := newTranscriptFixture(false)
	runID := pulid.MustNew("arun_")
	fixture.runs.runs[runID] = &agent.AgentRun{
		ID:         runID,
		Transcript: &agent.RunTranscript{Messages: []agent.TranscriptMessage{{Role: "Assistant"}}},
	}

	transcript, err := fixture.resolver().Transcript(
		fixture.ctx(t),
		&agent.AgentRun{ID: runID, Transcript: fixture.runs.runs[runID].Transcript},
	)
	require.Error(t, err)
	assert.Nil(t, transcript)
	assert.True(t, errortypes.IsAuthorizationError(err))
	assert.Empty(t, fixture.runs.requests, "nothing is read for a reader who may not read runs")

	_, err = fixture.resolver().Transcript(t.Context(), &agent.AgentRun{ID: runID})
	require.Error(t, err, "an unauthenticated read is refused")
}
