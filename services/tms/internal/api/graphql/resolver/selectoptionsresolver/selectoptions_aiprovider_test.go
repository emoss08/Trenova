package selectoptionsresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAIProviderSelectService struct {
	aiproviderports.AIProviderService
	byID       map[pulid.ID]*aiprovider.Provider
	selectReq  *aiproviderports.AIProviderSelectOptionsRequest
	selectList []*aiprovider.Provider
	lookups    []aiproviderports.GetAIProvidersByIDsRequest
}

func (f *fakeAIProviderSelectService) SelectOptions(
	_ context.Context,
	req *aiproviderports.AIProviderSelectOptionsRequest,
) (*pagination.ListResult[*aiprovider.Provider], error) {
	f.selectReq = req
	return &pagination.ListResult[*aiprovider.Provider]{
		Items:      f.selectList,
		Total:      len(f.selectList),
		TotalExact: true,
	}, nil
}

func (f *fakeAIProviderSelectService) GetByIDs(
	_ context.Context,
	req aiproviderports.GetAIProvidersByIDsRequest,
) ([]*aiprovider.Provider, error) {
	f.lookups = append(f.lookups, req)
	providers := make([]*aiprovider.Provider, 0, len(req.ProviderIDs))
	for _, id := range req.ProviderIDs {
		if provider, ok := f.byID[id]; ok {
			providers = append(providers, provider)
		}
	}
	return providers, nil
}

type decidingPermissionEngine struct {
	permissionports.PermissionEngine
	allowed bool
	checked []*permissionports.PermissionCheckRequest
}

func (e *decidingPermissionEngine) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	e.checked = append(e.checked, req)
	return &permissionports.PermissionCheckResult{Allowed: e.allowed}, nil
}

func aiProviderResolver(
	svc *fakeAIProviderSelectService,
	engine *decidingPermissionEngine,
) *QueryResolver {
	return &QueryResolver{&Deps{
		Core:              &resolvercore.Core{PermissionEngine: engine},
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
	assert.Equal(t, aitask.TaskAssistantChat, svc.selectReq.Task)
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
		Name:    "Local vLLM",
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
	assert.Equal(t, "Local vLLM", result.Edges[0].Node.Label)
	require.Len(t, svc.lookups, 1, "every selected provider is read in one batch")
	assert.Equal(t, []pulid.ID{kept.ID, gone}, svc.lookups[0].ProviderIDs)
	assert.Equal(t, orgID, svc.lookups[0].TenantInfo.OrgID)
	assert.Equal(t, buID, svc.lookups[0].TenantInfo.BuID)
	assert.Nil(t, svc.selectReq)
}
