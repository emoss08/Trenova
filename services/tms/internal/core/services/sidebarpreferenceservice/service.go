package sidebarpreferenceservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/sidebarpreference"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger           *zap.Logger
	Repo             repositories.SidebarPreferenceRepository
	PermissionEngine services.PermissionEngine
}

type Service struct {
	l           *zap.Logger
	repo        repositories.SidebarPreferenceRepository
	permissions services.PermissionEngine
}

func New(p Params) *Service {
	return &Service{
		l:           p.Logger.Named("service.sidebarpreference"),
		repo:        p.Repo,
		permissions: p.PermissionEngine,
	}
}

type Request struct {
	TenantInfo pagination.TenantInfo
	Principal  services.PrincipalInfo
}

type UpdateRequest struct {
	Request

	Document *sidebarpreference.Document
	Version  int64
}

type EffectivePreferences struct {
	Document *sidebarpreference.Document
	Version  int64
}

type CustomizationOptions struct {
	Sections          []sidebarpreference.SectionDefinition
	AttentionMetrics  []sidebarpreference.AttentionMetricDefinition
	QuickActions      []sidebarpreference.QuickActionDefinition
	MaxQuickActions   int
	ActivityPageSizes []int
}

func (s *Service) GetEffective(
	ctx context.Context,
	req *Request,
) (*EffectivePreferences, error) {
	log := s.l.With(
		zap.String("operation", "GetEffective"),
		zap.String("userID", req.TenantInfo.UserID.String()),
	)

	entity, found, err := s.repo.Get(
		ctx,
		&repositories.GetSidebarPreferenceRequest{TenantInfo: req.TenantInfo},
	)
	if err != nil {
		log.Error("failed to get sidebar preference", zap.Error(err))
		return nil, err
	}

	doc := sidebarpreference.DefaultDocument()
	version := int64(0)
	if found {
		doc = entity.Preferences.Normalize()
		version = entity.Version
	}

	filtered, err := s.filterDocument(ctx, req, doc)
	if err != nil {
		return nil, err
	}

	return &EffectivePreferences{Document: filtered, Version: version}, nil
}

func (s *Service) GetOptions(
	ctx context.Context,
	req *Request,
) (*CustomizationOptions, error) {
	allowed, err := s.checkCatalogPermissions(ctx, req)
	if err != nil {
		return nil, err
	}

	metricCatalog := sidebarpreference.AttentionMetricCatalog()
	metrics := make([]sidebarpreference.AttentionMetricDefinition, 0, len(metricCatalog))
	for _, metric := range metricCatalog {
		if allowed(metric.Resource, permission.OpRead) {
			metrics = append(metrics, metric)
		}
	}

	actionCatalog := sidebarpreference.QuickActionCatalog()
	actions := make([]sidebarpreference.QuickActionDefinition, 0, len(actionCatalog))
	for _, action := range actionCatalog {
		if allowed(action.Resource, action.Operation) {
			actions = append(actions, action)
		}
	}

	return &CustomizationOptions{
		Sections:          sidebarpreference.SectionCatalog(),
		AttentionMetrics:  metrics,
		QuickActions:      actions,
		MaxQuickActions:   sidebarpreference.MaxQuickActions,
		ActivityPageSizes: sidebarpreference.ActivityPageSizes(),
	}, nil
}

func (s *Service) Update(
	ctx context.Context,
	req *UpdateRequest,
) (*EffectivePreferences, error) {
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

	normalized := req.Document.Normalize()

	entity, found, err := s.repo.Get(
		ctx,
		&repositories.GetSidebarPreferenceRequest{TenantInfo: req.TenantInfo},
	)
	if err != nil {
		log.Error("failed to get sidebar preference", zap.Error(err))
		return nil, err
	}

	var saved *sidebarpreference.SidebarPreference
	if found {
		entity.Version = req.Version
		entity.Preferences = normalized
		saved, err = s.repo.Update(ctx, entity)
	} else {
		if req.Version != 0 {
			return nil, dberror.CreateVersionMismatchError(
				"SidebarPreference",
				req.TenantInfo.UserID.String(),
			)
		}
		saved, err = s.repo.Create(ctx, &sidebarpreference.SidebarPreference{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			UserID:         req.TenantInfo.UserID,
			Preferences:    normalized,
			Version:        1,
		})
	}
	if err != nil {
		log.Error("failed to save sidebar preference", zap.Error(err))
		return nil, err
	}

	filtered, err := s.filterDocument(ctx, &req.Request, saved.Preferences.Normalize())
	if err != nil {
		return nil, err
	}

	return &EffectivePreferences{Document: filtered, Version: saved.Version}, nil
}

func (s *Service) filterDocument(
	ctx context.Context,
	req *Request,
	doc *sidebarpreference.Document,
) (*sidebarpreference.Document, error) {
	allowed, err := s.checkCatalogPermissions(ctx, req)
	if err != nil {
		return nil, err
	}

	metrics := make([]string, 0, len(doc.AttentionMetrics))
	for _, key := range doc.AttentionMetrics {
		for _, metric := range sidebarpreference.AttentionMetricCatalog() {
			if metric.Key == key && allowed(metric.Resource, permission.OpRead) {
				metrics = append(metrics, key)
				break
			}
		}
	}
	doc.AttentionMetrics = metrics

	actions := make([]string, 0, len(doc.QuickActionIDs))
	for _, id := range doc.QuickActionIDs {
		for _, action := range sidebarpreference.QuickActionCatalog() {
			if action.ID == id && allowed(action.Resource, action.Operation) {
				actions = append(actions, id)
				break
			}
		}
	}
	doc.QuickActionIDs = actions

	return doc, nil
}

type resourceOperation struct {
	resource  string
	operation permission.Operation
}

func (s *Service) checkCatalogPermissions(
	ctx context.Context,
	req *Request,
) (func(permission.Resource, permission.Operation) bool, error) {
	metricCatalog := sidebarpreference.AttentionMetricCatalog()
	actionCatalog := sidebarpreference.QuickActionCatalog()

	pairs := make([]resourceOperation, 0, len(metricCatalog)+len(actionCatalog))
	seen := make(map[resourceOperation]struct{}, len(metricCatalog)+len(actionCatalog))

	addPair := func(resource permission.Resource, operation permission.Operation) {
		pair := resourceOperation{resource: resource.String(), operation: operation}
		if _, ok := seen[pair]; ok {
			return
		}
		seen[pair] = struct{}{}
		pairs = append(pairs, pair)
	}

	for _, metric := range metricCatalog {
		addPair(metric.Resource, permission.OpRead)
	}
	for _, action := range actionCatalog {
		addPair(action.Resource, action.Operation)
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
		return nil, fmt.Errorf("check sidebar catalog permissions: %w", err)
	}

	allowedPairs := make(map[resourceOperation]bool, len(pairs))
	for idx, pair := range pairs {
		allowedPairs[pair] = idx < len(result.Results) && result.Results[idx].Allowed
	}

	return func(resource permission.Resource, operation permission.Operation) bool {
		return allowedPairs[resourceOperation{resource: resource.String(), operation: operation}]
	}, nil
}
