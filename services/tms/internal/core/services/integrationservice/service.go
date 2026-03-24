package integrationservice

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
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
}

type Service struct {
	l            *zap.Logger
	repo         repositories.IntegrationRepository
	encryption   *encryptionservice.Service
	auditService services.AuditService
	registry     *permission.Registry
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.integration"),
		repo:         p.Repo,
		encryption:   p.Encryption,
		auditService: p.AuditService,
		registry:     p.Registry,
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
			item.Configured = isConfigured(installedRecord.Configuration, spec)
		}

		item.Status = buildCatalogStatus(item.Enabled, item.Configured)
		items = append(items, item)
	}

	sortCatalogItems(items)
	return &services.CatalogResponse{Items: items}, nil
}

func isConfigured(configuration map[string]any, spec integration.IntegrationSpec) bool {
	for _, field := range spec.Fields {
		if !field.Required {
			continue
		}
		val := readConfigString(configuration, field.Key)
		if val == "" {
			return false
		}
	}
	return true
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
				Fields:  buildFieldValues(nil, spec),
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
		Fields:    buildFieldValues(record.Configuration, spec),
		Spec:      spec.Fields,
		UpdatedAt: record.UpdatedAt,
	}, nil
}

func buildFieldValues(
	configuration map[string]any,
	spec integration.IntegrationSpec,
) []services.ConfigFieldValue {
	fields := make([]services.ConfigFieldValue, 0, len(spec.Fields))
	for _, f := range spec.Fields {
		val := readConfigString(configuration, f.Key)
		fv := services.ConfigFieldValue{
			Key:      f.Key,
			HasValue: val != "",
		}
		if !f.Sensitive {
			fv.Value = val
		}
		fields = append(fields, fv)
	}
	return fields
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

	finalConfig, err := s.buildFinalConfig(spec, req.Configuration, existing)
	if err != nil {
		return nil, err
	}

	if err = validateRequiredFields(spec, finalConfig, req.Enabled); err != nil {
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
		Fields:    buildFieldValues(updated.Configuration, spec),
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

	tester, ok := connectionTesters[typ]
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
		return nil, errortypes.NewBusinessError(
			"failed to connect to " + string(typ),
		).WithInternal(err)
	}

	if !cfg.Enabled {
		if _, err = s.UpdateConfig(
			ctx,
			tenantInfo,
			typ,
			&services.UpdateConfigRequest{
				TenantInfo:    tenantInfo,
				Enabled:       true,
				Configuration: map[string]string{},
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
		CheckedAt: time.Now().Unix(),
	}, nil
}

type RuntimeConfig struct {
	Enabled bool
	Config  map[string]string
}

func (s *Service) GetRuntimeConfig(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
) (*RuntimeConfig, error) {
	return s.getRuntimeConfig(ctx, tenantInfo, typ, true)
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
			return nil, errortypes.NewBusinessError(string(typ) + " integration is not configured")
		}

		return nil, errortypes.NewBusinessError(
			"failed to retrieve " + string(typ) + " configuration",
		).WithInternal(err)
	}

	if requireEnabled && !record.Enabled {
		return nil, errortypes.NewBusinessError(string(typ) + " integration is disabled")
	}

	cfg := make(map[string]string, len(spec.Fields))
	for _, field := range spec.Fields {
		val := strings.TrimSpace(readConfigString(record.Configuration, field.Key))
		if field.Sensitive && val != "" {
			decrypted, decErr := s.encryption.DecryptString(val)
			if decErr != nil {
				return nil, errortypes.NewBusinessError(
					"failed to decrypt " + string(typ) + " configuration",
				).WithInternal(decErr)
			}
			cfg[field.Key] = decrypted
		} else {
			cfg[field.Key] = val
		}
	}

	for _, field := range spec.Fields {
		if field.Required && cfg[field.Key] == "" {
			return nil, errortypes.NewBusinessError(
				string(typ) + " integration " + field.Label + " is missing",
			)
		}
	}

	return &RuntimeConfig{
		Enabled: record.Enabled,
		Config:  cfg,
	}, nil
}

func (s *Service) buildFinalConfig(
	spec integration.IntegrationSpec,
	incoming map[string]string,
	existing *integration.Integration,
) (map[string]any, error) {
	finalConfig := make(map[string]any, len(spec.Fields))

	for idx := range spec.Fields {
		field := &spec.Fields[idx]
		val := strings.TrimSpace(incoming[field.Key])

		if field.Sensitive {
			stored, storeErr := s.resolveSensitiveField(field, val, existing)
			if storeErr != nil {
				return nil, storeErr
			}
			finalConfig[field.Key] = stored
		} else {
			finalConfig[field.Key] = resolveNonSensitiveField(field, val)
		}
	}

	return finalConfig, nil
}

func (s *Service) resolveSensitiveField(
	field *integration.ConfigFieldSpec,
	incoming string,
	existing *integration.Integration,
) (string, error) {
	if incoming == "" {
		if existing != nil {
			return readConfigString(existing.Configuration, field.Key), nil
		}
		return "", nil
	}

	encrypted, err := s.encryption.EncryptString(incoming)
	if err != nil {
		return "", errortypes.NewBusinessError(
			"failed to encrypt configuration value",
		).WithInternal(err)
	}
	return encrypted, nil
}

func resolveNonSensitiveField(
	field *integration.ConfigFieldSpec,
	incoming string,
) string {
	if incoming == "" && field.Default != "" {
		return field.Default
	}
	return incoming
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
		val := readConfigString(config, field.Key)
		if val == "" {
			return errortypes.NewBusinessError(
				field.Label + " is required when integration is enabled",
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

func readConfigString(configuration map[string]any, key string) string {
	if len(configuration) == 0 {
		return ""
	}

	value, ok := configuration[key]
	if !ok || value == nil {
		return ""
	}

	stringValue, ok := value.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(stringValue)
}
