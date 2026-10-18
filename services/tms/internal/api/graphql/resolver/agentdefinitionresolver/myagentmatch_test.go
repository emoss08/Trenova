package agentdefinitionresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentsuggestion"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingRecommender struct {
	answer *agentports.AgentMatch
	asked  []*agentports.AgentMatchRequest
}

func (r *recordingRecommender) MatchAgent(
	_ context.Context,
	req *agentports.AgentMatchRequest,
) (*agentports.AgentMatch, error) {
	r.asked = append(r.asked, req)
	if r.answer == nil {
		return &agentports.AgentMatch{Outcome: agentsuggestion.MatchNone}, nil
	}

	return r.answer, nil
}

type assistantPermissions struct {
	permissionports.PermissionEngine
	allowed bool
	checked []*permissionports.PermissionCheckRequest
}

func (e *assistantPermissions) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	e.checked = append(e.checked, req)

	return &permissionports.PermissionCheckResult{Allowed: e.allowed}, nil
}

func recommendationResolver(
	t *testing.T,
	recommender *recordingRecommender,
	permissions *assistantPermissions,
) (*QueryResolver, context.Context, pulid.ID) {
	t.Helper()

	userID := pulid.MustNew("usr_")
	ctx := gqlctx.WithAuthContext(t.Context(), &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})

	return &QueryResolver{&Deps{
		Core:             &base.Core{PermissionEngine: permissions},
		AgentRecommender: recommender,
	}}, ctx, userID
}

func TestMyAgentMatch_AsksForThePersonTheAgentChosenAndTheContextAndMapsTheAnswer(t *testing.T) {
	t.Parallel()

	suggested := &agentdefinition.Definition{ID: pulid.MustNew("agdef_"), Name: "Customer service"}
	current := pulid.MustNew("agdef_")
	recommender := &recordingRecommender{answer: &agentports.AgentMatch{
		Outcome: agentsuggestion.MatchBetterAgent,
		Agent:   suggested,
		Tools: []agentports.AgentMatchTool{
			{Name: "draft_reply", Summary: "Draft an email reply to a customer."},
			{Name: "shipment_eta", Summary: "A shipment's projected arrival."},
		},
	}}
	permissions := &assistantPermissions{allowed: true}
	resolver, ctx, userID := recommendationResolver(t, recommender, permissions)
	currentID := current.String()

	result, err := resolver.MyAgentMatch(ctx, gqlmodel.MyAgentMatchInput{
		Text:           "When will the Acme load arrive?",
		CurrentAgentID: &currentID,
		ContextKinds:   []string{"shipment", "customer"},
	})

	require.NoError(t, err)
	assert.Equal(t, &gqlmodel.MyAgentMatch{
		Outcome: agentsuggestion.MatchBetterAgent,
		Agent:   suggested,
		Tools: []*gqlmodel.AgentMatchTool{
			{Name: "draft_reply", Summary: "Draft an email reply to a customer."},
			{Name: "shipment_eta", Summary: "A shipment's projected arrival."},
		},
	}, result)

	require.Len(t, recommender.asked, 1)
	asked := recommender.asked[0]
	assert.Equal(t, "When will the Acme load arrive?", asked.Text)
	assert.Equal(t, current, asked.CurrentAgentID)
	assert.Equal(t, []string{"shipment", "customer"}, asked.ContextKinds)
	require.NotNil(t, asked.Actor)
	assert.Equal(t, userID, asked.Actor.UserID)

	require.Len(t, permissions.checked, 1)
	assert.Equal(t, permission.ResourceAssistant.String(), permissions.checked[0].Resource)
	assert.Equal(t, permission.OpCreate, permissions.checked[0].Operation)
}

func TestMyAgentMatch_SaysWhenNoAgentCanHelp(t *testing.T) {
	t.Parallel()

	recommender := &recordingRecommender{answer: &agentports.AgentMatch{
		Outcome: agentsuggestion.MatchNoAgent,
		Tools:   []agentports.AgentMatchTool{{Name: "list_invoices", Summary: "List invoices."}},
	}}
	resolver, ctx, _ := recommendationResolver(t, recommender, &assistantPermissions{allowed: true})

	result, err := resolver.MyAgentMatch(ctx, gqlmodel.MyAgentMatchInput{Text: "invoices past due"})

	require.NoError(t, err)
	assert.Equal(t, &gqlmodel.MyAgentMatch{
		Outcome: agentsuggestion.MatchNoAgent,
		Tools:   []*gqlmodel.AgentMatchTool{{Name: "list_invoices", Summary: "List invoices."}},
	}, result)
	assert.True(t, recommender.asked[0].CurrentAgentID.IsNil())
}

func TestMyAgentMatch_RefusesSomeoneWhoMayNotStartAConversation(t *testing.T) {
	t.Parallel()

	recommender := &recordingRecommender{}
	resolver, ctx, _ := recommendationResolver(t, recommender, &assistantPermissions{allowed: false})

	result, err := resolver.MyAgentMatch(ctx, gqlmodel.MyAgentMatchInput{Text: "When will it arrive?"})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Empty(t, recommender.asked)
}

func TestMyAgentMatch_RefusesAnAgentIDThatIsNotOne(t *testing.T) {
	t.Parallel()

	recommender := &recordingRecommender{}
	resolver, ctx, _ := recommendationResolver(t, recommender, &assistantPermissions{allowed: true})
	notAnID := "not-an-id"

	_, err := resolver.MyAgentMatch(ctx, gqlmodel.MyAgentMatchInput{
		Text:           "When will the Acme load arrive?",
		CurrentAgentID: &notAnID,
	})

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "currentAgentId", validation.Field)
	assert.Equal(t, errortypes.ErrInvalid, validation.Code)
	assert.Empty(t, recommender.asked)
}
