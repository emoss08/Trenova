package homelayoutservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homelayout"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/homelayoutports"
	"go.uber.org/zap"
)

// resolve walks the four tiers a home screen can come from and narrows the
// winner to what the viewer may actually see.
//
//  1. the viewer's own layout, unless the preset they diverged from is locked
//  2. the highest-priority preset assigned to one of their roles
//  3. the organization's default preset
//  4. the layout shipped for their role's core responsibility
//
// Every tier passes through Normalize (which drops widgets retired from the
// catalog) and then permission narrowing (which drops widgets whose data the
// viewer cannot read). Both are silent by design: a home screen degrades to
// fewer cards, never to an error. A widget the plan leaves out is the one
// exception: it stays in its slot, locked with the add-on that carries it, and
// nothing about it is read.
func (s *Service) resolve(
	ctx context.Context,
	req *Request,
	document *homelayout.Document,
) (*EffectiveLayout, error) {
	roles, presets, err := s.loadPresetContext(ctx, req)
	if err != nil {
		return nil, err
	}

	return s.resolveWith(ctx, req, document, roles, presets)
}

// resolveWith is resolve with the roles and presets already in hand, so a
// write path that had to load them for its own checks does not fetch the same
// rows twice in one request.
func (s *Service) resolveWith(
	ctx context.Context,
	req *Request,
	document *homelayout.Document,
	roles []*rbac.Role,
	presets []*homelayout.Preset,
) (*EffectiveLayout, error) {
	assigned, source := pickPreset(presets, roles)
	responsibility := primaryResponsibility(roles)

	effective := &EffectiveLayout{
		Density:      document.Density,
		CanCustomize: true,
	}

	unlockedByPreset := assigned == nil || !assigned.Locked

	switch {
	case document.Mode == homelayout.ModeCustom && unlockedByPreset:
		effective.Layout = document.Layout
		effective.Source = SourceUser
		if assigned != nil {
			effective.PresetID = assigned.ID
			effective.PresetName = assigned.Name
		} else {
			effective.PresetName = homelayout.BuiltInLayoutName(responsibility)
		}
	case assigned != nil:
		effective.Layout = assigned.Layout
		effective.Source = source
		effective.PresetID = assigned.ID
		effective.PresetName = assigned.Name
		effective.Locked = assigned.Locked
		effective.CanCustomize = !assigned.Locked
	default:
		effective.Layout = homelayout.BuiltInLayout(responsibility)
		effective.Source = SourceBuiltIn
		effective.PresetName = homelayout.BuiltInLayoutName(responsibility)
	}

	narrowed, err := s.narrow(ctx, req, effective.Layout.Normalize())
	if err != nil {
		return nil, err
	}
	effective.Layout = narrowed.Layout
	effective.Locks = narrowed.Locks

	return effective, nil
}

// loadPresetContext fetches the two inputs preset resolution needs: the
// viewer's roles and the tenant's presets.
func (s *Service) loadPresetContext(
	ctx context.Context,
	req *Request,
) ([]*rbac.Role, []*homelayout.Preset, error) {
	roles, err := s.rbac.GetAuthorizedRoles(ctx, req.TenantInfo.UserID, req.TenantInfo.OrgID)
	if err != nil {
		s.l.Error("failed to load roles for home layout", zap.Error(err))
		return nil, nil, fmt.Errorf("load roles for home layout: %w", err)
	}

	presets, err := s.repo.ListPresets(
		ctx,
		&homelayoutports.ListHomeLayoutPresetsRequest{TenantInfo: req.TenantInfo},
	)
	if err != nil {
		s.l.Error("failed to list home layout presets", zap.Error(err))
		return nil, nil, err
	}

	return roles, presets, nil
}

// pickPreset takes the first preset assigned to any of the viewer's roles. The
// repository already returns presets highest-priority-first, so "first match"
// is "highest priority", and the org default is the fallback rather than a
// competitor — an explicit role assignment always outranks it.
func pickPreset(
	presets []*homelayout.Preset,
	roles []*rbac.Role,
) (*homelayout.Preset, Source) {
	var orgDefault *homelayout.Preset

	for _, preset := range presets {
		if preset.IsOrgDefault && orgDefault == nil {
			orgDefault = preset
		}

		for _, role := range roles {
			if preset.AppliesToRole(role.ID, role.CoreResponsibility) {
				return preset, SourceRolePreset
			}
		}
	}

	if orgDefault != nil {
		return orgDefault, SourceOrgDefault
	}

	return nil, SourceBuiltIn
}

// primaryResponsibility picks the core responsibility that decides which
// shipped layout a user lands on. Roles arrive ordered by the RBAC repository,
// so the first one carrying a responsibility wins and a user with none falls
// through to the general-purpose layout.
func primaryResponsibility(roles []*rbac.Role) rbac.CoreResponsibility {
	for _, role := range roles {
		if role.CoreResponsibility != "" {
			return role.CoreResponsibility
		}
	}
	return ""
}

