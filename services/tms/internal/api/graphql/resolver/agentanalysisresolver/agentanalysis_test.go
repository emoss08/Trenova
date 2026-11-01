package agentanalysisresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/agent/ports/analysisports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubRuns struct {
	listed []*analysisports.ListRunsRequest
	read   []*analysisports.RunRequest
}

func (s *stubRuns) ListRuns(
	_ context.Context,
	req *analysisports.ListRunsRequest,
) (*analysisports.RunListResponse, error) {
	s.listed = append(s.listed, req)
	return &analysisports.RunListResponse{Items: []*analysis.Run{{Title: "Lane margin"}}}, nil
}

func (s *stubRuns) Run(_ context.Context, req *analysisports.RunRequest) (*analysis.Run, error) {
	s.read = append(s.read, req)
	return &analysis.Run{ID: req.RunID, Title: "Lane margin"}, nil
}

func grantKey(resource permission.Resource, operation permission.Operation) string {
	return resource.String() + "|" + string(operation)
}

func TestAgentAnalysisRuns_NeedsAgentRunRead(t *testing.T) {
	t.Parallel()

	engine := resolvertest.RefusingEngine()
	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
	ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

	_, err := (&QueryResolver{deps}).AgentAnalysisRuns(ctx, nil, nil, nil)

	resolvertest.RequireRefused(t, err, engine, permission.ResourceAgentRun, permission.OpRead)
}

func TestAgentAnalysisRuns_ReadsThePageAfterTheCursorInTheCallersTenant(t *testing.T) {
	t.Parallel()

	engine := resolvertest.RefusingEngine()
	engine.Granted[grantKey(permission.ResourceAgentRun, permission.OpRead)] = true
	runs := &stubRuns{}
	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}, AnalysisRunService: runs}
	authCtx := resolvertest.UserAuthContext()
	ctx := resolvertest.AuthedContext(t, authCtx)
	before, limit := 1_790_000_000, 25
	beforeID := pulid.MustNew("aanr_")
	beforeRaw := beforeID.String()

	page, err := (&QueryResolver{deps}).AgentAnalysisRuns(ctx, &before, &beforeRaw, &limit)
	require.NoError(t, err)

	require.Len(t, runs.listed, 1)
	req := runs.listed[0]
	assert.Equal(t, int64(1_790_000_000), req.Before)
	assert.Equal(t, beforeID, req.BeforeID)
	assert.Equal(t, 25, req.Limit)
	assert.Equal(t, authCtx.OrganizationID, req.TenantInfo.OrgID)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "Lane margin", page.Items[0].Title)
}

func TestListRunsRequest_BoundsThePage(t *testing.T) {
	t.Parallel()

	var invalid *errortypes.Error

	tooMany := maxRunsPerPage + 1
	_, err := listRunsRequest(nil, nil, &tooMany)
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, "limit", invalid.Field)

	negative := -1
	_, err = listRunsRequest(&negative, nil, nil)
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, "before", invalid.Field)

	malformed := "not-an-id"
	_, err = listRunsRequest(nil, &malformed, nil)
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, "beforeId", invalid.Field)
}

func TestAgentAnalysisRun_OverseesOnlyWithAgentRunRead(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		granted  []string
		oversees bool
		refused  bool
	}{
		{
			name:     "someone who oversees agents reads any analysis",
			granted:  []string{grantKey(permission.ResourceAgentRun, permission.OpRead)},
			oversees: true,
		},
		{
			name:    "the person it ran for reads it through code mode",
			granted: []string{grantKey(permission.ResourceCodeMode, permission.OpRead)},
		},
		{name: "neither is refused", refused: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			for _, key := range tc.granted {
				engine.Granted[key] = true
			}
			runs := &stubRuns{}
			deps := &Deps{
				Core:               &resolvercore.Core{PermissionEngine: engine},
				AnalysisRunService: runs,
			}
			authCtx := resolvertest.UserAuthContext()
			ctx := resolvertest.AuthedContext(t, authCtx)
			runID := pulid.MustNew("aanr_")

			run, err := (&QueryResolver{deps}).AgentAnalysisRun(ctx, runID.String())

			if tc.refused {
				var authzErr *errortypes.AuthorizationError
				require.ErrorAs(t, err, &authzErr)
				assert.Empty(t, runs.read)
				return
			}
			require.NoError(t, err)
			require.Len(t, runs.read, 1)
			assert.Equal(t, runID, runs.read[0].RunID)
			assert.Equal(t, tc.oversees, runs.read[0].Oversees)
			assert.Equal(t, authCtx.UserID, runs.read[0].TenantInfo.UserID)
			assert.Equal(t, runID, run.ID)
		})
	}
}
