/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package consolidationsetting

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/consolidation"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/consolidationsettingvalidator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

// ServiceParams defines the dependencies required for initializing the Service.
// This includes a logger, consolidation settings repository, permission service, audit service, and validator.
type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.ConsolidationSettingRepository
	PermService  services.PermissionService
	AuditService services.AuditService
	Validator    *consolidationsettingvalidator.Validator
}

// Service is a service that manages consolidation settings entities.
// It provides methods to get and update consolidation settings entities.
type Service struct {
	l    *zerolog.Logger
	repo repositories.ConsolidationSettingRepository
	ps   services.PermissionService
	as   services.AuditService
	v    *consolidationsettingvalidator.Validator
}

// NewService initializes a new instance of service with its dependencies.
//
// Parameters:
//   - p: ServiceParams containing logger, consolidation settings repository, permission service, audit service, and validator.
//
// Returns:
//   - A new instance of service.
func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "consolidationsetting").
		Logger()

	return &Service{
		l:    &log,
		repo: p.Repo,
		ps:   p.PermService,
		as:   p.AuditService,
		v:    p.Validator,
	}
}

// Get returns a consolidation settings
//
// Parameters:
//   - ctx: The context for the operation.
//   - req: GetConsolidationSettingRequest containing orgID, buID, and userID.
//
// Returns:
//   - *consolidation.ConsolidationSettings: The consolidation settings entity.
//   - error: If any database operation fails.
func (s *Service) Get(
	ctx context.Context,
	req *repositories.GetConsolidationSettingRequest,
) (*consolidation.ConsolidationSettings, error) {
	log := s.l.With().
		Str("operation", "Get").
		Str("orgID", req.OrgID.String()).
		Str("buID", req.BuID.String()).
		Logger()

	// * Check if the user has permission to read the consolidation settings
	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         req.UserID,
				Resource:       permission.ResourceConsolidationSettings,
				Action:         permission.ActionRead,
				BusinessUnitID: req.BuID,
				OrganizationID: req.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	// * If the user does not have permission to read the consolidation settings, return an error
	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read consolidation settings",
		)
	}

	// * Get the consolidation settings by organization ID
	entity, err := s.repo.GetByOrgID(ctx, req.OrgID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get consolidation settings")
		return nil, err
	}

	return entity, nil
}

// Update updates a consolidation settings
//
// Parameters:
//   - ctx: The context for the operation.
//   - sc: The consolidation settings entity to update.
//   - userID: The user ID of the user updating the consolidation settings.
//
// Returns:
//   - *consolidation.ConsolidationSettings: The updated consolidation settings entity.
//   - error: If any database operation fails.
func (s *Service) Update(
	ctx context.Context,
	cs *consolidation.ConsolidationSettings,
	userID pulid.ID,
) (*consolidation.ConsolidationSettings, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("orgID", cs.OrganizationID.String()).
		Str("buID", cs.BusinessUnitID.String()).
		Logger()

	// * Check if the user has permission to update the consolidation settings
	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceConsolidationSettings,
				Action:         permission.ActionUpdate,
				BusinessUnitID: cs.BusinessUnitID,
				OrganizationID: cs.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	// * Check if the user has permission to update the consolidation settings
	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to update consolidation settings",
		)
	}

	// * Create a validation context for the consolidation settings
	// * IsUpdate is true because we are updating an existing entity
	valCtx := &validator.ValidationContext{
		IsUpdate: true,
		IsCreate: false,
	}

	// * Validate the consolidation settings
	if err := s.v.Validate(ctx, valCtx, cs); err != nil {
		return nil, err
	}

	// * Get the original consolidation settings for comparison when logging the action
	original, err := s.repo.GetByOrgID(ctx, cs.OrganizationID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get consolidation settings")
		return nil, err
	}

	// * Update the consolidation settings
	updatedEntity, err := s.repo.Update(ctx, cs)
	if err != nil {
		log.Error().Err(err).Msg("failed to update consolidation settings")
		return nil, err
	}

	// * Log the action
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceConsolidationSettings,
			ResourceID:     updatedEntity.GetID(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Consolidation Settings updated"),
		audit.WithDiff(original, updatedEntity),
		audit.WithCritical(),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log action")
		return nil, err
	}

	return updatedEntity, nil
}