// narrowed is a layout narrowed for one viewer, with the widgets and the
// metrics the plan leaves out, which narrowing keeps in place.
type narrowed struct {
	Layout *homelayout.Layout
	Locks  map[string]WidgetLock
}

// narrow drops widgets the viewer cannot read and locks the ones the plan does
// not include. A KPI widget is checked against the measure it points at as
// well as its own resource, so a dispatcher who cannot see receivables does
// not get an empty receivables tile.
func (s *Service) narrow(
	ctx context.Context,
	req *Request,
	layout *homelayout.Layout,
) (*narrowed, error) {
	access, err := s.checkCatalogAccess(ctx, req)
	if err != nil {
		return nil, err
	}

	widgets := make([]homelayout.Widget, 0, len(layout.Widgets))
	locks := make(map[string]WidgetLock)
	for _, widget := range layout.Widgets {
		definition, ok := homelayout.WidgetDefinitionFor(widget.Key)
		if !ok {
			continue
		}
		if denial := access.widgetDenial(definition); denial != nil {
			widgets = append(widgets, widget)
			locks[widget.ID] = WidgetLock{Widget: denial}
			continue
		}
		if !access.widgetAllowed(definition) {
			continue
		}

		var lock WidgetLock
		widget.Config, lock = narrowConfig(definition, widget.Config, access)
		if configEmptied(definition, widget.Config) {
			continue
		}
		if lock.locked() {
			locks[widget.ID] = lock
		}

		widgets = append(widgets, widget)
	}

	return &narrowed{Layout: &homelayout.Layout{Widgets: widgets}, Locks: locks}, nil
}

// narrowConfig drops the measures the viewer cannot read and notes the ones
// the plan leaves out. A widget whose every measure is left out by the plan is
// locked as a whole, with the first measure's add-on.
func narrowConfig(
	definition homelayout.WidgetDefinition,
	config homelayout.WidgetConfig,
	access *catalogAccess,
) (homelayout.WidgetConfig, WidgetLock) {
	var lock WidgetLock

	switch definition.ConfigKind {
	case homelayout.ConfigKindMetric:
		denial, allowed := access.metric(config.Metric)
		switch {
		case denial != nil:
			lock.Widget = denial
		case !allowed:
			config.Metric = ""
		}
	case homelayout.ConfigKindMetricRow:
		metrics := make([]string, 0, len(config.Metrics))
		for _, metric := range config.Metrics {
			denial, allowed := access.metric(metric)
			switch {
			case denial != nil:
				metrics = append(metrics, metric)
				lock.Metrics = append(lock.Metrics, MetricLock{Metric: metric, Denial: *denial})
			case allowed:
				metrics = append(metrics, metric)
			}
		}
		config.Metrics = metrics
		if len(metrics) > 0 && len(lock.Metrics) == len(metrics) {
			whole := lock.Metrics[0].Denial
			lock = WidgetLock{Widget: &whole}
		}
	case homelayout.ConfigKindNone,
		homelayout.ConfigKindQueue,
		homelayout.ConfigKindTrend,
		homelayout.ConfigKindReport,
		homelayout.ConfigKindDashboard,
		homelayout.ConfigKindText:
	}

	return config, lock
}

// configEmptied reports whether narrowing left a widget with nothing to show,
// which is the case for a metric widget whose every measure was filtered away.
func configEmptied(
	definition homelayout.WidgetDefinition,
	config homelayout.WidgetConfig,
) bool {
	switch definition.ConfigKind {
	case homelayout.ConfigKindMetric:
		return config.Metric == ""
	case homelayout.ConfigKindMetricRow:
		return len(config.Metrics) == 0
	default:
		return false
	}
}

type resourceOperation struct {
	resource  string
	operation permission.Operation
}

// catalogAccess is what one viewer may see of the widget and metric catalog:
// each resource the catalog references answered once, with the plan's denial
// when the plan is why, and each feature the catalog names asked of the plan
// once.
type catalogAccess struct {
	pairs    map[resourceOperation]permissionports.PermissionCheckResult
	features map[platformcatalog.FeatureKey]*permissionports.PlanDenial
}

func (a *catalogAccess) allowed(
	resource permission.Resource,
	operation permission.Operation,
) bool {
	if resource == "" {
		return true
	}
	return a.pairs[resourceOperation{resource: resource.String(), operation: operation}].Allowed
}

// resourceDenial is the plan's denial of a resource, nil when the plan includes
// it or when the viewer's own access is the reason.
func (a *catalogAccess) resourceDenial(
	resource permission.Resource,
	operation permission.Operation,
) *permissionports.PlanDenial {
	if resource == "" {
		return nil
	}
	return a.pairs[resourceOperation{resource: resource.String(), operation: operation}].PlanDenial
}

