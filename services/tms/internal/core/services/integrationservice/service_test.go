package integrationservice

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const expectedCatalogItems = 11

type stubIntegrationRepo struct {
	listByTenantResult []*integration.Integration
	listByTenantErr    error
	getByTypeResult    *integration.Integration
	getByTypeErr       error
	upsertResult       *integration.Integration
	upsertErr          error
}

type stubAuditService struct{}

func (s *stubAuditService) List(
	context.Context,
	*repositories.ListAuditEntriesRequest,
) (*pagination.ListResult[*audit.Entry], error) {
	return nil, nil
}

func (s *stubAuditService) ListByResourceID(
	context.Context,
	*repositories.ListByResourceIDRequest,
) (*pagination.ListResult[*audit.Entry], error) {
	return nil, nil
}

func (s *stubAuditService) GetByID(
	context.Context,
	repositories.GetAuditEntryByIDOptions,
) (*audit.Entry, error) {
	return nil, nil
}

func (s *stubAuditService) LogAction(*services.LogActionParams, ...services.LogOption) error {
	return nil
}

func (s *stubAuditService) LogActions([]services.BulkLogEntry) error {
	return nil
}

func (s *stubAuditService) RegisterSensitiveFields(
	permission.Resource,
	[]services.SensitiveField,
) error {
	return nil
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
	entity *integration.Integration,
) (*integration.Integration, error) {
	if s.upsertErr != nil {
		return nil, s.upsertErr
	}
	if s.upsertResult != nil {
		return s.upsertResult, nil
	}
	return entity, nil
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

	samsara := findCatalogItem(t, resp.Items, integration.TypeSamsara)
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
	samsara := findCatalogItem(t, resp.Items, integration.TypeSamsara)
	require.False(t, samsara.Configured)
	require.Equal(t, "needs_setup", string(samsara.Status.Configuration))
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
	require.Equal(t, integration.TypeResend, resp.Items[0].Type)
	require.Equal(t, integration.TypePostmark, resp.Items[1].Type)
	require.Equal(t, integration.TypeSamsara, resp.Items[2].Type)
	require.Equal(t, integration.TypeGoogleMaps, resp.Items[3].Type)
	require.Equal(t, integration.TypePCMiler, resp.Items[4].Type)
	require.Equal(t, integration.TypeOpenWeatherMap, resp.Items[5].Type)
	require.Equal(t, integration.TypeOpenAI, resp.Items[6].Type)
	require.Equal(t, integration.TypeOANDAExchangeRates, resp.Items[7].Type)
	require.Equal(t, integration.TypeAmazonSES, resp.Items[8].Type)
	require.Equal(t, integration.TypeSendGrid, resp.Items[9].Type)
	require.Equal(t, integration.TypeMailgun, resp.Items[10].Type)
}

func TestListCatalogIncludesPlannedEmailProviderLogos(t *testing.T) {
	t.Parallel()

	repo := &stubIntegrationRepo{}
	svc := New(Params{
		Logger: zap.NewNop(),
		Repo:   repo,
	})

	resp, err := svc.ListCatalog(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)
	require.Len(t, resp.Items, expectedCatalogItems)

	amazonSES := findCatalogItem(t, resp.Items, integration.TypeAmazonSES)
	require.Equal(t, "/integrations/logos/aws_light.svg", amazonSES.LogoURL)
	require.Equal(t, "/integrations/logos/aws_light.svg", amazonSES.LogoLightURL)
	require.Equal(t, "/integrations/logos/aws_dark.svg", amazonSES.LogoDarkURL)

	sendGrid := findCatalogItem(t, resp.Items, integration.TypeSendGrid)
	require.Equal(t, "/integrations/logos/sendgrid_light.svg", sendGrid.LogoURL)
	require.Equal(t, "/integrations/logos/sendgrid_light.svg", sendGrid.LogoLightURL)
	require.Equal(t, "/integrations/logos/sendgrid_dark.svg", sendGrid.LogoDarkURL)

	mailgun := findCatalogItem(t, resp.Items, integration.TypeMailgun)
	require.Equal(t, "/integrations/logos/mailgun_light.svg", mailgun.LogoURL)
	require.Equal(t, "/integrations/logos/mailgun_light.svg", mailgun.LogoLightURL)
	require.Equal(t, "/integrations/logos/mailgun_dark.svg", mailgun.LogoDarkURL)

	postmark := findCatalogItem(t, resp.Items, integration.TypePostmark)
	require.Equal(t, "/integrations/logos/postmark_all.png", postmark.LogoURL)
	require.Equal(t, "/integrations/logos/postmark_all.png", postmark.LogoLightURL)
	require.Equal(t, "/integrations/logos/postmark_all.png", postmark.LogoDarkURL)
}

