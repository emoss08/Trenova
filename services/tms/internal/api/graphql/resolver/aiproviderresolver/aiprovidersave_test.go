package aiproviderresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/infrastructure/config/aiconfig"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiplatform"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubProviders struct {
	aiproviderports.AIProviderService

	created []*aiproviderports.SaveAIProviderRequest
	updated []*aiproviderports.SaveAIProviderRequest
	deleted []aiproviderports.DeleteAIProviderRequest
	tested  []aiproviderports.GetAIProviderByIDRequest
	actors  []*permissionports.RequestActor
}

func (s *stubProviders) Create(
	_ context.Context,
	req *aiproviderports.SaveAIProviderRequest,
	actor *permissionports.RequestActor,
) (*aiprovider.Provider, error) {
	s.created = append(s.created, req)
	s.actors = append(s.actors, actor)
	return &aiprovider.Provider{ID: pulid.MustNew("aip_"), Name: req.Name}, nil
}

func (s *stubProviders) Update(
	_ context.Context,
	req *aiproviderports.SaveAIProviderRequest,
	actor *permissionports.RequestActor,
) (*aiprovider.Provider, error) {
	s.updated = append(s.updated, req)
	s.actors = append(s.actors, actor)
	return &aiprovider.Provider{ID: req.ID, Name: req.Name}, nil
}

func (s *stubProviders) Delete(
	_ context.Context,
	req aiproviderports.DeleteAIProviderRequest,
	actor *permissionports.RequestActor,
) error {
	s.deleted = append(s.deleted, req)
	s.actors = append(s.actors, actor)
	return nil
}

func (s *stubProviders) Test(
	_ context.Context,
	req aiproviderports.GetAIProviderByIDRequest,
) (*aiproviderports.TestAIProviderResult, error) {
	s.tested = append(s.tested, req)
	return &aiproviderports.TestAIProviderResult{
		Success:   true,
		Message:   "Answered",
		LatencyMS: 120,
	}, nil
}

func saveInput() gqlmodel.SaveAIProviderInput {
	return gqlmodel.SaveAIProviderInput{
		Name:                 "Claude",
		Kind:                 aiprovider.KindAnthropicMessages,
		BaseURL:              "https://api.anthropic.com",
		Model:                "claude-opus-5-5",
		StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
		Platform:             aiplatform.PlatformDirect,
		EmbeddingInputStyle:  aiprovider.EmbeddingInputStyleNone,
		OnCap:                aiprovider.CapActionNext,
		MaxTokens:            8192,
		Tasks:                []aitask.Task{aitask.TaskAssistantChat},
	}
}

func TestAIProviderAdmin_ChecksTheAIProviderPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("aip_").String()
	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"aiProviderCatalog", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).AiProviderCatalog(ctx)
			return err
		}},
		{"createAIProvider", permission.OpCreate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CreateAIProvider(ctx, saveInput())
			return err
		}},
		{"updateAIProvider", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateAIProvider(ctx, id, saveInput())
			return err
		}},
		{"deleteAIProvider", permission.OpDelete, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).DeleteAIProvider(ctx, id)
			return err
		}},
		{"testAIProvider", permission.OpManage, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).TestAIProvider(ctx, id)
			return err
		}},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			err := call.run(ctx, deps)

			resolvertest.RequireRefused(
				t,
				err,
				engine,
				permission.ResourceAIProvider,
				call.operation,
			)
		})
	}
}

func TestSaveRequestFromInput_MapsTheEditorsInputWithoutTouchingAnAbsentKey(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	input := saveInput()
	inputCost, monthlyCap := "15", "250.00"
	input.InputCostPerMillion = &inputCost
	input.MonthlyCapUsd = &monthlyCap
	region := "us-east-1"
	input.PlatformSettings = &gqlmodel.AIProviderPlatformSettingsInput{Region: &region}
	input.ExtraBody = map[string]any{"top_k": 5}
	input.Version = 3

	req, err := saveRequestFromInput(pulid.Nil, &input, tenant)
	require.NoError(t, err)

	assert.Equal(t, tenant, req.TenantInfo)
	assert.Nil(t, req.APIKey, "an absent key keeps the stored credential")
	require.NotNil(t, req.InputCostPerMillion)
	assert.True(t, decimal.RequireFromString("15").Equal(*req.InputCostPerMillion))
	assert.Nil(t, req.OutputCostPerMillion, "an absent price stays unknown")
	require.NotNil(t, req.MonthlyCapUSD)
	assert.True(t, decimal.RequireFromString("250").Equal(*req.MonthlyCapUSD))
	require.NotNil(t, req.PlatformSettings)
	assert.Equal(t, "us-east-1", req.PlatformSettings.Region)
	assert.Equal(t, map[string]any{"top_k": 5}, req.ExtraBody)
	assert.Equal(t, []aitask.Task{aitask.TaskAssistantChat}, req.Tasks)
	assert.Equal(t, int64(3), req.Version)
}

