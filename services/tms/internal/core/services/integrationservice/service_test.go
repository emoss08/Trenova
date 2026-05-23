package integrationservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const expectedCatalogItems = 5

type stubIntegrationRepo struct {
	listByTenantResult []*integration.Integration
	listByTenantErr    error
	getByTypeResult    *integration.Integration
	getByTypeErr       error
}

func (s *stubIntegrationRepo) ListByTenant(
	_ context.Context,
	_ pagination.TenantInfo,
) ([]*integration.Integration, error) {
	return s.listByTenantResult, s.listByTenantErr
}

func (s *stubIntegrationRepo) ListEnabledByType(
	_ context.Context,
	_ integration.Type,
) ([]*integration.Integration, error) {
	return nil, nil
}

func (s *stubIntegrationRepo) GetByType(
	_ context.Context,
	_ pagination.TenantInfo,
	_ integration.Type,
) (*integration.Integration, error) {
	return s.getByTypeResult, s.getByTypeErr
}

func (s *stubIntegrationRepo) Upsert(
	_ context.Context,
	_ *integration.Integration,
) (*integration.Integration, error) {
	return nil, nil
}

func TestListCatalogIncludesBackendCardMetadata(t *testing.T) {
	t.Parallel()

	repo := &stubIntegrationRepo{
		listByTenantResult: []*integration.Integration{
			{
				Type:    integration.TypeSamsara,
				Enabled: true,
				Configuration: map[string]any{
					"token": "encrypted-token",
				},
			},
		},
	}

	svc := New(Params{
		Logger: zap.NewNop(),
		Repo:   repo,
	})

	resp, err := svc.ListCatalog(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)
	require.Len(t, resp.Items, expectedCatalogItems)

	samsara := resp.Items[0]
	require.Equal(t, integration.TypeSamsara, samsara.Type)
	require.Contains(t, samsara.LogoURL, "samsara")
	require.Contains(t, samsara.LogoLightURL, "samsara")
	require.Contains(t, samsara.LogoDarkURL, "samsara")
	require.Equal(t, "Telematics", samsara.CategoryLabel)
	require.Equal(t, "View Integration", samsara.PrimaryActionLabel)
	require.Equal(t, "connected", string(samsara.Status.Connection))
	require.Equal(t, "configured", string(samsara.Status.Configuration))
	require.NotEmpty(t, samsara.Links)
	require.Equal(t, "docs", string(samsara.Links[0].Kind))
	require.NotEmpty(t, samsara.DocsURL)
	require.NotEmpty(t, samsara.WebsiteURL)
	require.NotEmpty(t, samsara.ConfigSpec)
	require.True(t, samsara.SupportsTestConnect)
}

func TestListCatalogSamsaraConfiguredFalseWithoutToken(t *testing.T) {
	t.Parallel()

	repo := &stubIntegrationRepo{
		listByTenantResult: []*integration.Integration{
			{
				Type:          integration.TypeSamsara,
				Enabled:       true,
				Configuration: map[string]any{},
			},
		},
	}

	svc := New(Params{
		Logger: zap.NewNop(),
		Repo:   repo,
	})

	resp, err := svc.ListCatalog(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)
	require.Len(t, resp.Items, expectedCatalogItems)
	require.False(t, resp.Items[0].Configured)
	require.Equal(t, "needs_setup", string(resp.Items[0].Status.Configuration))
}

func TestListCatalogSortedBySortOrderThenName(t *testing.T) {
	t.Parallel()

	repo := &stubIntegrationRepo{}
	svc := New(Params{
		Logger: zap.NewNop(),
		Repo:   repo,
	})

	resp, err := svc.ListCatalog(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)
	require.Len(t, resp.Items, expectedCatalogItems)
	require.Equal(t, integration.TypeSamsara, resp.Items[0].Type)
	require.Equal(t, integration.TypeGoogleMaps, resp.Items[1].Type)
	require.Equal(t, integration.TypeOpenWeatherMap, resp.Items[2].Type)
	require.Equal(t, integration.TypeOpenAI, resp.Items[3].Type)
	require.Equal(t, integration.TypeExchangeRateAPI, resp.Items[4].Type)
}

func TestGetClientRuntimeConfigAllowsBrowserSafeIntegrations(t *testing.T) {
	t.Parallel()

	encryption := encryptionservice.New(encryptionservice.Params{Config: &config.Config{}})
	repo := &stubIntegrationRepo{
		getByTypeResult: &integration.Integration{
			Type:    integration.TypeGoogleMaps,
			Enabled: true,
			Configuration: map[string]any{
				"apiKey": "browser-map-key",
			},
		},
	}
	svc := New(Params{
		Logger:     zap.NewNop(),
		Repo:       repo,
		Encryption: encryption,
	})

	resp, err := svc.GetClientRuntimeConfig(
		t.Context(),
		pagination.TenantInfo{},
		integration.TypeGoogleMaps,
	)

	require.NoError(t, err)
	require.Equal(t, map[string]string{"apiKey": "browser-map-key"}, resp.Config)
}

func TestGetClientRuntimeConfigRejectsServerSideIntegrations(t *testing.T) {
	t.Parallel()

	svc := New(Params{
		Logger: zap.NewNop(),
		Repo:   &stubIntegrationRepo{},
	})

	resp, err := svc.GetClientRuntimeConfig(
		t.Context(),
		pagination.TenantInfo{},
		integration.TypeSamsara,
	)

	require.Nil(t, resp)
	require.Error(t, err)
	require.Contains(t, err.Error(), "runtime configuration is not available to clients")
}
