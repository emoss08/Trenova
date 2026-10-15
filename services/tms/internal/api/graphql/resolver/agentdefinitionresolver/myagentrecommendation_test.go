package agentdefinitionresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingRecommender struct {
	answer *agentports.AgentRecommendation
	asked  []*agentports.AgentRecommendationRequest
}

func (r *recordingRecommender) RecommendAgent(
	_ context.Context,
	req *agentports.AgentRecommendationRequest,
) (*agentports.AgentRecommendation, error) {
	r.asked = append(r.asked, req)

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

func TestMyAgentRecommendation_AsksForThePersonAndTheAgentChosenAndMapsTheAnswer(t *testing.T) {
	t.Parallel()

	suggested := &agentdefinition.Definition{ID: pulid.MustNew("agdef_"), Name: "Customer service"}
	current := pulid.MustNew("agdef_")
	recommender := &recordingRecommender{answer: &agentports.AgentRecommendation{
		Agent: suggested,
		Tools: []string{"draft_reply", "shipment_eta"},
	}}
	permissions := &assistantPermissions{allowed: true}
	resolver, ctx, userID := recommendationResolver(t, recommender, permissions)
	currentID := current.String()

	result, err := resolver.MyAgentRecommendation(ctx, gqlmodel.MyAgentRecommendationInput{
		Text:           "When will the Acme load arrive?",
		CurrentAgentID: &currentID,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Same(t, suggested, result.Agent)
	assert.False(t, result.Current)
	assert.Equal(t, []string{"draft_reply", "shipment_eta"}, result.Tools)

	require.Len(t, recommender.asked, 1)
	asked := recommender.asked[0]
	assert.Equal(t, "When will the Acme load arrive?", asked.Text)
	assert.Equal(t, current, asked.CurrentAgentID)
	require.NotNil(t, asked.Actor)
	assert.Equal(t, userID, asked.Actor.UserID)

	require.Len(t, permissions.checked, 1)
	assert.Equal(t, permission.ResourceAssistant.String(), permissions.checked[0].Resource)
	assert.Equal(t, permission.OpCreate, permissions.checked[0].Operation)
}

func TestMyAgentRecommendation_IsNullWhenNoAgentFits(t *testing.T) {
	t.Parallel()

	recommender := &recordingRecommender{}
	resolver, ctx, _ := recommendationResolver(t, recommender, &assistantPermissions{allowed: true})

	result, err := resolver.MyAgentRecommendation(ctx, gqlmodel.MyAgentRecommendationInput{
		Text: "When will the Acme load arrive?",
	})

	require.NoError(t, err)
	assert.Nil(t, result)
	require.Len(t, recommender.asked, 1)
	assert.True(t, recommender.asked[0].CurrentAgentID.IsNil())
}

func TestMyAgentRecommendation_RefusesSomeoneWhoMayNotStartAConversation(t *testing.T) {
	t.Parallel()

	recommender := &recordingRecommender{}
	resolver, ctx, _ := recommendationResolver(t, recommender, &assistantPermissions{allowed: false})

	result, err := resolver.MyAgentRecommendation(ctx, gqlmodel.MyAgentRecommendationInput{
		Text: "When will the Acme load arrive?",
	})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Empty(t, recommender.asked, "nothing is matched for someone refused")
}

func TestMyAgentRecommendation_RefusesAnAgentIDThatIsNotOne(t *testing.T) {
	t.Parallel()

	recommender := &recordingRecommender{}
	resolver, ctx, _ := recommendationResolver(t, recommender, &assistantPermissions{allowed: true})
	notAnID := "not-an-id"

	_, err := resolver.MyAgentRecommendation(ctx, gqlmodel.MyAgentRecommendationInput{
		Text:           "When will the Acme load arrive?",
		CurrentAgentID: &notAnID,
	})

	require.Error(t, err)
	assert.Empty(t, recommender.asked)
}
