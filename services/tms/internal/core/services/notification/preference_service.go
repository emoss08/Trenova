/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package notification

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/notificationpreferencevalidator"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type PreferenceServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.NotificationPreferenceRepository
	PermService  services.PermissionService
	AuditService services.AuditService
	Validator    *notificationpreferencevalidator.Validator
}

type PreferenceService struct {
	l    *zerolog.Logger
	repo repositories.NotificationPreferenceRepository
	ps   services.PermissionService
	as   services.AuditService
	v    *notificationpreferencevalidator.Validator
}

func NewPreferenceService(p PreferenceServiceParams) *PreferenceService {
	log := p.Logger.With().
		Str("service", "notification_preference").
		Logger()

	return &PreferenceService{
		l:    &log,
		repo: p.Repo,
		ps:   p.PermService,
		as:   p.AuditService,
		v:    p.Validator,
	}
}

func (s *PreferenceService) List(
	ctx context.Context,
	opts *ports.LimitOffsetQueryOptions,
) (*ports.ListResult[*notification.NotificationPreference], error) {
	log := s.l.With().Str("operation", "List").Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         opts.TenantOpts.UserID,
			Resource:       permission.ResourceUser,
			Action:         permission.ActionRead,
			BusinessUnitID: opts.TenantOpts.BuID,
			OrganizationID: opts.TenantOpts.OrgID,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read notification preferences",
		)
	}

	entities, err := s.repo.List(ctx, repositories.ListNotificationPreferencesRequest{
		Filter: opts,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to list notification preferences")
		return nil, err
	}

	return entities, nil
}

func (s *PreferenceService) GetByID(
	ctx context.Context,
	id pulid.ID,
) (*notification.NotificationPreference, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *PreferenceService) Create(
	ctx context.Context,
	pref *notification.NotificationPreference,
	userID pulid.ID,
) (*notification.NotificationPreference, error) {
	log := s.l.With().
		Str("operation", "Create").
		Str("resource", string(pref.Resource)).
		Logger()

	// Users can create their own preferences
	if pref.UserID != userID {
		result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceUser,
				Action:         permission.ActionManage,
				BusinessUnitID: pref.BusinessUnitID,
				OrganizationID: pref.OrganizationID,
			},
		})
		if err != nil {
			log.Error().Err(err).Msg("failed to check permissions")
			return nil, err
		}

		if !result.Allowed {
			return nil, errors.NewAuthorizationError(
				"You do not have permission to create notification preferences for other users",
			)
		}
	}

	// Validate the preference
	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.v.Validate(ctx, valCtx, pref); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, pref)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceUser,
			ResourceID:     createdEntity.ID.String(),
			Action:         permission.ActionCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Notification preference created"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log notification preference creation")
	}

	return createdEntity, nil
}

func (s *PreferenceService) Update(
	ctx context.Context,
	pref *notification.NotificationPreference,
	userID pulid.ID,
) (*notification.NotificationPreference, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("preferenceID", pref.ID.String()).
		Logger()

	// Get the original preference
	original, err := s.repo.GetByID(ctx, pref.ID)
	if err != nil {
		return nil, err
	}

	// Users can update their own preferences
	if original.UserID != userID {
		result, permErr := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceUser,
				Action:         permission.ActionManage,
				BusinessUnitID: pref.BusinessUnitID,
				OrganizationID: pref.OrganizationID,
			},
		})
		if permErr != nil {
			log.Error().Err(permErr).Msg("failed to check permissions")
			return nil, permErr
		}

		if !result.Allowed {
			return nil, errors.NewAuthorizationError(
				"You do not have permission to update this notification preference",
			)
		}
	}

	// Validate the preference
	valCtx := &validator.ValidationContext{
		IsCreate: false,
		IsUpdate: true,
	}

	if err = s.v.Validate(ctx, valCtx, pref); err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, pref)
	if err != nil {
		log.Error().Err(err).Msg("failed to update notification preference")
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceUser,
			ResourceID:     updatedEntity.ID.String(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Notification preference updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log notification preference update")
	}

	return updatedEntity, nil
}

func (s *PreferenceService) Delete(
	ctx context.Context,
	id pulid.ID,
	userID pulid.ID,
) error {
	log := s.l.With().
		Str("operation", "Delete").
		Str("preferenceID", id.String()).
		Logger()

	// Get the preference to check ownership
	pref, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Users can delete their own preferences
	if pref.UserID != userID {
		result, permErr := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceUser,
				Action:         permission.ActionManage,
				BusinessUnitID: pref.BusinessUnitID,
				OrganizationID: pref.OrganizationID,
			},
		})
		if permErr != nil {
			log.Error().Err(permErr).Msg("failed to check permissions")
			return permErr
		}

		if !result.Allowed {
			return errors.NewAuthorizationError(
				"You do not have permission to delete this notification preference",
			)
		}
	}

	if err = s.repo.Delete(ctx, id); err != nil {
		log.Error().Err(err).Msg("failed to delete notification preference")
		return err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceUser,
			ResourceID:     pref.ID.String(),
			Action:         permission.ActionDelete,
			UserID:         userID,
			PreviousState:  jsonutils.MustToJSON(pref),
			OrganizationID: pref.OrganizationID,
			BusinessUnitID: pref.BusinessUnitID,
		},
		audit.WithComment("Notification preference deleted"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log notification preference deletion")
	}

	return nil
}

func (s *PreferenceService) GetUserPreferences(
	ctx context.Context,
	userID pulid.ID,
	orgID pulid.ID,
) ([]*notification.NotificationPreference, error) {
	log := s.l.With().
		Str("operation", "GetUserPreferences").
		Str("userID", userID.String()).
		Logger()

	prefs, err := s.repo.GetUserPreferences(ctx, &repositories.GetUserPreferencesRequest{
		UserID:         userID,
		OrganizationID: orgID,
		IsActive:       true,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get user preferences")
		return nil, err
	}

	return prefs, nil
}

// HasManagePermission checks if a user has manage permissions
func (s *PreferenceService) HasManagePermission(
	ctx context.Context,
	userID,
	orgID,
	buID pulid.ID,
) (bool, error) {
	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         userID,
			Resource:       permission.ResourceUser,
			Action:         permission.ActionManage,
			BusinessUnitID: buID,
			OrganizationID: orgID,
		},
	})
	if err != nil {
		return false, err
	}

	return result.Allowed, nil
}
