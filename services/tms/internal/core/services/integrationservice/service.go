package integrationservice

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/configspec"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/core/services/secretconfig"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.IntegrationRepository
	Encryption   *encryptionservice.Service
	AuditService services.AuditService
	Registry     *permission.Registry

	FuelCardConnectors     []services.FuelCardProvider                `group:"fuelCardConnectors"`
	CarrierIntelConnectors []services.CarrierIntelConnector           `group:"carrierIntelConnectors"`
	CarrierIntelControls   repositories.CarrierIntelControlRepository `                               optional:"true"`
}

type Service struct {
	l                      *zap.Logger
	repo                   repositories.IntegrationRepository
	encryption             *encryptionservice.Service
	secrets                secretconfig.Codec
	auditService           services.AuditService
	registry               *permission.Registry
	fuelCardConnectors     map[integration.Type]services.FuelCardProvider
	carrierIntelConnectors map[integration.Type]services.CarrierIntelConnector
	carrierIntelControls   repositories.CarrierIntelControlRepository
}

func New(p Params) *Service {
	connectors := make(map[integration.Type]services.FuelCardProvider, len(p.FuelCardConnectors))
	for _, connector := range p.FuelCardConnectors {
		if connector != nil {
			connectors[connector.IntegrationType()] = connector
		}
	}

	intelConnectors := make(
		map[integration.Type]services.CarrierIntelConnector,
		len(p.CarrierIntelConnectors),
	)
	for _, connector := range p.CarrierIntelConnectors {
		if connector != nil {
			intelConnectors[connector.IntegrationType()] = connector
		}
	}

	return &Service{
		l:                      p.Logger.Named("service.integration"),
		repo:                   p.Repo,
		encryption:             p.Encryption,
		secrets:                secretconfig.NewCodec(p.Encryption, p.Logger),
		auditService:           p.AuditService,
		registry:               p.Registry,
		fuelCardConnectors:     connectors,
		carrierIntelConnectors: intelConnectors,
		carrierIntelControls:   p.CarrierIntelControls,
	}
}

func (s *Service) ListCatalog(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*services.CatalogResponse, error) {
	installed, err := s.repo.ListByTenant(ctx, tenantInfo)
	if err != nil {
		return nil, errortypes.NewBusinessError(
			"failed to list integrations",
		).WithInternal(err)
	}

	installedByType := make(map[integration.Type]*integration.Integration, len(installed))
	for idx := range installed {
		record := installed[idx]
		installedByType[record.Type] = record
	}

	items := make([]services.CatalogItem, 0, len(services.CatalogDefinitions))
	for idx := range services.CatalogDefinitions {
		def := services.CatalogDefinitions[idx]
		installedRecord := installedByType[def.Type]

		item := def
		item.Enabled = false
		item.Configured = false
		item.Status = services.CatalogStatus{}

		spec, hasSpec := integration.ConfigSpecs[def.Type]
		if hasSpec {
			item.ConfigSpec = spec.Fields
			item.SupportsTestConnect = spec.SupportsTestConnect
		}

		if installedRecord != nil {
			item.Enabled = installedRecord.Enabled
			item.Configured = integration.HasRequiredConfiguration(
				installedRecord.Configuration,
				spec,
			)
		}

		item.Status = buildCatalogStatus(item.Enabled, item.Configured)
		items = append(items, item)
	}

	sortCatalogItems(items)
	return &services.CatalogResponse{Items: items}, nil
}

func buildCatalogStatus(enabled, configured bool) services.CatalogStatus {
	connection := services.CatalogConnectionStatusDisconnected
	connectionLabel := "Disconnected"
	if enabled {
		connection = services.CatalogConnectionStatusConnected
		connectionLabel = "Connected"
	}

	configuration := services.CatalogConfigurationStatusNeedsSetup
	configurationLabel := "Needs Setup"
	if configured {
		configuration = services.CatalogConfigurationStatusConfigured
		configurationLabel = "Configured"
	}

	return services.CatalogStatus{
		Connection:         connection,
		ConnectionLabel:    connectionLabel,
		Configuration:      configuration,
		ConfigurationLabel: configurationLabel,
	}
}