func (a *catalogAccess) featureDenial(
	feature platformcatalog.FeatureKey,
) *permissionports.PlanDenial {
	if feature == "" {
		return nil
	}
	return a.features[feature]
}

// widgetDenial is why the plan leaves a widget out: its resource's feature
// first, then the features of the further resources it reads, then the feature
// the widget names.
func (a *catalogAccess) widgetDenial(
	definition homelayout.WidgetDefinition,
) *permissionports.PlanDenial {
	if denial := a.resourceDenial(definition.Resource, definition.Operation); denial != nil {
		return denial
	}
	for _, resource := range definition.AlsoReads {
		if denial := a.resourceDenial(resource, permission.OpRead); denial != nil {
			return denial
		}
	}
	return a.featureDenial(definition.Feature)
}

func (a *catalogAccess) widgetAllowed(definition homelayout.WidgetDefinition) bool {
	if !a.allowed(definition.Resource, definition.Operation) {
		return false
	}
	for _, resource := range definition.AlsoReads {
		if !a.allowed(resource, permission.OpRead) {
			return false
		}
	}
	return true
}

// metric reports why the plan leaves a measure out, and otherwise whether the
// viewer may read it. An unknown measure is not allowed.
func (a *catalogAccess) metric(key string) (*permissionports.PlanDenial, bool) {
	metric, ok := homelayout.MetricDefinitionFor(key)
	if !ok {
		return nil, false
	}
	if denial := a.resourceDenial(metric.Resource, metric.Operation); denial != nil {
		return denial, false
	}
	if denial := a.featureDenial(metric.Feature); denial != nil {
		return denial, false
	}
	return nil, a.allowed(metric.Resource, metric.Operation)
}

// checkCatalogAccess asks the permission engine about every resource the
// catalog can reference in one batch, so resolving a home screen costs a single
// authorization round trip no matter how many widgets are on it, and asks the
// plan about each feature the catalog names.
func (s *Service) checkCatalogAccess(
	ctx context.Context,
	req *Request,
) (*catalogAccess, error) {
	widgetCatalog := homelayout.WidgetCatalog()
	metricCatalog := homelayout.MetricCatalog()

	pairs := make([]resourceOperation, 0, len(widgetCatalog)+len(metricCatalog))
	seen := make(map[resourceOperation]struct{}, len(widgetCatalog)+len(metricCatalog))
	features := make(map[platformcatalog.FeatureKey]struct{}, 4)

	addPair := func(resource permission.Resource, operation permission.Operation) {
		if resource == "" {
			return
		}
		pair := resourceOperation{resource: resource.String(), operation: operation}
		if _, ok := seen[pair]; ok {
			return
		}
		seen[pair] = struct{}{}
		pairs = append(pairs, pair)
	}

	for _, widget := range widgetCatalog {
		addPair(widget.Resource, widget.Operation)
		for _, resource := range widget.AlsoReads {
			addPair(resource, permission.OpRead)
		}
		if widget.Feature != "" {
			features[widget.Feature] = struct{}{}
		}
	}
	for _, metric := range metricCatalog {
		addPair(metric.Resource, metric.Operation)
		if metric.Feature != "" {
			features[metric.Feature] = struct{}{}
		}
	}

	checks := make([]permissionports.ResourceOperationCheck, 0, len(pairs))
	for _, pair := range pairs {
		checks = append(checks, permissionports.ResourceOperationCheck{
			Resource:  pair.resource,
			Operation: pair.operation,
		})
	}

	result, err := s.permissions.CheckBatch(ctx, &permissionports.BatchPermissionCheckRequest{
		PrincipalType:  req.Principal.Type,
		PrincipalID:    req.Principal.ID,
		UserID:         req.Principal.UserID,
		APIKeyID:       req.Principal.APIKeyID,
		BusinessUnitID: req.TenantInfo.BuID,
		OrganizationID: req.TenantInfo.OrgID,
		Checks:         checks,
	})
	if err != nil {
		return nil, fmt.Errorf("check home widget catalog permissions: %w", err)
	}

	access := &catalogAccess{
		pairs:    make(map[resourceOperation]permissionports.PermissionCheckResult, len(pairs)),
		features: make(map[platformcatalog.FeatureKey]*permissionports.PlanDenial, len(features)),
	}
	for idx, pair := range pairs {
		if idx < len(result.Results) {
			access.pairs[pair] = result.Results[idx]
		}
	}

	for feature := range features {
		enabled, err := s.features.Enabled(ctx, feature)
		if err != nil {
			return nil, fmt.Errorf("read whether the plan includes %s: %w", feature, err)
		}
		if !enabled {
			denial := permissionports.NewPlanDenial(feature)
			access.features[feature] = &denial
		}
	}

	return access, nil
}