func TestSaveRequestFromInput_CarriesANewKeyAndAnEmptyOneThatClearsIt(t *testing.T) {
	t.Parallel()

	input := saveInput()
	key := "sk-ant-new"
	input.APIKey = &key
	req, err := saveRequestFromInput(pulid.Nil, &input, pagination.TenantInfo{})
	require.NoError(t, err)
	require.NotNil(t, req.APIKey)
	assert.Equal(t, "sk-ant-new", *req.APIKey)

	cleared := ""
	input.APIKey = &cleared
	req, err = saveRequestFromInput(pulid.Nil, &input, pagination.TenantInfo{})
	require.NoError(t, err)
	require.NotNil(t, req.APIKey)
	assert.Empty(t, *req.APIKey)
}

func TestSaveRequestFromInput_NamesAMalformedPrice(t *testing.T) {
	t.Parallel()

	input := saveInput()
	price := "cheap"
	input.CacheReadCostPerMillion = &price

	_, err := saveRequestFromInput(pulid.Nil, &input, pagination.TenantInfo{})

	var invalid *errortypes.Error
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, "cacheReadCostPerMillion", invalid.Field)
}

func TestCreateAIProvider_SavesAsTheCallerInTheirTenant(t *testing.T) {
	t.Parallel()

	providers := &stubProviders{}
	deps := &Deps{
		Core:              &resolvercore.Core{PermissionEngine: &mocks.AllowAllPermissionEngine{}},
		AiProviderService: providers,
	}
	authCtx := resolvertest.UserAuthContext()
	ctx := resolvertest.AuthedContext(t, authCtx)

	created, err := (&MutationResolver{deps}).CreateAIProvider(ctx, saveInput())
	require.NoError(t, err)

	require.Len(t, providers.created, 1)
	assert.True(t, providers.created[0].ID.IsNil())
	assert.Equal(t, authCtx.OrganizationID, providers.created[0].TenantInfo.OrgID)
	assert.Equal(t, authCtx.UserID, providers.actors[0].UserID)
	assert.Equal(t, "Claude", created.Name)
}

func TestUpdateAndDeleteAIProvider_NameTheProvider(t *testing.T) {
	t.Parallel()

	providers := &stubProviders{}
	deps := &Deps{
		Core:              &resolvercore.Core{PermissionEngine: &mocks.AllowAllPermissionEngine{}},
		AiProviderService: providers,
	}
	ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())
	id := pulid.MustNew("aip_")

	_, err := (&MutationResolver{deps}).UpdateAIProvider(ctx, id.String(), saveInput())
	require.NoError(t, err)
	deleted, err := (&MutationResolver{deps}).DeleteAIProvider(ctx, id.String())
	require.NoError(t, err)

	assert.Equal(t, id, providers.updated[0].ID)
	assert.True(t, deleted)
	assert.Equal(t, id, providers.deleted[0].ID)
}

func TestTestAIProvider_ProbesTheProviderAndReturnsTheResult(t *testing.T) {
	t.Parallel()

	providers := &stubProviders{}
	deps := &Deps{
		Core:              &resolvercore.Core{PermissionEngine: &mocks.AllowAllPermissionEngine{}},
		AiProviderService: providers,
	}
	authCtx := resolvertest.UserAuthContext()
	ctx := resolvertest.AuthedContext(t, authCtx)
	id := pulid.MustNew("aip_")

	result, err := (&MutationResolver{deps}).TestAIProvider(ctx, id.String())
	require.NoError(t, err)

	require.Len(t, providers.tested, 1)
	assert.Equal(t, id, providers.tested[0].ID)
	assert.Equal(t, authCtx.OrganizationID, providers.tested[0].TenantInfo.OrgID)
	assert.True(t, result.Success)
	assert.Equal(t, int64(120), result.LatencyMS)
}

func TestAIProviderCatalog_OffersServerCredentialsOnlyWhereTheServerAllowsThem(t *testing.T) {
	t.Parallel()

	ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())
	engine := &mocks.AllowAllPermissionEngine{}

	withoutConfig, err := (&QueryResolver{&Deps{
		Core: &resolvercore.Core{PermissionEngine: engine},
	}}).AiProviderCatalog(ctx)
	require.NoError(t, err)
	assert.False(t, withoutConfig.ServerCloudCredentials)
	assert.NotEmpty(t, withoutConfig.Kinds)
	assert.NotEmpty(t, withoutConfig.Tasks)

	allowed, err := (&QueryResolver{&Deps{
		Core:     &resolvercore.Core{PermissionEngine: engine},
		AIConfig: &aiconfig.Config{ServerCloudCredentials: true},
	}}).AiProviderCatalog(ctx)
	require.NoError(t, err)
	assert.True(t, allowed.ServerCloudCredentials)
}