func TestUpdateResendConfigAllowsEnabledWithoutWebhookSecret(t *testing.T) {
	t.Parallel()

	encryption := testEncryptionService()
	repo := &stubIntegrationRepo{
		getByTypeErr: errortypes.NewNotFoundError("integration not found"),
	}
	svc := New(Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		Encryption:   encryption,
		AuditService: &stubAuditService{},
	})

	resp, err := svc.UpdateConfig(
		t.Context(),
		pagination.TenantInfo{},
		integration.TypeResend,
		&services.UpdateConfigRequest{
			Enabled: true,
			Configuration: map[string]string{
				"apiKey":  "re_test_key",
				"baseUrl": "https://api.resend.com",
			},
		},
		"",
	)

	require.NoError(t, err)
	require.True(t, resp.Enabled)
	tokenField := findConfigField(t, resp.Fields, "webhookToken")
	require.NotEmpty(t, tokenField.Value)
	require.True(t, strings.HasPrefix(tokenField.Value, "resend_"))
}

func TestUpdatePostmarkConfigGeneratesWebhookToken(t *testing.T) {
	t.Parallel()

	encryption := testEncryptionService()
	repo := &stubIntegrationRepo{
		getByTypeErr: errortypes.NewNotFoundError("integration not found"),
	}
	svc := New(Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		Encryption:   encryption,
		AuditService: &stubAuditService{},
	})

	resp, err := svc.UpdateConfig(
		t.Context(),
		pagination.TenantInfo{},
		integration.TypePostmark,
		&services.UpdateConfigRequest{
			Enabled: true,
			Configuration: map[string]string{
				"serverToken":   "postmark-token",
				"baseUrl":       "https://api.postmarkapp.com",
				"messageStream": "outbound",
			},
		},
		"",
	)

	require.NoError(t, err)
	require.True(t, resp.Enabled)
	tokenField := findConfigField(t, resp.Fields, "webhookToken")
	require.NotEmpty(t, tokenField.Value)
	require.True(t, strings.HasPrefix(tokenField.Value, "postmark_"))
}

func findConfigField(
	t *testing.T,
	fields []services.ConfigFieldValue,
	key string,
) services.ConfigFieldValue {
	t.Helper()

	for _, field := range fields {
		if field.Key == key {
			return field
		}
	}

	require.Failf(t, "config field not found", "key %s was not present", key)
	return services.ConfigFieldValue{}
}

func findCatalogItem(
	t *testing.T,
	items []services.CatalogItem,
	typ integration.Type,
) services.CatalogItem {
	t.Helper()

	for _, item := range items {
		if item.Type == typ {
			return item
		}
	}

	require.Failf(t, "catalog item not found", "type %s was not present", typ)
	return services.CatalogItem{}
}

func TestGetClientRuntimeConfigAllowsBrowserSafeIntegrations(t *testing.T) {
	t.Parallel()

	encryption := testEncryptionService()
	apiKey, err := encryption.EncryptString("browser-map-key")
	require.NoError(t, err)
	repo := &stubIntegrationRepo{
		getByTypeResult: &integration.Integration{
			Type:    integration.TypeGoogleMaps,
			Enabled: true,
			Configuration: map[string]any{
				"apiKey": apiKey,
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
	require.True(t, resp.Enabled)
	require.True(t, resp.Configured)
	require.True(t, resp.Ready)
	require.Empty(t, resp.MissingRequiredFields)
	require.Equal(t, map[string]string{"apiKey": "browser-map-key"}, resp.Config)
}

func testEncryptionService() *encryptionservice.Service {
	return encryptionservice.New(encryptionservice.Params{
		Config: &config.Config{
			Security: config.SecurityConfig{
				Encryption: config.EncryptionConfig{
					Key: "unit-test-encryption-key-with-at-least-32-bytes",
				},
			},
		},
	})
}

func TestGetClientRuntimeConfigReturnsReadinessForUnconfiguredIntegration(t *testing.T) {
	t.Parallel()

	svc := New(Params{
		Logger: zap.NewNop(),
		Repo: &stubIntegrationRepo{
			getByTypeErr: errortypes.NewNotFoundError("integration not found"),
		},
	})

	resp, err := svc.GetClientRuntimeConfig(
		t.Context(),
		pagination.TenantInfo{},
		integration.TypeOANDAExchangeRates,
	)

	require.NoError(t, err)
	require.False(t, resp.Enabled)
	require.False(t, resp.Configured)
	require.False(t, resp.Ready)
	require.Equal(t, []string{"apiKey"}, resp.MissingRequiredFields)
	require.Empty(t, resp.Config)
}

func TestGetClientRuntimeConfigReturnsDisabledReadinessWithoutConfig(t *testing.T) {
	t.Parallel()

	svc := New(Params{
		Logger: zap.NewNop(),
		Repo: &stubIntegrationRepo{
			getByTypeResult: &integration.Integration{
				Type:    integration.TypeOANDAExchangeRates,
				Enabled: false,
				Configuration: map[string]any{
					"apiKey": "encrypted-api-key",
				},
			},
		},
	})

	resp, err := svc.GetClientRuntimeConfig(
		t.Context(),
		pagination.TenantInfo{},
		integration.TypeOANDAExchangeRates,
	)

	require.NoError(t, err)
	require.False(t, resp.Enabled)
	require.True(t, resp.Configured)
	require.False(t, resp.Ready)
	require.Empty(t, resp.MissingRequiredFields)
	require.Empty(t, resp.Config)
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
