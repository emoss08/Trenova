package homelayoutservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homelayout"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/homelayoutports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Source records which tier of the resolution chain produced the layout on
// screen. The UI needs it to say what "reset" would restore and whether the
// user is looking at something of their own.
type Source string

const (
	SourceUser       = Source("user")
	SourceRolePreset = Source("rolePreset")
	SourceOrgDefault = Source("orgDefault")
	SourceBuiltIn    = Source("builtIn")
)

type Params struct {
	fx.In

	Logger           *zap.Logger
	Repo             homelayoutports.HomeLayoutRepository
	RBACRepo         permissionports.RBACRepository
	RoleRepo         permissionports.RoleRepository
	PermissionEngine permissionports.PermissionEngine
	Features         platformcatalogports.FeatureGate
}

type Service struct {
	l           *zap.Logger
	repo        homelayoutports.HomeLayoutRepository
	rbac        permissionports.RBACRepository
	roles       permissionports.RoleRepository
	permissions permissionports.PermissionEngine
	features    platformcatalogports.FeatureGate
}

func New(p Params) *Service {
	return &Service{
		l:           p.Logger.Named("service.homelayout"),
		repo:        p.Repo,
		rbac:        p.RBACRepo,
		roles:       p.RoleRepo,
		permissions: p.PermissionEngine,
		features:    p.Features,
	}
}

type Request struct {
	TenantInfo pagination.TenantInfo
	Principal  kernelports.PrincipalInfo
}

type UpdateRequest struct {
	Request

	Document *homelayout.Document
	Version  int64
}

// EffectiveLayout is what the home screen renders: a layout already narrowed to
// what this viewer may see, plus the provenance the UI needs to explain it.
type EffectiveLayout struct {
	Layout *homelayout.Layout
	// Locks holds, by widget id, the widgets and the measures the plan leaves
	// out. Such a widget stays in Layout, in its slot, and nothing about it is
	// read; a widget the viewer may not read is not in Layout at all.
	Locks        map[string]WidgetLock
	Density      string
	Source       Source
	PresetID     pulid.ID
	PresetName   string
	Locked       bool
	CanCustomize bool
	Version      int64
}

// WidgetLock is what the plan leaves out of one widget on a home screen.
// Widget is set when the whole widget is locked; Metrics names the measures of
// a metric strip that are locked while the rest of the strip is not.
type WidgetLock struct {
	Widget  *permissionports.PlanDenial
	Metrics []MetricLock
}

func (l WidgetLock) locked() bool {
	return l.Widget != nil || len(l.Metrics) > 0
}

type MetricLock struct {
	Metric string
	Denial permissionports.PlanDenial
}

// LockedWidget is a widget the viewer could add if the plan included it, and
// the add-on that carries it.
type LockedWidget struct {
	Definition homelayout.WidgetDefinition
	Denial     permissionports.PlanDenial
}

type LockedMetric struct {
	Definition homelayout.MetricDefinition
	Denial     permissionports.PlanDenial
}

// WidgetCatalog is what one viewer may choose from. Widgets and Metrics are
// the ones they can add now; LockedWidgets and LockedMetrics are the ones the
// plan leaves out, which a picker names with their add-on and offers no way
// to add.
type WidgetCatalog struct {
	Widgets       []homelayout.WidgetDefinition
	Metrics       []homelayout.MetricDefinition
	LockedWidgets []LockedWidget
	LockedMetrics []LockedMetric
	Categories    []homelayout.CategoryDefinition
	Densities     []string
	GridColumns   int
	MaxWidgets    int
}

func (s *Service) GetEffective(ctx context.Context, req *Request) (*EffectiveLayout, error) {
	log := s.l.With(
		zap.String("operation", "GetEffective"),
		zap.String("userID", req.TenantInfo.UserID.String()),
	)

	preference, found, err := s.repo.GetPreference(
		ctx,
		&homelayoutports.GetHomePreferenceRequest{TenantInfo: req.TenantInfo},
	)
	if err != nil {
		log.Error("failed to get home preference", zap.Error(err))
		return nil, err
	}

	document := homelayout.DefaultDocument()
	version := int64(0)
	if found {
		document = preference.Preferences.Normalize()
		version = preference.Version
	}

	resolved, err := s.resolve(ctx, req, document)
	if err != nil {
		return nil, err
	}
	resolved.Version = version

	return resolved, nil
}

