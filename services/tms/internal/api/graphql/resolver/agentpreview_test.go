package resolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type previewPermissions struct {
	services.PermissionEngine

	granted map[string]bool
}

func (p *previewPermissions) Check(
	_ context.Context,
	req *services.PermissionCheckRequest,
) (*services.PermissionCheckResult, error) {
	return &services.PermissionCheckResult{
		Allowed: p.granted[req.Resource+"|"+string(req.Operation)],
	}, nil
}

type previewProposals struct {
	services.AgentProposalService

	proposal *agent.AgentProposal
}

func (p *previewProposals) GetByID(
	context.Context,
	repositories.GetAgentProposalByIDRequest,
) (*agent.AgentProposal, error) {
	return p.proposal, nil
}

type previewDecisions struct {
	services.AgentDecisionService

	yours bool
	asked []pulid.ID
}

func (d *previewDecisions) AssertOwnProposal(
	_ context.Context,
	proposalID pulid.ID,
	_ pagination.TenantInfo,
	_ *services.RequestActor,
) error {
	d.asked = append(d.asked, proposalID)
	if !d.yours {
		return errortypes.NewNotFoundError("That proposal was not raised in one of your conversations")
	}

	return nil
}

type previewPlans struct {
	services.AgentPlanService

	yours bool
}

func (p *previewPlans) AssertOwnPlan(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
	*services.RequestActor,
) error {
	if !p.yours {
		return errortypes.NewNotFoundError("That plan was not raised in one of your conversations")
	}

	return nil
}

func (p *previewPlans) GetByID(
	_ context.Context,
	req repositories.GetAgentPlanByIDRequest,
) (*agent.AgentPlan, error) {
	return &agent.AgentPlan{ID: req.ID}, nil
}

type previewService struct {
	services.ProposalPreviewService

	proposals int
	plans     int
	viewer    *services.PreviewViewer
	mods      map[string]any
}

func (s *previewService) ForProposal(
	_ context.Context,
	req *services.ProposalPreviewRequest,
) (*agent.ProposalPreview, error) {
	s.proposals++
	s.viewer = req.Viewer
	s.mods = req.Modifications

	return &agent.ProposalPreview{ProposalID: req.Proposal.ID}, nil
}

func (s *previewService) ForPlan(
	_ context.Context,
	req *services.PlanPreviewRequest,
) (*agent.PlanPreview, error) {
	s.plans++

	return &agent.PlanPreview{PlanID: req.Plan.ID}, nil
}

type previewHarness struct {
	resolver  *queryResolver
	previews  *previewService
	decisions *previewDecisions
	plans     *previewPlans
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
		previews:  &previewService{},
		decisions: &previewDecisions{},
		plans:     &previewPlans{},
		ctx:       gqlctx.WithAuthContext(t.Context(), auth),
		proposal:  proposalID,
	}
	h.resolver = &queryResolver{&Resolver{
		l:                      zap.NewNop(),
		permissionEngine:       &previewPermissions{granted: grants},
		agentProposalService:   &previewProposals{proposal: &agent.AgentProposal{ID: proposalID}},
		agentDecisionService:   h.decisions,
		agentPlanService:       h.plans,
		proposalPreviewService: h.previews,
	}}

	return h
}

var (
	proposalRead  = permission.ResourceAgentProposal.String() + "|" + string(permission.OpRead)
	assistantRead = permission.ResourceAssistant.String() + "|" + string(permission.OpRead)
)

func TestAgentProposalPreview_NeedsProposalRead(t *testing.T) {
	t.Parallel()

	denied := newPreviewHarness(t)
	_, err := denied.resolver.AgentProposalPreview(denied.ctx, denied.proposal.String(), nil)
	require.Error(t, err)
	assert.Zero(t, denied.previews.proposals, "nothing is previewed for a reader without access")

	allowed := newPreviewHarness(t, proposalRead)
	mods := map[string]any{"message": "Call dispatch"}
	preview, err := allowed.resolver.AgentProposalPreview(allowed.ctx, allowed.proposal.String(), mods)
	require.NoError(t, err)
	assert.Equal(t, allowed.proposal, preview.ProposalID)
	assert.Equal(t, mods, allowed.previews.mods)
	require.NotNil(t, allowed.previews.viewer)
	assert.NotNil(t, allowed.previews.viewer.Reads, "reads are checked for this reader")

	_, err = allowed.resolver.AgentPlanPreview(allowed.ctx, pulid.MustNew("apl_").String())
	require.NoError(t, err)
	_, err = denied.resolver.AgentPlanPreview(denied.ctx, pulid.MustNew("apl_").String())
	require.Error(t, err)
}

func TestMyProposalPreview_IsTheCallersOwnAlone(t *testing.T) {
	t.Parallel()

	withoutAssistant := newPreviewHarness(t, proposalRead)
	_, err := withoutAssistant.resolver.MyProposalPreview(
		withoutAssistant.ctx, withoutAssistant.proposal.String(), nil)
	require.Error(t, err, "the self-scoped query needs assistant:read")

	notYours := newPreviewHarness(t, assistantRead)
	_, err = notYours.resolver.MyProposalPreview(notYours.ctx, notYours.proposal.String(), nil)
	require.True(t, errortypes.IsNotFoundError(err), "someone else's proposal is not found")
	assert.Equal(t, []pulid.ID{notYours.proposal}, notYours.decisions.asked)
	assert.Zero(t, notYours.previews.proposals)

	yours := newPreviewHarness(t, assistantRead)
	yours.decisions.yours = true
	_, err = yours.resolver.MyProposalPreview(yours.ctx, yours.proposal.String(), nil)
	require.NoError(t, err)
	assert.Equal(t, 1, yours.previews.proposals)
}

func TestMyPlanPreview_IsTheCallersOwnAlone(t *testing.T) {
	t.Parallel()

	notYours := newPreviewHarness(t, assistantRead)
	_, err := notYours.resolver.MyPlanPreview(notYours.ctx, pulid.MustNew("apl_").String())
	require.True(t, errortypes.IsNotFoundError(err))
	assert.Zero(t, notYours.previews.plans)

	yours := newPreviewHarness(t, assistantRead)
	yours.plans.yours = true
	_, err = yours.resolver.MyPlanPreview(yours.ctx, pulid.MustNew("apl_").String())
	require.NoError(t, err)
	assert.Equal(t, 1, yours.previews.plans)
}
