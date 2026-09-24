package agentextensionservice

import (
	"context"
	"errors"
	"math"
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/agentextension"
	"github.com/emoss08/trenova/internal/core/domain/configspec"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/core/services/secretconfig"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	secretResourceKind = "agent_extension"
	unlimitedRequests  = math.MaxInt32
)

var (
	_ serviceports.AgentExtensionGate = (*Service)(nil)
	_ serviceports.WebResearcher      = (*Service)(nil)
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.AgentExtensionRepository
	Usage        repositories.AgentExtensionUsageRepository
	Encryption   *encryptionservice.Service
	AuditService serviceports.AuditService
	WebSearch    serviceports.WebSearchProvider
}

type Service struct {
	l         *zap.Logger
	repo      repositories.AgentExtensionRepository
	usage     repositories.AgentExtensionUsageRepository
	secrets   secretconfig.Codec
	audit     serviceports.AuditService
	webSearch serviceports.WebSearchProvider
	now       func() int64
}

func New(p Params) *Service {
	return &Service{
		l:         p.Logger.Named("service.agentextension"),
		repo:      p.Repo,
		usage:     p.Usage,
		secrets:   secretconfig.NewCodec(p.Encryption, p.Logger),
		audit:     p.AuditService,
		webSearch: p.WebSearch,
		now:       timeutils.NowUnix,
	}
}

func secretScope(tenantInfo pagination.TenantInfo, typ agentextension.Type) secretconfig.Scope {
	return secretconfig.Scope{
		Purpose:      encryptionservice.PurposeAgentExtensionSecret,
		Tenant:       tenantInfo,
		ResourceKind: secretResourceKind,
		Subject:      typ.String(),
		Bind:         true,
	}
}

func lookup(typ agentextension.Type) (agentextension.Spec, definition, error) {
	spec, hasSpec := agentextension.SpecFor(typ)
	def, hasDef := definitionFor(typ)
	if !hasSpec || !hasDef {
		return agentextension.Spec{}, definition{}, errortypes.NewNotFoundError(
			"There is no extension named {0}", typ.String(),
		)
	}

	return spec, def, nil
}

func (s *Service) ListCatalog(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*serviceports.AgentExtensionCatalogResponse, error) {
	installed, err := s.repo.ListByTenant(ctx, tenantInfo)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to list extensions").WithInternal(err)
	}

	byType := make(map[agentextension.Type]*agentextension.Extension, len(installed))
	for _, record := range installed {
		byType[record.Type] = record
	}

	now := s.now()
	usage, err := s.usage.Summarize(ctx, repositories.SummarizeExtensionUsageParams{
		TenantInfo: tenantInfo,
		FromDay:    timeutils.DayKeyUTC(timeutils.MonthStartUTC(now)),
		Today:      timeutils.DayKeyUTC(now),
	})
	if err != nil {
		s.l.Warn("could not read extension usage; showing the catalog without it", zap.Error(err))
		usage = map[agentextension.Type]agentextension.UsageSummary{}
	}

	items := make([]serviceports.AgentExtensionCatalogItem, 0, len(definitions))
	for idx := range definitions {
		def := &definitions[idx]
		spec, ok := agentextension.SpecFor(def.Type)
		if !ok {
			continue
		}
		items = append(items, catalogItem(def, spec, byType[def.Type], usage[def.Type]))
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].SortOrder == items[j].SortOrder {
			return items[i].Name < items[j].Name
		}
		return items[i].SortOrder < items[j].SortOrder
	})

	return &serviceports.AgentExtensionCatalogResponse{
		Items:      items,
		Categories: categoryOptions(),
	}, nil
}