func sortCatalogItems(items []services.CatalogItem) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].SortOrder == items[j].SortOrder {
			return items[i].Name < items[j].Name
		}
		return items[i].SortOrder < items[j].SortOrder
	})
}

func webhookTokenPrefix(typ integration.Type) string {
	switch typ {
	case integration.TypeResend:
		return "resend_"
	case integration.TypePostmark:
		return "postmark_"
	case integration.TypeSamsara:
		return "samsara_"
	default:
		return ""
	}
}

func newWebhookToken(prefix string) (string, error) {
	var tokenBytes [24]byte
	if _, err := rand.Read(tokenBytes[:]); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(tokenBytes[:]), nil
}

func (s *Service) GetConfig(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
) (*services.ConfigResponse, error) {
	spec, ok := integration.ConfigSpecs[typ]
	if !ok {
		return nil, errortypes.NewBusinessError("unsupported integration type")
	}

	record, err := s.repo.GetByType(ctx, tenantInfo, typ)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return &services.ConfigResponse{
				Type:    typ,
				Enabled: false,
				Fields:  configspec.Values(nil, spec.Fields),
				Spec:    spec.Fields,
			}, nil
		}

		return nil, errortypes.NewBusinessError(
			"failed to retrieve integration configuration",
		).WithInternal(err)
	}

	return &services.ConfigResponse{
		Type:      typ,
		Enabled:   record.Enabled,
		Fields:    configspec.Values(record.Configuration, spec.Fields),
		Spec:      spec.Fields,
		UpdatedAt: record.UpdatedAt,
	}, nil
}

func (s *Service) UpdateConfig(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
	req *services.UpdateConfigRequest,
	userID pulid.ID,
) (*services.ConfigResponse, error) {
	log := s.l.With(zap.String("Operation", "UpdateConfig"), zap.String("Type", string(typ)))

	spec, ok := integration.ConfigSpecs[typ]
	if !ok {
		return nil, errortypes.NewBusinessError("unsupported integration type")
	}

	existing, err := s.repo.GetByType(ctx, tenantInfo, typ)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return nil, err
	}

	scope := newSecretScope(tenantInfo, typ, spec)
	var existingConfig map[string]any
	if existing != nil {
		existingConfig = existing.Configuration
	}
	finalConfig, err := s.secrets.Merge(spec.Fields, req.Configuration, existingConfig, scope)
	if err != nil {
		return nil, err
	}
	if prefix := webhookTokenPrefix(typ); prefix != "" &&
		integration.ReadConfigString(finalConfig, "webhookToken") == "" {
		finalConfig["webhookToken"], err = newWebhookToken(prefix)
		if err != nil {
			return nil, errortypes.NewBusinessError("failed to generate webhook token").
				WithInternal(err)
		}
	}

	if err = validateRequiredFields(spec, finalConfig, req.Enabled); err != nil {
		return nil, err
	}

	roleChange := &carrierIntelRoleChange{
		tenant:  tenantInfo,
		typ:     typ,
		enabled: req.Enabled,
		role:    integration.ReadConfigString(finalConfig, integration.ConfigKeyCarrierIntelRole),
	}
	if err = s.checkCarrierIntelRole(ctx, roleChange); err != nil {
		return nil, err
	}

	catalogDef := findCatalogDefinition(typ)
	entity := &integration.Integration{
		BusinessUnitID: tenantInfo.BuID,
		OrganizationID: tenantInfo.OrgID,
		Type:           typ,
		Name:           catalogDef.Name,
		Description:    catalogDef.Description,
		Enabled:        req.Enabled,
		BuiltBy:        "Trenova",
		Category:       catalogDef.Category,
		DocsURL:        catalogDef.DocsURL,
		WebsiteURL:     catalogDef.WebsiteURL,
		EnabledByID:    userID,
		Configuration:  finalConfig,
	}

	updated, err := s.repo.Upsert(ctx, entity)
	if err != nil {
		return nil, errortypes.NewBusinessError(
			"failed to save integration configuration",
		).WithInternal(err)
	}

	if err = s.syncCarrierIntelRole(ctx, roleChange); err != nil {
		log.Error("failed to sync carrier intelligence provider role", zap.Error(err))
		return nil, errortypes.NewBusinessError(
			"The integration was saved but carrier intelligence settings could not be updated",
		).WithInternal(err)
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceIntegration,
		ResourceID:     updated.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(updated),
		PreviousState:  jsonutils.MustToJSON(entity),
		BusinessUnitID: updated.BusinessUnitID,
		OrganizationID: updated.OrganizationID,
	},
		auditservice.WithComment(catalogDef.Name+" Config Updated"),
		auditservice.WithDiff(entity, updated)); err != nil {
		log.Error("failed to log audit actions", zap.Error(err))
	}

	return &services.ConfigResponse{
		Type:      typ,
		Enabled:   updated.Enabled,
		Fields:    configspec.Values(updated.Configuration, spec.Fields),
		Spec:      spec.Fields,
		UpdatedAt: updated.UpdatedAt,
	}, nil
}

