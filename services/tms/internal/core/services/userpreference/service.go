package userpreference

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/userpreference"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger *zap.Logger
	Repo   repositories.UserPreferenceRepository
}

type Service struct {
	l    *zap.Logger
	repo repositories.UserPreferenceRepository
}

func NewService(p ServiceParams) *Service {
	return &Service{
		l:    p.Logger.Named("service.userpreference"),
		repo: p.Repo,
	}
}

func (s *Service) GetOrCreateByUserID(
	ctx context.Context,
	userID, orgID, buID pulid.ID,
) (*userpreference.UserPreference, error) {
	log := s.l.With(
		zap.String("operation", "GetOrCreateByUserID"),
		zap.String("userID", userID.String()),
	)

	up, err := s.repo.GetOrCreateByUserID(ctx, userID, orgID, buID)
	if err != nil {
		log.Error("failed to get or create user preference", zap.Error(err))
		return nil, err
	}

	return up, nil
}

func (s *Service) Update(
	ctx context.Context,
	up *userpreference.UserPreference,
) (*userpreference.UserPreference, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("userID", up.UserID.String()),
	)

	multiErr := errortypes.NewMultiError()
	up.Validate(multiErr)

	if multiErr.HasErrors() {
		return nil, multiErr
	}

	updated, err := s.repo.Update(ctx, up)
	if err != nil {
		log.Error("failed to update user preference", zap.Error(err))
		return nil, err
	}

	return updated, nil
}

func (s *Service) Upsert(
	ctx context.Context,
	up *userpreference.UserPreference,
) (*userpreference.UserPreference, error) {
	log := s.l.With(
		zap.String("operation", "Upsert"),
		zap.String("userID", up.UserID.String()),
	)

	multiErr := errortypes.NewMultiError()
	up.Validate(multiErr)

	if multiErr.HasErrors() {
		return nil, multiErr
	}

	upserted, err := s.repo.Upsert(ctx, up)
	if err != nil {
		log.Error("failed to upsert user preference", zap.Error(err))
		return nil, err
	}

	return upserted, nil
}

// MergePreferences merges new preferences with existing ones
// This is useful for partial updates from the frontend
func (s *Service) MergePreferences(
	ctx context.Context,
	userID pulid.ID,
	updates *userpreference.PreferenceData,
) (*userpreference.UserPreference, error) {
	log := s.l.With(
		zap.String("operation", "MergePreferences"),
		zap.String("userID", userID.String()),
	)

	existing, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		log.Error("failed to get existing preferences", zap.Error(err))
		return nil, err
	}

	if updates.DismissedNotices != nil {
		existing.Preferences.DismissedNotices = utils.MergeUniqueStringSlices(
			existing.Preferences.DismissedNotices,
			updates.DismissedNotices,
		)
	}

	if updates.DismissedDialogs != nil {
		existing.Preferences.DismissedDialogs = utils.MergeUniqueStringSlices(
			existing.Preferences.DismissedDialogs,
			updates.DismissedDialogs,
		)
	}

	if updates.UISettings != nil {
		if existing.Preferences.UISettings == nil {
			existing.Preferences.UISettings = make(map[string]any)
		}
		for k, v := range updates.UISettings {
			existing.Preferences.UISettings[k] = v
		}
	}

	return s.Update(ctx, existing)
}

func (s *Service) AddDismissedNotice(
	ctx context.Context,
	userID pulid.ID,
	noticeKey string,
) (*userpreference.UserPreference, error) {
	log := s.l.With(
		zap.String("operation", "AddDismissedNotice"),
		zap.String("userID", userID.String()),
		zap.String("noticeKey", noticeKey),
	)

	up, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		log.Error("failed to get user preference", zap.Error(err))
		return nil, err
	}

	up.AddDismissed(noticeKey, false)

	return s.Update(ctx, up)
}

func (s *Service) AddDismissedDialog(
	ctx context.Context,
	userID pulid.ID,
	dialogKey string,
) (*userpreference.UserPreference, error) {
	log := s.l.With(
		zap.String("operation", "AddDismissedDialog"),
		zap.String("userID", userID.String()),
		zap.String("dialogKey", dialogKey),
	)

	up, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		log.Error("failed to get user preference", zap.Error(err))
		return nil, err
	}

	up.AddDismissed(dialogKey, true)

	return s.Update(ctx, up)
}

func (s *Service) SetUISetting(
	ctx context.Context,
	userID pulid.ID,
	key string,
	value any,
) (*userpreference.UserPreference, error) {
	log := s.l.With(
		zap.String("operation", "SetUISetting"),
		zap.String("userID", userID.String()),
		zap.String("key", key),
	)

	up, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		log.Error("failed to get user preference", zap.Error(err))
		return nil, err
	}

	up.SetUISetting(key, value)

	return s.Update(ctx, up)
}

func (s *Service) Delete(
	ctx context.Context,
	userID pulid.ID,
) error {
	log := s.l.With(
		zap.String("operation", "Delete"),
		zap.String("userID", userID.String()),
	)

	err := s.repo.Delete(ctx, userID)
	if err != nil {
		log.Error("failed to delete user preference", zap.Error(err))
		return err
	}

	return nil
}
