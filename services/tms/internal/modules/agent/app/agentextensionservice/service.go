package agentextensionservice

import (
	"context"
	"errors"
	"math"
	"net/url"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/identity/app/planservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/secretconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/configspec"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
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
	_ agentextensionports.AgentExtensionGate = (*Service)(nil)
	_ agentextensionports.WebResearcher      = (*Service)(nil)
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         agentextensionports.AgentExtensionRepository
	Usage        agentextensionports.AgentExtensionUsageRepository
	Encryption   *encryptionservice.Service
	AuditService auditports.AuditService
	Connectors   []agentextensionports.ExtensionConnector `group:"agent_extension_connectors"`
	Plans        platformplanports.PlanService            `                                   optional:"true"`
	Sites        agentextensionports.PortalSiteRepository
	Runs         agentextensionports.PortalRunRepository
	Analyses     agentextensionports.AnalysisRunRepository
	Workflows    kernelports.WorkflowStarter `optional:"true"`
	Config       *config.Config              `optional:"true"`
}

type Service struct {
	l          *zap.Logger
	repo       agentextensionports.AgentExtensionRepository
	usage      agentextensionports.AgentExtensionUsageRepository
	secrets    secretconfig.Codec
	audit      auditports.AuditService
	connectors map[agentextension.Type]agentextensionports.ExtensionConnector
	webSearch  map[agentextension.Type]agentextensionports.WebSearchProvider
	browsers   map[agentextension.Type]agentextensionports.BrowserProvider
	sandboxes  map[agentextension.Type]agentextensionports.SandboxProvider
	plans      platformplanports.PlanService
	sites      agentextensionports.PortalSiteRepository
	runs       agentextensionports.PortalRunRepository
	analyses   agentextensionports.AnalysisRunRepository
	workflows  kernelports.WorkflowStarter
	appHost    string
	now        func() int64
}

func New(p Params) *Service {
	connectors := make(
		map[agentextension.Type]agentextensionports.ExtensionConnector,
		len(p.Connectors),
	)
	webSearch := make(
		map[agentextension.Type]agentextensionports.WebSearchProvider,
		len(p.Connectors),
	)
	browsers := make(
		map[agentextension.Type]agentextensionports.BrowserProvider,
		len(p.Connectors),
	)
	sandboxes := make(
		map[agentextension.Type]agentextensionports.SandboxProvider,
		len(p.Connectors),
	)
	for _, connector := range p.Connectors {
		if connector == nil {
			continue
		}
		typ := connector.Type()
		connectors[typ] = connector
		if provider, ok := connector.(agentextensionports.WebSearchProvider); ok {
			webSearch[typ] = provider
		}
		if provider, ok := connector.(agentextensionports.BrowserProvider); ok {
			browsers[typ] = provider
		}
		if provider, ok := connector.(agentextensionports.SandboxProvider); ok {
			sandboxes[typ] = provider
		}
	}

	return &Service{
		l:          p.Logger.Named("service.agentextension"),
		repo:       p.Repo,
		usage:      p.Usage,
		secrets:    secretconfig.NewCodec(p.Encryption, p.Logger),
		audit:      p.AuditService,
		connectors: connectors,
		webSearch:  webSearch,
		browsers:   browsers,
		sandboxes:  sandboxes,
		plans:      p.Plans,
		sites:      p.Sites,
		runs:       p.Runs,
		analyses:   p.Analyses,
		workflows:  p.Workflows,
		appHost:    appHostOf(p.Config),
		now:        timeutils.NowUnix,
	}
}

// appHostOf is the host Trenova's own web app is served from, which no portal
// site may cover.
func appHostOf(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	parsed, err := url.Parse(cfg.App.GetWebBaseURL())
	if err != nil {
		return ""
	}

	return strings.ToLower(parsed.Hostname())
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
) (*agentextensionports.AgentExtensionCatalogResponse, error) {
	installed, err := s.repo.ListByTenant(ctx, tenantInfo)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to list extensions").WithInternal(err)
	}

	byType := make(map[agentextension.Type]*agentextension.Extension, len(installed))
	for _, record := range installed {
		byType[record.Type] = record
	}

	now := s.now()
	usage, err := s.usage.Summarize(ctx, agentextensionports.SummarizeExtensionUsageParams{
		TenantInfo: tenantInfo,
		FromDay:    timeutils.DayKeyUTC(timeutils.MonthStartUTC(now)),
		Today:      timeutils.DayKeyUTC(now),
	})
	if err != nil {
		s.l.Warn("could not read extension usage; showing the catalog without it", zap.Error(err))
		usage = map[agentextension.Type]agentextension.UsageSummary{}
	}

	items := make([]agentextensionports.AgentExtensionCatalogItem, 0, len(definitions))
	for idx := range definitions {
		def := &definitions[idx]
		spec, ok := agentextension.SpecFor(def.Type)
		if !ok {
			continue
		}
		item := catalogItem(def, spec, byType[def.Type], usage[def.Type])
		if holder := enabledFor(installed, spec.Capability, def.Type); holder != nil {
			if held, found := definitionFor(holder.Type); found {
				item.CapabilityHeldBy = held.Vendor
			}
		}
		items = append(items, item)
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].SortOrder == items[j].SortOrder {
			return items[i].Name < items[j].Name
		}
		return items[i].SortOrder < items[j].SortOrder
	})

	return &agentextensionports.AgentExtensionCatalogResponse{
		Items:      items,
		Categories: categoryOptions(),
	}, nil
}

