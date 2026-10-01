package resolver

import (
	"net/http"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/emoss08/trenova/internal/api/graphql/generated"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// A refusal reaches the person deciding as its reasons, each with the field
// it names and the parameter that carries it, not only as one sentence.
func TestMyProposalPreview_ServesEachReasonOfARefusal(t *testing.T) {
	t.Parallel()

	auth := &authctx.AuthContext{
		PrincipalType:  string(services.PrincipalTypeUser),
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
	proposalID := pulid.MustNew("ap_")
	previews := &resolvertest.PreviewService{}
	root := FromBase(&base.Resolver{
		L: zap.NewNop(),
		PermissionEngine: &resolvertest.PreviewPermissions{Granted: map[string]bool{
			permission.ResourceAssistant.String() + "|" + string(permission.OpRead): true,
		}},
		AgentProposalService: &resolvertest.PreviewProposals{
			Proposal: &agent.AgentProposal{ID: proposalID},
		},
		AgentDecisionService:   &resolvertest.PreviewDecisions{Yours: true},
		AgentPlanService:       &resolvertest.PreviewPlans{},
		ProposalPreviewService: previews,
	})
	ctx := gqlctx.WithAuthContext(t.Context(), auth)
	previews.Warnings = []agent.PreviewWarning{{
		Code:    agent.PreviewWarningWouldFail,
		Args:    []string{"validation failed"},
		Message: "This would be refused as it stands: validation failed",
		Reasons: []agent.PreviewReason{{
			Field:   "bol",
			Label:   "BOL",
			Message: "BOL is already in use by shipment(s) with Pro Number(s): SEED-DET-009",
			Param:   "shipment.bol",
		}, {
			Message: "The customer is on credit hold",
		}},
	}, {
		Code:    agent.PreviewWarningWithheld,
		Message: "Some of it is hidden.",
	}}

	server := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: root}))
	server.AddTransport(transport.POST{})
	gql := client.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.ServeHTTP(w, r.WithContext(ctx))
	}))

	var response struct {
		MyProposalPreview struct {
			Warnings []struct {
				Code    string
				Message string
				Reasons []struct {
					Field   string
					Label   string
					Message string
					Param   string
				}
			}
		}
	}
	gql.MustPost(`query ($id: ID!) {
		myProposalPreview(id: $id) {
			warnings { code message reasons { field label message param } }
		}
	}`, &response, client.Var("id", proposalID.String()))

	warnings := response.MyProposalPreview.Warnings
	require.Len(t, warnings, 2)
	assert.Equal(t, "would_fail", warnings[0].Code)
	require.Len(t, warnings[0].Reasons, 2)
	bol := warnings[0].Reasons[0]
	assert.Equal(t, "bol", bol.Field)
	assert.Equal(t, "BOL", bol.Label)
	assert.Equal(t, "shipment.bol", bol.Param)
	assert.Contains(t, bol.Message, "SEED-DET-009")
	whole := warnings[0].Reasons[1]
	assert.Empty(t, whole.Field, "a refusal of the whole write names no field")
	assert.Empty(t, whole.Param)
	assert.NotNil(t, warnings[1].Reasons, "every warning serves a list, empty when it has none")
	assert.Empty(t, warnings[1].Reasons)
}