func (s *Service) TestConnection(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
	userID pulid.ID,
) (*services.TestConnectionResponse, error) {
	log := s.l.With(zap.String("Operation", "TestConnection"), zap.String("Type", string(typ)))

	spec, ok := integration.ConfigSpecs[typ]
	if !ok {
		return nil, errortypes.NewBusinessError("unsupported integration type")
	}

	if !spec.SupportsTestConnect {
		return nil, errortypes.NewBusinessError(
			"this integration does not support connection testing",
		)
	}

	tester, ok := s.testerFor(typ)
	if !ok {
		return nil, errortypes.NewBusinessError(
			"no connection tester registered for this integration type",
		)
	}

	cfg, err := s.getRuntimeConfig(ctx, tenantInfo, typ, false)
	if err != nil {
		return nil, err
	}

	if err = tester.Test(ctx, cfg.Config); err != nil {
		log.Error("connection test failed", zap.Error(err))
		if typ == integration.TypePCMiler || typ.SupportsCarrierIntelligence() {
			return nil, errortypes.NewBusinessError(err.Error()).WithInternal(err)
		}
		return nil, errortypes.NewBusinessError(
			"failed to connect to {0}", string(typ),
		).WithInternal(err)
	}

	if !cfg.Enabled {
		preserved := make(map[string]string, len(spec.Fields))
		for _, field := range spec.Fields {
			if !field.Sensitive {
				preserved[field.Key] = cfg.Config[field.Key]
			}
		}
		if _, err = s.UpdateConfig(
			ctx,
			tenantInfo,
			typ,
			&services.UpdateConfigRequest{
				TenantInfo:    tenantInfo,
				Enabled:       true,
				Configuration: preserved,
			},
			userID,
		); err != nil {
			log.Error(
				"Failed to enable integration after successful connection test",
				zap.Error(err),
			)
			return nil, err
		}
	}

	return &services.TestConnectionResponse{
		Provider:  typ,
		Success:   true,
		CheckedAt: timeutils.NowUnix(),
	}, nil
}

type RuntimeConfig struct {
	Enabled               bool              `json:"enabled"`
	Configured            bool              `json:"configured"`
	Ready                 bool              `json:"ready"`
	MissingRequiredFields []string          `json:"missingRequiredFields"`
	Config                map[string]string `json:"config"`
}

var clientRuntimeConfigFields = map[integration.Type]map[string]struct{}{
	integration.TypeGoogleMaps: {
		"apiKey": {},
	},
	integration.TypeOpenWeatherMap: {
		"apiKey": {},
	},
	integration.TypeOANDAExchangeRates: {},
}

func (s *Service) GetRuntimeConfig(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
) (*RuntimeConfig, error) {
	return s.getRuntimeConfig(ctx, tenantInfo, typ, true)
}

