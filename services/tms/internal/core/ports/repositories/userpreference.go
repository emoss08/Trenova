package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/userpreference"
	"github.com/emoss08/trenova/pkg/pulid"
)

type UserPreferenceRepository interface {
	GetOrCreateByUserID(
		ctx context.Context,
		userID, orgID, buID pulid.ID,
	) (*userpreference.UserPreference, error)
	GetByUserID(ctx context.Context, userID pulid.ID) (*userpreference.UserPreference, error)
	Update(
		ctx context.Context,
		up *userpreference.UserPreference,
	) (*userpreference.UserPreference, error)
	Upsert(
		ctx context.Context,
		up *userpreference.UserPreference,
	) (*userpreference.UserPreference, error)
	Delete(ctx context.Context, userID pulid.ID) error
}

type UserPreferenceCacheRepository interface {
	GetByUserID(ctx context.Context, userID pulid.ID) (*userpreference.UserPreference, error)
	Set(ctx context.Context, up *userpreference.UserPreference) error
	Invalidate(ctx context.Context, userID pulid.ID) error
}