func catalogItem(
	def *definition,
	spec agentextension.Spec,
	record *agentextension.Extension,
	usage agentextension.UsageSummary,
) agentextensionports.AgentExtensionCatalogItem {
	capability := capabilityDefinitions[spec.Capability]
	item := agentextensionports.AgentExtensionCatalogItem{
		Type:                def.Type,
		Name:                def.Name,
		Vendor:              def.Vendor,
		Summary:             def.Summary,
		Description:         def.Description,
		Category:            def.Category,
		CategoryLabel:       categoryLabels[def.Category],
		Capability:          spec.Capability,
		CapabilityLabel:     capability.Label,
		BrandDomain:         def.BrandDomain,
		DocsURL:             def.DocsURL,
		WebsiteURL:          def.WebsiteURL,
		PricingURL:          def.PricingURL,
		Capabilities:        def.Capabilities,
		Tools:               capability.Tools,
		DataNotice:          def.DataNotice,
		Featured:            def.Featured,
		SortOrder:           def.SortOrder,
		ReleasedAt:          def.ReleasedAt,
		Availability:        agentextension.AvailabilitySelectedAgents,
		ConfigSpec:          spec.Fields,
		SupportsTestConnect: spec.SupportsTestConnect,
		DailyRequestLimit:   dailyLimitOf(nil, spec),
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
	item.DailyRequestLimit = dailyLimitOf(record.Configuration, spec)

	return item
}

func dailyLimitOf(configuration map[string]any, spec agentextension.Spec) int {
	return spec.DailyLimit(
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
) (*agentextensionports.AgentExtensionConfigResponse, error) {
	spec, _, err := lookup(typ)
	if err != nil {
		return nil, err
	}

	record, err := s.find(ctx, tenantInfo, typ)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return &agentextensionports.AgentExtensionConfigResponse{
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
) *agentextensionports.AgentExtensionConfigResponse {
	return &agentextensionports.AgentExtensionConfigResponse{
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
	req *agentextensionports.UpdateAgentExtensionRequest,
) (*agentextensionports.AgentExtensionConfigResponse, error) {
	spec, def, err := lookup(typ)
	if err != nil {
		return nil, err
	}

	tenantInfo := req.TenantInfo
	if req.Enabled {
		if err = s.requirePlan(ctx, tenantInfo, spec.Capability); err != nil {
			return nil, err
		}
	}
	existing, err := s.find(ctx, tenantInfo, typ)
	if err != nil {
		return nil, err
	}
	if err = checkVersion(existing, req.Version); err != nil {
		return nil, err
	}
	if req.Enabled {
		if err = s.checkCapabilityFree(ctx, tenantInfo, &def, spec); err != nil {
			return nil, err
		}
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
		name:     def.title(),
		critical: existing == nil || existing.Enabled != saved.Enabled,
	})

	return configResponse(saved, spec), nil
}

func (s *Service) checkCapabilityFree(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	def *definition,
	spec agentextension.Spec,
) error {
	installed, err := s.repo.ListByTenant(ctx, tenantInfo)
	if err != nil {
		return errortypes.NewBusinessError("failed to list extensions").WithInternal(err)
	}

	holder := enabledFor(installed, spec.Capability, def.Type)
	if holder == nil {
		return nil
	}
	held, ok := definitionFor(holder.Type)
	if !ok {
		return nil
	}

	multiErr := errortypes.NewMultiError()
	multiErr.Add(
		"enabled",
		errortypes.ErrInvalid,
		"{0} is already on and gives agents the same tools. Turn it off before turning on {1}.",
		held.Vendor,
		def.Vendor,
	)

	return multiErr
}

func enabledFor(
	installed []*agentextension.Extension,
	capability agentextension.Capability,
	except agentextension.Type,
) *agentextension.Extension {
	for _, typ := range agentextension.TypesFor(capability) {
		if typ == except {
			continue
		}
		for _, record := range installed {
			if record.Type == typ && record.Enabled {
				return record
			}
		}
	}

	return nil
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

	options := []auditports.LogOption{
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

	if err := s.audit.LogAction(&auditports.LogActionParams{
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

func (s *Service) ActiveCapabilities(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (agentextension.Active, error) {
	records, err := s.repo.ListByTenant(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	active := agentextension.ActiveFrom(records)
	for capability := range active {
		allowed, planErr := s.allows(ctx, tenantInfo, capability)
		if planErr != nil {
			return nil, planErr
		}
		if !allowed {
			delete(active, capability)
		}
	}

	return active, nil
}

type extensionSettings struct {
	typ       agentextension.Type
	def       definition
	values    map[string]string
	connector agentextensionports.ExtensionConnector
}

func (s *Service) serving(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	capability agentextension.Capability,
) (*extensionSettings, error) {
	label := capabilityDefinitions[capability].Label
	allowed, err := s.allows(ctx, tenantInfo, capability)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, &ToolError{Message: label + " is not included in this organization's plan."}
	}

	records, err := s.repo.ListByTenant(ctx, tenantInfo)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to list extensions").WithInternal(err)
	}

	served, ok := agentextension.ActiveFrom(records)[capability]
	if !ok {
		if enabledFor(records, capability, "") != nil {
			return nil, &ToolError{Message: label + " is not set up for this organization. " +
				"An administrator can finish setting it up under AI Control, Extensions."}
		}
		for _, typ := range agentextension.TypesFor(capability) {
			spec, _ := agentextension.SpecFor(typ)
			for _, record := range records {
				if record.Type == typ && spec.Configured(record.Configuration) {
					return nil, &ToolError{Message: label + " is turned off for this " +
						"organization. An administrator can turn it on under AI Control, Extensions."}
				}
			}
		}
		return nil, &ToolError{Message: label + " is not set up for this organization. An " +
			"administrator can set it up under AI Control, Extensions."}
	}

	for _, record := range records {
		if record.Type == served.Type {
			return s.settingsOf(tenantInfo, record)
		}
	}

	return nil, errortypes.NewBusinessError(
		"failed to read the {0} extension",
		served.Type.String(),
	)
}

func (s *Service) settingsFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ agentextension.Type,
) (*extensionSettings, error) {
	spec, def, err := lookup(typ)
	if err != nil {
		return nil, err
	}

	record, err := s.find(ctx, tenantInfo, typ)
	if err != nil {
		return nil, err
	}
	if record == nil || !spec.Configured(record.Configuration) {
		return nil, &ToolError{Message: def.title() + " is not set up for this organization. " +
			"An administrator can set it up under AI Control, Extensions."}
	}

	return s.settingsOf(tenantInfo, record)
}

func (s *Service) settingsOf(
	tenantInfo pagination.TenantInfo,
	record *agentextension.Extension,
) (*extensionSettings, error) {
	spec, def, err := lookup(record.Type)
	if err != nil {
		return nil, err
	}

	connector, ok := s.connectors[record.Type]
	if !ok {
		return nil, errortypes.NewBusinessError(
			"the {0} extension has no connector", record.Type.String(),
		)
	}

	values, err := s.secrets.ReadAll(
		record.Configuration,
		spec.Fields,
		secretScope(tenantInfo, record.Type),
	)
	if err != nil {
		return nil, errortypes.NewBusinessError(
			"failed to read the {0} extension settings", record.Type.String(),
		).WithInternal(err)
	}

	return &extensionSettings{
		typ:       record.Type,
		def:       def,
		values:    values,
		connector: connector,
	}, nil
}

func (s *Service) allows(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	capability agentextension.Capability,
) (bool, error) {
	return planservice.Allows(ctx, s.plans, tenantInfo, capabilityDefinitions[capability].Plan)
}

func (s *Service) requirePlan(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	capability agentextension.Capability,
) error {
	return planservice.RequireCapability(
		ctx,
		s.plans,
		tenantInfo,
		capabilityDefinitions[capability].Plan,
	)
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
	reserved, err := s.usage.Reserve(ctx, agentextensionports.ReserveExtensionRequestParams{
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
		spec, _ := agentextension.SpecFor(call.typ)
		capability, _ := agentextension.CapabilitySpecFor(spec.Capability)
		return 0, &ToolError{Message: dailyLimitMessage(capability, call.limit)}
	}

	return day, nil
}

func (s *Service) settle(ctx context.Context, call meteredCall, outcome settledCall) {
	if err := s.usage.RecordOutcome(ctx, agentextensionports.RecordExtensionOutcomeParams{
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