func (s *Service) GetCatalog(ctx context.Context, req *Request) (*WidgetCatalog, error) {
	access, err := s.checkCatalogAccess(ctx, req)
	if err != nil {
		return nil, err
	}

	widgetCatalog := homelayout.WidgetCatalog()
	widgets := make([]homelayout.WidgetDefinition, 0, len(widgetCatalog))
	var lockedWidgets []LockedWidget
	for _, widget := range widgetCatalog {
		if denial := access.widgetDenial(widget); denial != nil {
			lockedWidgets = append(lockedWidgets, LockedWidget{Definition: widget, Denial: *denial})
			continue
		}
		if access.widgetAllowed(widget) {
			widgets = append(widgets, widget)
		}
	}

	metricCatalog := homelayout.MetricCatalog()
	metrics := make([]homelayout.MetricDefinition, 0, len(metricCatalog))
	var lockedMetrics []LockedMetric
	for _, metric := range metricCatalog {
		denial, allowed := access.metric(metric.Key)
		switch {
		case denial != nil:
			lockedMetrics = append(lockedMetrics, LockedMetric{Definition: metric, Denial: *denial})
		case allowed:
			metrics = append(metrics, metric)
		}
	}

	return &WidgetCatalog{
		Widgets:       widgets,
		Metrics:       metrics,
		LockedWidgets: lockedWidgets,
		LockedMetrics: lockedMetrics,
		Categories:    homelayout.CategoryCatalog(),
		Densities:     homelayout.Densities(),
		GridColumns:   homelayout.GridColumns,
		MaxWidgets:    homelayout.MaxWidgets,
	}, nil
}

// Update saves the viewer's own home screen. A viewer whose assigned preset is
// locked cannot save at all — the check happens here rather than in the UI so
// the lock holds against a direct API call.
func (s *Service) Update(ctx context.Context, req *UpdateRequest) (*EffectiveLayout, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("userID", req.TenantInfo.UserID.String()),
	)

	if req.Document == nil {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("preferences", errortypes.ErrRequired, "Preferences document is required")
		return nil, multiErr
	}

	multiErr := errortypes.NewMultiError()
	req.Document.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	roles, presets, err := s.loadPresetContext(ctx, &req.Request)
	if err != nil {
		return nil, err
	}

	assigned, _ := pickPreset(presets, roles)
	if assigned != nil && assigned.Locked {
		return nil, errortypes.NewAuthorizationError(
			"Your administrator manages this home screen",
		)
	}

	normalized := req.Document.Normalize()
	if normalized.Mode == homelayout.ModeCustom && assigned != nil {
		normalized.BasedOnPresetID = assigned.ID
	}

	saved, err := s.save(ctx, &req.Request, normalized, req.Version)
	if err != nil {
		log.Error("failed to save home preference", zap.Error(err))
		return nil, err
	}

	resolved, err := s.resolveWith(ctx, &req.Request, saved.Preferences.Normalize(), roles, presets)
	if err != nil {
		return nil, err
	}
	resolved.Version = saved.Version

	return resolved, nil
}

// Reset puts the viewer back on whatever they were assigned. The stored row is
// kept rather than deleted so the version chain stays continuous and a later
// customization does not collide with a stale client version.
func (s *Service) Reset(ctx context.Context, req *Request) (*EffectiveLayout, error) {
	log := s.l.With(
		zap.String("operation", "Reset"),
		zap.String("userID", req.TenantInfo.UserID.String()),
	)

	preference, found, err := s.repo.GetPreference(
		ctx,
		&homelayoutports.GetHomePreferenceRequest{TenantInfo: req.TenantInfo},
	)
	if err != nil {
		log.Error("failed to get home preference", zap.Error(err))
		return nil, err
	}

	if !found {
		return s.GetEffective(ctx, req)
	}

	document := homelayout.DefaultDocument()
	document.Density = preference.Preferences.Normalize().Density

	saved, err := s.save(ctx, req, document, preference.Version)
	if err != nil {
		log.Error("failed to reset home preference", zap.Error(err))
		return nil, err
	}

	resolved, err := s.resolve(ctx, req, saved.Preferences.Normalize())
	if err != nil {
		return nil, err
	}
	resolved.Version = saved.Version

	return resolved, nil
}

func (s *Service) save(
	ctx context.Context,
	req *Request,
	document *homelayout.Document,
	version int64,
) (*homelayout.Preference, error) {
	entity, found, err := s.repo.GetPreference(
		ctx,
		&homelayoutports.GetHomePreferenceRequest{TenantInfo: req.TenantInfo},
	)
	if err != nil {
		return nil, err
	}

	if !found {
		if version != 0 {
			return nil, dberror.CreateVersionMismatchError(
				"HomePreference",
				req.TenantInfo.UserID.String(),
			)
		}
		return s.repo.CreatePreference(ctx, &homelayout.Preference{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			UserID:         req.TenantInfo.UserID,
			Preferences:    document,
			Version:        1,
		})
	}

	entity.Version = version
	entity.Preferences = document

	return s.repo.UpdatePreference(ctx, entity)
}