func catalogItem(
	def *definition,
	spec agentextension.Spec,
	record *agentextension.Extension,
	usage agentextension.UsageSummary,
) serviceports.AgentExtensionCatalogItem {
	item := serviceports.AgentExtensionCatalogItem{
		Type:                def.Type,
		Name:                def.Name,
		Vendor:              def.Vendor,
		Summary:             def.Summary,
		Description:         def.Description,
		Category:            def.Category,
		CategoryLabel:       categoryLabels[def.Category],
		BrandDomain:         def.BrandDomain,
		DocsURL:             def.DocsURL,
		WebsiteURL:          def.WebsiteURL,
		PricingURL:          def.PricingURL,
		Capabilities:        def.Capabilities,
		Tools:               def.Tools,
		DataNotice:          def.DataNotice,
		Featured:            def.Featured,
		SortOrder:           def.SortOrder,
		ReleasedAt:          def.ReleasedAt,
		Availability:        agentextension.AvailabilitySelectedAgents,
		ConfigSpec:          spec.Fields,
		SupportsTestConnect: spec.SupportsTestConnect,
		DailyRequestLimit:   agentextension.DefaultDailyRequestLimit,
		Usage:               usage,
	}
	if record == nil {
		return item
	}

	item.Enabled = record.Enabled
	item.Configured = spec.Configured(record.Configuration)
	item.Availability = record.Availability
	item.EnabledAt = record.EnabledAt
	item.UpdatedAt = record.UpdatedAt
	item.Version = record.Version
	item.DailyRequestLimit = dailyLimitOf(record.Configuration)

	return item
}

func dailyLimitOf(configuration map[string]any) int {
	return agentextension.DailyRequestLimit(
		map[string]string{
			agentextension.ConfigKeyDailyRequestLimit: configspec.ReadString(
				configuration,
				agentextension.ConfigKeyDailyRequestLimit,
			),
		},
		errortypes.NewMultiError(),
	)
}

func (s *Service) GetConfig(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ agentextension.Type,
) (*serviceports.AgentExtensionConfigResponse, error) {
	spec, _, err := lookup(typ)
	if err != nil {
		return nil, err
	}

	record, err := s.find(ctx, tenantInfo, typ)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return &serviceports.AgentExtensionConfigResponse{
			Type:         typ,
			Availability: agentextension.AvailabilitySelectedAgents,
			Fields:       configspec.Values(nil, spec.Fields),
			Spec:         spec.Fields,
		}, nil
	}

	return configResponse(record, spec), nil
}

func configResponse(
	record *agentextension.Extension,
	spec agentextension.Spec,
) *serviceports.AgentExtensionConfigResponse {
	return &serviceports.AgentExtensionConfigResponse{
		Type:         record.Type,
		Enabled:      record.Enabled,
		Availability: record.Availability,
		Fields:       configspec.Values(record.Configuration, spec.Fields),
		Spec:         spec.Fields,
		Version:      record.Version,
		UpdatedAt:    record.UpdatedAt,
	}
}

func (s *Service) find(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ agentextension.Type,
) (*agentextension.Extension, error) {
	record, err := s.repo.GetByType(ctx, tenantInfo, typ)
	if err == nil {
		return record, nil
	}
	if errortypes.IsNotFoundError(err) {
		return nil, nil
	}

	return nil, errortypes.NewBusinessError(
		"failed to read the {0} extension", typ.String(),
	).WithInternal(err)
}

