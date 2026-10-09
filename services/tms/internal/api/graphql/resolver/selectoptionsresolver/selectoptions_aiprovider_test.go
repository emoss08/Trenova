package selectoptionsresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAIProviderSelectService struct {
	services.AIProviderService
	byID       map[pulid.ID]*aiprovider.Provider
	selectReq  *repositories.AIProviderSelectOptionsRequest
	selectList []*aiprovider.Provider
	lookups    []repositories.GetAIProviderByIDRequest
}

func (f *fakeAIProviderSelectService) SelectOptions(
	_ context.Context,
	req *repositories.AIProviderSelectOptionsRequest,
) (*pagination.ListResult[*aiprovider.Provider], error) {
	f.selectReq = req
	return &pagination.ListResult[*aiprovider.Provider]{
		Items: f.selectList,
		Total: len(f.selectList),
	}, nil
}

func (f *fakeAIProviderSelectService) GetByID(
	_ context.Context,
	req repositories.GetAIProviderByIDRequest,
) (*aiprovider.Provider, error) {
	f.lookups = append(f.lookups, req)
	provider, ok := f.byID[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("AIProvider not found within your organization")
	}
	return provider, nil
}

type decidingPermissionEngine struct {
	services.PermissionEngine
	allowed bool
	checked []*services.PermissionCheckRequest
}

func (e *decidingPermissionEngine) Check(
	_ context.Context,
	req *services.PermissionCheckRequest,
) (*services.PermissionCheckResult, error) {
	e.checked = append(e.checked, req)
	return &services.PermissionCheckResult{Allowed: e.allowed}, nil
}

func aiProviderResolver(
	svc *fakeAIProviderSelectService,
	engine *decidingPermissionEngine,
) *QueryResolver {
	return &QueryResolver{&Deps{
		Core:              &base.Core{PermissionEngine: engine},
		AiProviderService: svc,
	}}
}

func TestSelectOptions_AIProviderListsAssistantChatProvidersForTheTenant(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	svc := &fakeAIProviderSelectService{
		selectList: []*aiprovider.Provider{{
			ID:        pulid.MustNew("aiprv_"),
			Name:      "Anthropic",
			Model:     "claude-opus-5-5",
			Kind:      aiprovider.KindAnthropicMessages,
			Enabled:   true,
			CreatedAt: 1780415883,
		}},
	}
	engine := &decidingPermissionEngine{allowed: true}
	ctx := gqlctx.WithAuthContext(t.Context(), testGraphQLAuthContext(orgID, buID, userID))

	result, err := aiProviderResolver(svc, engine).SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceAiProvider,
		Query:    stringutils.Ptr("opus"),
	})
	require.NoError(t, err)

	require.NotNil(t, svc.selectReq)
	assert.Equal(t, aiprovider.TaskAssistantChat, svc.selectReq.Task)
	assert.Equal(t, "opus", svc.selectReq.SelectQueryRequest.Query)
	assert.Equal(t, orgID, svc.selectReq.SelectQueryRequest.TenantInfo.OrgID)
	assert.Equal(t, buID, svc.selectReq.SelectQueryRequest.TenantInfo.BuID)
	require.Len(t, result.Edges, 1)
	assert.Equal(t, "Anthropic", result.Edges[0].Node.Label)
	require.NotNil(t, result.Edges[0].Node.Description)
	assert.Equal(t, "claude-opus-5-5", *result.Edges[0].Node.Description)
	assert.Equal(t, "claude-opus-5-5", result.Edges[0].Node.Meta["model"])
	require.Len(t, engine.checked, 1)
	assert.Equal(t, permission.ResourceAIProvider.String(), engine.checked[0].Resource)
	assert.Equal(t, permission.OpRead, engine.checked[0].Operation)
}

func TestSelectOptions_AIProviderRefusesSomeoneWhoCannotReadProviders(t *testing.T) {
	t.Parallel()

	svc := &fakeAIProviderSelectService{}
	engine := &decidingPermissionEngine{allowed: false}
	ctx := gqlctx.WithAuthContext(
		t.Context(),
		testGraphQLAuthContext(pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")),
	)

	_, err := aiProviderResolver(svc, engine).SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceAiProvider,
	})
	require.Error(t, err)
	assert.Nil(t, svc.selectReq)
	assert.Empty(t, svc.lookups)
}

// A saved choice still names its provider after it was switched off, and a
// provider deleted since is left out rather than failing the whole lookup.
func TestSelectOptions_AIProviderByIDsKeepsKnownProvidersInTheTenant(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	kept := &aiprovider.Provider{
		ID:      pulid.MustNew("aiprv_"),
		Name:    "Local Ollama",
		Model:   "llama3.3",
		Enabled: false,
	}
	gone := pulid.MustNew("aiprv_")
	svc := &fakeAIProviderSelectService{byID: map[pulid.ID]*aiprovider.Provider{kept.ID: kept}}
	ctx := gqlctx.WithAuthContext(
		t.Context(),
		testGraphQLAuthContext(orgID, buID, pulid.MustNew("usr_")),
	)

	result, err := aiProviderResolver(svc, &decidingPermissionEngine{allowed: true}).SelectOptions(
		ctx,
		gqlmodel.SelectOptionsInput{
			Resource: gqlmodel.SelectOptionResourceAiProvider,
			Ids:      []string{kept.ID.String(), gone.String()},
		},
	)
	require.NoError(t, err)

	require.Len(t, result.Edges, 1)
	assert.Equal(t, kept.ID.String(), result.Edges[0].Node.ID)
	assert.Equal(t, "Local Ollama", result.Edges[0].Node.Label)
	require.Len(t, svc.lookups, 2)
	for _, lookup := range svc.lookups {
		assert.Equal(t, orgID, lookup.TenantInfo.OrgID)
		assert.Equal(t, buID, lookup.TenantInfo.BuID)
	}
	assert.Nil(t, svc.selectReq)
}
