package agentpreviewresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type previewHarness struct {
	resolver  *QueryResolver
	previews  *resolvertest.PreviewService
	decisions *resolvertest.PreviewDecisions
	plans     *resolvertest.PreviewPlans
	ctx       context.Context
	proposal  pulid.ID
}

func newPreviewHarness(t *testing.T, granted ...string) *previewHarness {
	t.Helper()

	auth := &authctx.AuthContext{
		PrincipalType:  string(services.PrincipalTypeUser),
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
	grants := make(map[string]bool, len(granted))
	for _, grant := range granted {
		grants[grant] = true
	}
	proposalID := pulid.MustNew("ap_")
	h := &previewHarness{
		previews:  &resolvertest.PreviewService{},
		decisions: &resolvertest.PreviewDecisions{},
		plans:     &resolvertest.PreviewPlans{},
		ctx:       gqlctx.WithAuthContext(t.Context(), auth),
		proposal:  proposalID,
	}
	h.resolver = &QueryResolver{&base.Resolver{
		L:                      zap.NewNop(),
		PermissionEngine:       &resolvertest.PreviewPermissions{Granted: grants},
		AgentProposalService:   &resolvertest.PreviewProposals{Proposal: &agent.AgentProposal{ID: proposalID}},
		AgentDecisionService:   h.decisions,
		AgentPlanService:       h.plans,
		ProposalPreviewService: h.previews,
	}}

	return h
}

var assistantRead = permission.ResourceAssistant.String() + "|" + string(permission.OpRead)