func (s *Service) UpdateConfig(
	ctx context.Context,
	typ agentextension.Type,
	req *serviceports.UpdateAgentExtensionRequest,
) (*serviceports.AgentExtensionConfigResponse, error) {
	spec, def, err := lookup(typ)
	if err != nil {
		return nil, err
	}

	tenantInfo := req.TenantInfo
	existing, err := s.find(ctx, tenantInfo, typ)
	if err != nil {
		return nil, err
	}
	if err = checkVersion(existing, req.Version); err != nil {
		return nil, err
	}

	var existingConfig map[string]any
	if existing != nil {
		existingConfig = existing.Configuration
	}
	scope := secretScope(tenantInfo, typ)
	merged, err := s.secrets.Merge(spec.Fields, req.Configuration, existingConfig, scope)
	if err != nil {
		return nil, err
	}

	availability := req.Availability
	if availability == "" {
		availability = agentextension.AvailabilitySelectedAgents
	}

	entity := &agentextension.Extension{
		BusinessUnitID: tenantInfo.BuID,
		OrganizationID: tenantInfo.OrgID,
		Type:           typ,
		Enabled:        req.Enabled,
		Availability:   availability,
		Configuration:  merged,
	}
	if existing == nil {
		entity.Version = 1
	} else {
		entity.ID = existing.ID
		entity.Version = existing.Version
		entity.CreatedAt = existing.CreatedAt
		entity.EnabledByID = existing.EnabledByID
		entity.EnabledAt = existing.EnabledAt
	}
	s.stampEnablement(entity, existing, tenantInfo.UserID)

	if err = validate(entity, spec, merged); err != nil {
		return nil, err
	}

	var saved *agentextension.Extension
	if existing == nil {
		saved, err = s.repo.Create(ctx, entity)
	} else {
		saved, err = s.repo.Update(ctx, entity)
	}
	if err != nil {
		return nil, err
	}

	s.logChange(auditChange{
		tenant:   tenantInfo,
		before:   existing,
		after:    saved,
		spec:     spec,
		name:     def.Vendor + " " + def.Name,
		critical: existing == nil || existing.Enabled != saved.Enabled,
	})

	return configResponse(saved, spec), nil
}

func checkVersion(existing *agentextension.Extension, version int64) error {
	current := int64(0)
	if existing != nil {
		current = existing.Version
	}
	if version == current {
		return nil
	}

	return errortypes.NewConflictError(
		"These settings were changed by someone else while you were editing. Reload and try again.",
	)
}

func (s *Service) stampEnablement(
	entity *agentextension.Extension,
	existing *agentextension.Extension,
	userID pulid.ID,
) {
	if !entity.Enabled {
		return
	}
	if existing != nil && existing.Enabled {
		return
	}

	now := s.now()
	entity.EnabledByID = userID
	entity.EnabledAt = &now
}