func (s *Service) GetClientRuntimeConfig(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
) (*RuntimeConfig, error) {
	allowedFields, ok := clientRuntimeConfigFields[typ]
	if !ok {
		return nil, errortypes.NewBusinessError(
			"{0} runtime configuration is not available to clients", string(typ),
		)
	}

	spec, ok := integration.ConfigSpecs[typ]
	if !ok {
		return nil, errortypes.NewBusinessError("unsupported integration type")
	}

	record, err := s.repo.GetByType(ctx, tenantInfo, typ)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return &RuntimeConfig{
				Enabled:               false,
				Configured:            false,
				Ready:                 false,
				MissingRequiredFields: configspec.MissingRequired(nil, spec.Fields),
				Config:                map[string]string{},
			}, nil
		}

		return nil, errortypes.NewBusinessError(
			"failed to retrieve {0} configuration", string(typ),
		).WithInternal(err)
	}

	missingRequiredFields := configspec.MissingRequired(record.Configuration, spec.Fields)
	configured := len(missingRequiredFields) == 0
	ready := record.Enabled && configured
	clientCfg := make(map[string]string, len(allowedFields))
	if ready {
		fieldsByKey := configspec.ByKey(spec.Fields)
		for key := range allowedFields {
			value, err := s.secrets.ReadField(
				record.Configuration,
				fieldsByKey[key],
				newSecretScope(tenantInfo, typ, spec),
			)
			if err != nil {
				return nil, errortypes.NewBusinessError(
					"failed to decrypt {0} configuration", string(typ),
				).WithInternal(err)
			}
			if value != "" {
				clientCfg[key] = value
			}
		}
	}

	return &RuntimeConfig{
		Enabled:               record.Enabled,
		Configured:            configured,
		Ready:                 ready,
		MissingRequiredFields: missingRequiredFields,
		Config:                clientCfg,
	}, nil
}

func (s *Service) getRuntimeConfig(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
	requireEnabled bool,
) (*RuntimeConfig, error) {
	spec, ok := integration.ConfigSpecs[typ]
	if !ok {
		return nil, errortypes.NewBusinessError("unsupported integration type")
	}

	record, err := s.repo.GetByType(ctx, tenantInfo, typ)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errortypes.NewBusinessError(
				"{0} integration is not configured",
				string(typ),
			)
		}

		return nil, errortypes.NewBusinessError(
			"failed to retrieve {0} configuration", string(typ),
		).WithInternal(err)
	}

	if requireEnabled && !record.Enabled {
		return nil, errortypes.NewBusinessError("{0} integration is disabled", string(typ))
	}

	cfg := make(map[string]string, len(spec.Fields))
	scope := newSecretScope(tenantInfo, typ, spec)
	for _, field := range spec.Fields {
		val, readErr := s.secrets.ReadField(record.Configuration, &field, scope)
		if readErr != nil {
			return nil, errortypes.NewBusinessError(
				"failed to decrypt {0} configuration", string(typ),
			).WithInternal(readErr)
		}
		cfg[field.Key] = val
	}

	missingRequiredFields := configspec.MissingRequiredValues(cfg, spec.Fields)
	for _, field := range spec.Fields {
		if field.Required && cfg[field.Key] == "" {
			return nil, errortypes.NewBusinessError(
				"{0} integration {1} is missing", string(typ), field.Label,
			)
		}
	}

	return &RuntimeConfig{
		Enabled:               record.Enabled,
		Configured:            len(missingRequiredFields) == 0,
		Ready:                 record.Enabled && len(missingRequiredFields) == 0,
		MissingRequiredFields: missingRequiredFields,
		Config:                cfg,
	}, nil
}

func validateRequiredFields(
	spec integration.IntegrationSpec,
	config map[string]any,
	enabled bool,
) error {
	if !enabled {
		return nil
	}

	for _, field := range spec.Fields {
		if !field.Required {
			continue
		}
		val := integration.ReadConfigString(config, field.Key)
		if val == "" {
			return errortypes.NewBusinessError(
				"{0} is required when integration is enabled", field.Label,
			)
		}
	}
	return nil
}

func findCatalogDefinition(typ integration.Type) services.CatalogItem {
	for idx := range services.CatalogDefinitions {
		if services.CatalogDefinitions[idx].Type == typ {
			return services.CatalogDefinitions[idx]
		}
	}
	return services.CatalogItem{Name: string(typ)}
}
