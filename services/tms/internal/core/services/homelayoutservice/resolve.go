package homelayoutservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/homelayout"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
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
// fewer cards, never to an error.
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
	roles []*permission.Role,
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
	effective.Layout = narrowed

	return effective, nil
}

// loadPresetContext fetches the two inputs preset resolution needs: the
// viewer's roles and the tenant's presets.
func (s *Service) loadPresetContext(
	ctx context.Context,
	req *Request,
) ([]*permission.Role, []*homelayout.Preset, error) {
	roles, err := s.rbac.GetAuthorizedRoles(ctx, req.TenantInfo.UserID, req.TenantInfo.OrgID)
	if err != nil {
		s.l.Error("failed to load roles for home layout", zap.Error(err))
		return nil, nil, fmt.Errorf("load roles for home layout: %w", err)
	}

	presets, err := s.repo.ListPresets(
		ctx,
		&repositories.ListHomeLayoutPresetsRequest{TenantInfo: req.TenantInfo},
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
	roles []*permission.Role,
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
func primaryResponsibility(roles []*permission.Role) permission.CoreResponsibility {
	for _, role := range roles {
		if role.CoreResponsibility != "" {
			return role.CoreResponsibility
		}
	}
	return ""
}

// narrow drops widgets the viewer cannot read. A KPI widget is checked against
// the measure it points at as well as its own resource, so a dispatcher who
// cannot see receivables does not get an empty receivables tile.
func (s *Service) narrow(
	ctx context.Context,
	req *Request,
	layout *homelayout.Layout,
) (*homelayout.Layout, error) {
	allowed, err := s.checkCatalogPermissions(ctx, req)
	if err != nil {
		return nil, err
	}

	widgets := make([]homelayout.Widget, 0, len(layout.Widgets))
	for _, widget := range layout.Widgets {
		definition, ok := homelayout.WidgetDefinitionFor(widget.Key)
		if !ok {
			continue
		}
		if definition.RequiresPermission() &&
			!allowed(definition.Resource, definition.Operation) {
			continue
		}

		widget.Config = narrowConfig(definition, widget.Config, allowed)
		if configEmptied(definition, widget.Config) {
			continue
		}

		widgets = append(widgets, widget)
	}

	return &homelayout.Layout{Widgets: widgets}, nil
}

func narrowConfig(
	definition homelayout.WidgetDefinition,
	config homelayout.WidgetConfig,
	allowed func(permission.Resource, permission.Operation) bool,
) homelayout.WidgetConfig {
	switch definition.ConfigKind {
	case homelayout.ConfigKindMetric:
		if !metricAllowed(config.Metric, allowed) {
			config.Metric = ""
		}
	case homelayout.ConfigKindMetricRow:
		metrics := make([]string, 0, len(config.Metrics))
		for _, metric := range config.Metrics {
			if metricAllowed(metric, allowed) {
				metrics = append(metrics, metric)
			}
		}
		config.Metrics = metrics
	case homelayout.ConfigKindNone,
		homelayout.ConfigKindQueue,
		homelayout.ConfigKindTrend,
		homelayout.ConfigKindReport,
		homelayout.ConfigKindDashboard,
		homelayout.ConfigKindText:
	}

	return config
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

func metricAllowed(
	key string,
	allowed func(permission.Resource, permission.Operation) bool,
) bool {
	metric, ok := homelayout.MetricDefinitionFor(key)
	if !ok {
		return false
	}
	return allowed(metric.Resource, metric.Operation)
}

type resourceOperation struct {
	resource  string
	operation permission.Operation
}

// checkCatalogPermissions asks the permission engine about every resource the
// catalog can reference in one batch, so resolving a home screen costs a single
// authorization round trip no matter how many widgets are on it.
func (s *Service) checkCatalogPermissions(
	ctx context.Context,
	req *Request,
) (func(permission.Resource, permission.Operation) bool, error) {
	widgetCatalog := homelayout.WidgetCatalog()
	metricCatalog := homelayout.MetricCatalog()

	pairs := make([]resourceOperation, 0, len(widgetCatalog)+len(metricCatalog))
	seen := make(map[resourceOperation]struct{}, len(widgetCatalog)+len(metricCatalog))

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
	}
	for _, metric := range metricCatalog {
		addPair(metric.Resource, metric.Operation)
	}

	checks := make([]services.ResourceOperationCheck, 0, len(pairs))
	for _, pair := range pairs {
		checks = append(checks, services.ResourceOperationCheck{
			Resource:  pair.resource,
			Operation: pair.operation,
		})
	}

	result, err := s.permissions.CheckBatch(ctx, &services.BatchPermissionCheckRequest{
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

	allowedPairs := make(map[resourceOperation]bool, len(pairs))
	for idx, pair := range pairs {
		allowedPairs[pair] = idx < len(result.Results) && result.Results[idx].Allowed
	}

	return func(resource permission.Resource, operation permission.Operation) bool {
		if resource == "" {
			return true
		}
		return allowedPairs[resourceOperation{resource: resource.String(), operation: operation}]
	}, nil
}
