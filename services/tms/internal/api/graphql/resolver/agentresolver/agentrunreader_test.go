package agentresolver

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingRunService struct {
	agentruntimeports.AgentRunService

	lists []*agentruntimeports.ListAgentRunConnectionRequest
	gets  []agentruntimeports.GetAgentRunByIDRequest
}

func (s *recordingRunService) ListConnection(
	_ context.Context,
	req *agentruntimeports.ListAgentRunConnectionRequest,
) (*pagination.CursorListResult[*agent.AgentRun], error) {
	s.lists = append(s.lists, req)

	return &pagination.CursorListResult[*agent.AgentRun]{}, nil
}

func (s *recordingRunService) GetByID(
	_ context.Context,
	req agentruntimeports.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	s.gets = append(s.gets, req)

	return &agent.AgentRun{ID: req.ID}, nil
}

func TestAgentRunQueries_ReadForThePersonAsking(t *testing.T) {
	t.Parallel()

	for name, principal := range map[string]struct {
		kind   string
		userID pulid.ID
	}{
		"a person":   {kind: authctx.PrincipalTypeUser, userID: pulid.MustNew("usr_")},
		"an agent":   {kind: authctx.PrincipalTypeAgent},
		"an API key": {kind: authctx.PrincipalTypeAPIKey},
	} {
		engine := &resolvertest.GrantingPermissionEngine{Granted: map[string]bool{
			permission.ResourceAgentRun.String() + "|" + string(permission.OpRead): true,
		}}
		runs := &recordingRunService{}
		resolver := &QueryResolver{Deps: &Deps{
			Core:            &resolvercore.Core{PermissionEngine: engine},
			AgentRunService: runs,
		}}

		ctx := graphql.WithOperationContext(t.Context(), &graphql.OperationContext{})
		ctx = graphql.WithFieldContext(ctx, &graphql.FieldContext{})
		ctx = gqlctx.WithAuthContext(ctx, &authctx.AuthContext{
			PrincipalType:  principal.kind,
			PrincipalID:    pulid.MustNew("prn_"),
			UserID:         principal.userID,
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		})

		_, err := resolver.AgentRuns(ctx, gqlmodel.DataTableConnectionInput{})
		require.NoError(t, err, name)
		_, err = resolver.AgentRun(ctx, pulid.MustNew("ar_").String())
		require.NoError(t, err, name)

		want := agentruntimeports.PersonRunRead(principal.userID)
		require.Len(t, runs.lists, 1, name)
		assert.Equal(t, want, runs.lists[0].Reader, name)
		require.Len(t, runs.gets, 1, name)
		assert.Equal(t, want, runs.gets[0].Reader, name)
	}
}
