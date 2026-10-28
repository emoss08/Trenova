package agentruneventresolver

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

type recordingEventRepo struct {
	agentruntimeports.AgentRunEventRepository

	requests []*agentruntimeports.ListAgentRunEventConnectionRequest
}

func (r *recordingEventRepo) ListConnection(
	_ context.Context,
	req *agentruntimeports.ListAgentRunEventConnectionRequest,
) (*pagination.CursorListResult[*agent.AgentRunEvent], error) {
	r.requests = append(r.requests, req)

	return &pagination.CursorListResult[*agent.AgentRunEvent]{}, nil
}

func TestAgentRunEvents_ReadForThePersonAsking(t *testing.T) {
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
		events := &recordingEventRepo{}
		resolver := &QueryResolver{Deps: &Deps{
			Core:              &resolvercore.Core{PermissionEngine: engine},
			AgentRunEventRepo: events,
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

		_, err := resolver.AgentRunEvents(ctx, gqlmodel.DataTableConnectionInput{})
		require.NoError(t, err, name)

		require.Len(t, events.requests, 1, name)
		assert.Equal(
			t,
			agentruntimeports.PersonRunRead(principal.userID),
			events.requests[0].Reader,
			name,
		)
		assert.Equal(
			t,
			string(agentruntimeports.RunStepOwnerAgentRun),
			events.requests[0].OwnerKind,
			name,
		)
	}
}
