package integrationresolver

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/integrationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/integrationports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationResolvers_CheckTheIntegrationPermission(t *testing.T) {
	t.Parallel()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"integrationCatalog", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).IntegrationCatalog(ctx)
			return err
		}},
		{"integrationConfig", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).IntegrationConfig(ctx, integration.TypeSamsara)
			return err
		}},
		{
			"samsaraWorkerSyncReadiness",
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).SamsaraWorkerSyncReadiness(ctx)
				return err
			},
		},
		{"samsaraWorkerSyncDrift", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).SamsaraWorkerSyncDrift(ctx)
			return err
		}},
		{
			"samsaraWorkerSyncRunStatus",
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).SamsaraWorkerSyncRunStatus(ctx, "wf_1", nil)
				return err
			},
		},
		{
			"updateIntegrationConfig",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).UpdateIntegrationConfig(
					ctx,
					gqlmodel.UpdateIntegrationConfigInput{Type: integration.TypeSamsara},
				)
				return err
			},
		},
		{
			"testIntegrationConnection",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).TestIntegrationConnection(
					ctx,
					integration.TypeSamsara,
				)
				return err
			},
		},
		{
			"startSamsaraWorkerSync",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).StartSamsaraWorkerSync(ctx)
				return err
			},
		},
		{
			"detectSamsaraWorkerSyncDrift",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).DetectSamsaraWorkerSyncDrift(ctx)
				return err
			},
		},
		{
			"repairSamsaraWorkerSyncDrift",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).RepairSamsaraWorkerSyncDrift(ctx, nil)
				return err
			},
		},
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
				permission.ResourceIntegration,
				call.operation,
			)
		})
	}
}

func TestIntegrationRuntimeConfig_RefusesAnAgent(t *testing.T) {
	t.Parallel()

	engine := resolvertest.RefusingEngine()
	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
	authCtx := resolvertest.UserAuthContext()
	authCtx.PrincipalType = authctx.PrincipalTypeAgent

	_, err := (&QueryResolver{deps}).IntegrationRuntimeConfig(
		resolvertest.AuthedContext(t, authCtx),
		integration.TypeGoogleMaps,
	)

	var authzErr *errortypes.AuthorizationError
	require.True(t, errors.As(err, &authzErr), "%v", err)
	assert.Empty(t, engine.Requests)
}

func TestIntegrationRuntimeConfig_NeedsASignedInCaller(t *testing.T) {
	t.Parallel()

	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: resolvertest.RefusingEngine()}}

	_, err := (&QueryResolver{deps}).IntegrationRuntimeConfig(
		t.Context(),
		integration.TypeGoogleMaps,
	)

	var authnErr *errortypes.AuthenticationError
	require.True(t, errors.As(err, &authnErr), "%v", err)
}

func TestConfigurationFromInput_KeysEverySetting(t *testing.T) {
	t.Parallel()

	configuration, err := configurationFromInput([]*gqlmodel.IntegrationConfigValueInput{
		{Key: "token", Value: "secret"},
		{Key: "webhookSecret", Value: ""},
	})

	require.NoError(t, err)
	assert.Equal(t, map[string]string{"token": "secret", "webhookSecret": ""}, configuration)
}

func TestConfigurationFromInput_RefusesASettingGivenTwice(t *testing.T) {
	t.Parallel()

	_, err := configurationFromInput([]*gqlmodel.IntegrationConfigValueInput{
		{Key: "token", Value: "a"},
		{Key: "token", Value: "b"},
	})

	var validationErr *errortypes.Error
	require.True(t, errors.As(err, &validationErr), "%v", err)
	assert.Equal(t, "configuration.token", validationErr.Field)
	assert.Equal(t, errortypes.ErrDuplicate, validationErr.Code)
}

func TestConfigurationFromInput_RefusesASettingWithoutAKey(t *testing.T) {
	t.Parallel()

	_, err := configurationFromInput([]*gqlmodel.IntegrationConfigValueInput{{Value: "a"}})

	var validationErr *errortypes.Error
	require.True(t, errors.As(err, &validationErr), "%v", err)
	assert.Equal(t, errortypes.ErrRequired, validationErr.Code)
}

func TestIntegrationRuntimeConfig_ConfigIsTheClientSafeSettings(t *testing.T) {
	t.Parallel()

	resolver := &IntegrationRuntimeConfigResolver{&Deps{}}

	config, err := resolver.Config(t.Context(), &integrationservice.RuntimeConfig{
		Config: map[string]string{"apiKey": "browser-key"},
	})

	require.NoError(t, err)
	assert.Equal(t, map[string]any{"apiKey": "browser-key"}, config)
}

func TestIntegrationCatalog_StatusFieldsAreTheirStrings(t *testing.T) {
	t.Parallel()

	deps := &Deps{}
	ctx := t.Context()

	category, err := (&IntegrationCatalogItemResolver{deps}).Category(
		ctx,
		&integrationports.CatalogItem{Category: integration.CategoryTelematics},
	)
	require.NoError(t, err)
	assert.Equal(t, "Telematics", category)

	connection, err := (&IntegrationCatalogStatusResolver{deps}).Connection(
		ctx,
		&integrationports.CatalogStatus{Connection: "connected"},
	)
	require.NoError(t, err)
	assert.Equal(t, "connected", connection)
}
