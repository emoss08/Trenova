package homelayoutservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/homelayout"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
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
	Repo             repositories.HomeLayoutRepository
	RBACRepo         repositories.RBACRepository
	RoleRepo         repositories.RoleRepository
	PermissionEngine services.PermissionEngine
}

type Service struct {
	l           *zap.Logger
	repo        repositories.HomeLayoutRepository
	rbac        repositories.RBACRepository
	roles       repositories.RoleRepository
	permissions services.PermissionEngine
}

func New(p Params) *Service {
	return &Service{
		l:           p.Logger.Named("service.homelayout"),
		repo:        p.Repo,
		rbac:        p.RBACRepo,
		roles:       p.RoleRepo,
		permissions: p.PermissionEngine,
	}
}

type Request struct {
	TenantInfo pagination.TenantInfo
	Principal  services.PrincipalInfo
}

type UpdateRequest struct {
	Request

	Document *homelayout.Document
	Version  int64
}

// EffectiveLayout is what the home screen renders: a layout already narrowed to
// what this viewer may see, plus the provenance the UI needs to explain it.
type EffectiveLayout struct {
	Layout       *homelayout.Layout
	Density      string
	Source       Source
	PresetID     pulid.ID
	PresetName   string
	Locked       bool
	CanCustomize bool
	Version      int64
}

type WidgetCatalog struct {
	Widgets     []homelayout.WidgetDefinition
	Metrics     []homelayout.MetricDefinition
	Categories  []homelayout.CategoryDefinition
	Densities   []string
	GridColumns int
	MaxWidgets  int
}

func (s *Service) GetEffective(ctx context.Context, req *Request) (*EffectiveLayout, error) {
	log := s.l.With(
		zap.String("operation", "GetEffective"),
		zap.String("userID", req.TenantInfo.UserID.String()),
	)

	preference, found, err := s.repo.GetPreference(
		ctx,
		&repositories.GetHomePreferenceRequest{TenantInfo: req.TenantInfo},
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
	allowed, err := s.checkCatalogPermissions(ctx, req)
	if err != nil {
		return nil, err
	}

	widgetCatalog := homelayout.WidgetCatalog()
	widgets := make([]homelayout.WidgetDefinition, 0, len(widgetCatalog))
	for _, widget := range widgetCatalog {
		if !widget.RequiresPermission() || allowed(widget.Resource, widget.Operation) {
			widgets = append(widgets, widget)
		}
	}

	metricCatalog := homelayout.MetricCatalog()
	metrics := make([]homelayout.MetricDefinition, 0, len(metricCatalog))
	for _, metric := range metricCatalog {
		if allowed(metric.Resource, metric.Operation) {
			metrics = append(metrics, metric)
		}
	}

	return &WidgetCatalog{
		Widgets:     widgets,
		Metrics:     metrics,
		Categories:  homelayout.CategoryCatalog(),
		Densities:   homelayout.Densities(),
		GridColumns: homelayout.GridColumns,
		MaxWidgets:  homelayout.MaxWidgets,
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
		&repositories.GetHomePreferenceRequest{TenantInfo: req.TenantInfo},
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
		&repositories.GetHomePreferenceRequest{TenantInfo: req.TenantInfo},
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