func validate(
	entity *agentextension.Extension,
	spec agentextension.Spec,
	merged map[string]any,
) error {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	plain := make(map[string]string, len(spec.Fields))
	for idx := range spec.Fields {
		field := &spec.Fields[idx]
		value := configspec.ReadString(merged, field.Key)
		if field.Sensitive {
			if entity.Enabled && field.Required && value == "" {
				multiErr.Add(
					"configuration."+field.Key,
					errortypes.ErrRequired,
					field.Label+" is required to turn this extension on",
				)
			}
			continue
		}
		plain[field.Key] = value
		if entity.Enabled && field.Required && value == "" {
			multiErr.Add(
				"configuration."+field.Key,
				errortypes.ErrRequired,
				field.Label+" is required to turn this extension on",
			)
		}
	}
	agentextension.ValidateSettings(entity.Type, plain, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

type auditChange struct {
	tenant   pagination.TenantInfo
	before   *agentextension.Extension
	after    *agentextension.Extension
	spec     agentextension.Spec
	name     string
	critical bool
}

func (s *Service) logChange(change auditChange) {
	after := auditSnapshot(change.after, change.spec)
	before := auditSnapshot(change.before, change.spec)

	options := []serviceports.LogOption{
		auditservice.WithComment(change.name + " extension settings updated"),
		auditservice.WithDiff(before, after),
	}
	if change.critical {
		options = append(options, auditservice.WithCritical())
	}

	operation := permission.OpUpdate
	if change.before == nil {
		operation = permission.OpCreate
	}

	if err := s.audit.LogAction(&serviceports.LogActionParams{
		Resource:       permission.ResourceAgentExtension,
		ResourceID:     change.after.ID.String(),
		Operation:      operation,
		UserID:         change.tenant.UserID,
		CurrentState:   after,
		PreviousState:  before,
		BusinessUnitID: change.after.BusinessUnitID,
		OrganizationID: change.after.OrganizationID,
	}, options...); err != nil {
		s.l.Error("failed to log agent extension change", zap.Error(err))
	}
}

func auditSnapshot(record *agentextension.Extension, spec agentextension.Spec) map[string]any {
	if record == nil {
		return nil
	}

	configuration := make(map[string]any, len(spec.Fields))
	for idx := range spec.Fields {
		field := &spec.Fields[idx]
		value := configspec.ReadString(record.Configuration, field.Key)
		if field.Sensitive {
			configuration[field.Key] = value != ""
			continue
		}
		configuration[field.Key] = value
	}

	return map[string]any{
		"id":            record.ID.String(),
		"type":          record.Type.String(),
		"enabled":       record.Enabled,
		"availability":  record.Availability.String(),
		"configuration": configuration,
		"enabledById":   record.EnabledByID.String(),
		"version":       record.Version,
	}
}

func (s *Service) ActiveExtensions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (map[agentextension.Type]agentextension.Availability, error) {
	records, err := s.repo.ListByTenant(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	active := make(map[agentextension.Type]agentextension.Availability, len(records))
	for _, record := range records {
		spec, ok := agentextension.SpecFor(record.Type)
		if !ok || !record.Ready(spec) {
			continue
		}
		active[record.Type] = record.Availability
	}

	return active, nil
}

type runtimeSettings struct {
	values map[string]string
	record *agentextension.Extension
}

func (s *Service) runtime(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ agentextension.Type,
	requireEnabled bool,
) (*runtimeSettings, error) {
	spec, def, err := lookup(typ)
	if err != nil {
		return nil, err
	}

	record, err := s.find(ctx, tenantInfo, typ)
	if err != nil {
		return nil, err
	}
	if record == nil || !spec.Configured(record.Configuration) {
		return nil, &ToolError{Message: def.Vendor + " " + def.Name + " is not set up for this " +
			"organization. An administrator can set it up under AI Control, Extensions."}
	}
	if requireEnabled && !record.Enabled {
		return nil, &ToolError{Message: def.Vendor + " " + def.Name + " is turned off for this " +
			"organization. An administrator can turn it on under AI Control, Extensions."}
	}

	values, err := s.secrets.ReadAll(record.Configuration, spec.Fields, secretScope(tenantInfo, typ))
	if err != nil {
		return nil, errortypes.NewBusinessError(
			"failed to read the {0} extension settings", typ.String(),
		).WithInternal(err)
	}

	return &runtimeSettings{values: values, record: record}, nil
}

type ToolError struct {
	Message string
}

func (e *ToolError) Error() string { return e.Message }

func IsToolError(err error) bool {
	var toolErr *ToolError
	return errors.As(err, &toolErr)
}

type meteredCall struct {
	tenant pagination.TenantInfo
	typ    agentextension.Type
	limit  int
}

func (s *Service) reserve(ctx context.Context, call meteredCall) (int, error) {
	now := s.now()
	day := timeutils.DayKeyUTC(now)
	reserved, err := s.usage.Reserve(ctx, repositories.ReserveExtensionRequestParams{
		TenantInfo: call.tenant,
		Type:       call.typ,
		Day:        day,
		Limit:      call.limit,
		Now:        now,
	})
	if err != nil {
		return 0, errortypes.NewBusinessError("could not check the extension's daily limit").
			WithInternal(err)
	}
	if !reserved {
		return 0, &ToolError{Message: dailyLimitMessage(call.limit)}
	}

	return day, nil
}

func (s *Service) settle(ctx context.Context, call meteredCall, outcome settledCall) {
	if err := s.usage.RecordOutcome(ctx, repositories.RecordExtensionOutcomeParams{
		TenantInfo: call.tenant,
		Type:       call.typ,
		Day:        outcome.day,
		Failed:     outcome.failed,
		CostUSD:    outcome.cost,
		Now:        s.now(),
	}); err != nil {
		s.l.Warn("could not record extension usage", zap.Error(err),
			zap.String("type", call.typ.String()))
	}
}
