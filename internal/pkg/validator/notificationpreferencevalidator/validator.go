package notificationpreferencevalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Logger *logger.Logger
	Repo   repositories.NotificationPreferenceRepository
}

type Validator struct {
	l    *zerolog.Logger
	repo repositories.NotificationPreferenceRepository
}

func NewValidator(p ValidatorParams) *Validator {
	log := p.Logger.With().Str("validator", "notificationpreference").Logger()
	return &Validator{
		l:    &log,
		repo: p.Repo,
	}
}

func (v *Validator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	entity any,
) error {
	pref, ok := entity.(*notification.NotificationPreference)
	if !ok {
		return errors.NewValidationError("entity", errors.ErrInvalid, "expected NotificationPreference")
	}

	multiErr := errors.NewMultiError()

	// Basic validation
	pref.Validate(ctx, multiErr)

	// Additional business logic validation
	if valCtx.IsCreate {
		if err := v.validateCreate(ctx, pref, multiErr); err != nil {
			return err
		}
	}

	if valCtx.IsUpdate {
		if err := v.validateUpdate(ctx, pref, multiErr); err != nil {
			return err
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (v *Validator) validateCreate(
	ctx context.Context,
	pref *notification.NotificationPreference,
	multiErr *errors.MultiError,
) error {
	// Check for duplicate preferences (same user, entity type, and organization)
	existing, err := v.repo.GetUserPreferences(ctx, &repositories.GetUserPreferencesRequest{
		UserID:         pref.UserID,
		OrganizationID: pref.OrganizationID,
		EntityType:     pref.EntityType,
		IsActive:       true,
	})
	if err != nil {
		return err
	}

	if len(existing) > 0 {
		multiErr.Add("entityType", errors.ErrDuplicate,
			"An active notification preference already exists for this entity type")
	}

	return nil
}

func (v *Validator) validateUpdate(
	ctx context.Context,
	pref *notification.NotificationPreference,
	multiErr *errors.MultiError,
) error {
	// Check if preference exists
	original, err := v.repo.GetByID(ctx, pref.ID)
	if err != nil {
		return err
	}

	// Ensure user and organization cannot be changed
	if original.UserID != pref.UserID {
		multiErr.Add("userId", errors.ErrInvalid, "User ID cannot be changed")
	}

	if original.OrganizationID != pref.OrganizationID {
		multiErr.Add("organizationId", errors.ErrInvalid, "Organization ID cannot be changed")
	}

	// Check for duplicate if entity type is being changed
	if original.EntityType != pref.EntityType {
		existing, err := v.repo.GetUserPreferences(ctx, &repositories.GetUserPreferencesRequest{
			UserID:         pref.UserID,
			OrganizationID: pref.OrganizationID,
			EntityType:     pref.EntityType,
			IsActive:       true,
		})
		if err != nil {
			return err
		}

		for _, e := range existing {
			if e.ID != pref.ID {
				multiErr.Add("entityType", errors.ErrDuplicate,
					"An active notification preference already exists for this entity type")
				break
			}
		}
	}

	return nil
}
