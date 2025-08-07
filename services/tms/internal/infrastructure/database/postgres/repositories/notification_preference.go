/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"
	"database/sql"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils/queryfilters"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type NotificationPreferenceRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type notificationPreferenceRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewNotificationPreferenceRepository(
	p NotificationPreferenceRepositoryParams,
) repositories.NotificationPreferenceRepository {
	log := p.Logger.With().
		Str("repository", "notification_preference").
		Logger()

	return &notificationPreferenceRepository{
		db: p.DB,
		l:  &log,
	}
}

// filterQuery applies filters to the notification preference query
func (npr *notificationPreferenceRepository) filterQuery(
	q *bun.SelectQuery,
	opts *ports.LimitOffsetQueryOptions,
) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "np",
		Filter:     opts,
	})

	// Order by created date
	q = q.Order("np.created_at DESC")

	return q.Limit(opts.Limit).Offset(opts.Offset)
}

// Create creates a new notification preference
func (npr *notificationPreferenceRepository) Create(
	ctx context.Context,
	pref *notification.NotificationPreference,
) (*notification.NotificationPreference, error) {
	dba, err := npr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := npr.l.With().
		Str("operation", "Create").
		Str("userID", pref.UserID.String()).
		Str("resource", string(pref.Resource)).
		Logger()

	if _, err = dba.NewInsert().Model(pref).Returning("*").Exec(ctx); err != nil {
		log.Error().
			Err(err).
			Interface("preference", pref).
			Msg("failed to insert notification preference")
		return nil, eris.Wrap(err, "insert notification preference")
	}

	log.Info().Msg("notification preference created successfully")
	return pref, nil
}

// Update updates an existing notification preference
func (npr *notificationPreferenceRepository) Update(
	ctx context.Context,
	pref *notification.NotificationPreference,
) (*notification.NotificationPreference, error) {
	dba, err := npr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := npr.l.With().
		Str("operation", "Update").
		Str("preferenceID", pref.ID.String()).
		Logger()

	result, err := dba.NewUpdate().
		Model(pref).
		WherePK().
		Where("np.version = ?", pref.Version-1). // Optimistic locking
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error().
			Err(err).
			Interface("preference", pref).
			Msg("failed to update notification preference")
		return nil, eris.Wrap(err, "update notification preference")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, eris.Wrap(err, "get rows affected")
	}

	if rowsAffected == 0 {
		return nil, errors.NewBusinessError("notification preference was modified by another user").
			WithParam("id", pref.ID.String())
	}

	log.Info().Msg("notification preference updated successfully")
	return pref, nil
}

// Delete deletes a notification preference
func (npr *notificationPreferenceRepository) Delete(
	ctx context.Context,
	id pulid.ID,
) error {
	dba, err := npr.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	log := npr.l.With().
		Str("operation", "Delete").
		Str("preferenceID", id.String()).
		Logger()

	result, err := dba.NewDelete().
		Model((*notification.NotificationPreference)(nil)).
		Where("np.id = ?", id).
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete notification preference")
		return eris.Wrap(err, "delete notification preference")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return eris.Wrap(err, "get rows affected")
	}

	if rowsAffected == 0 {
		return errors.NewNotFoundError("notification preference not found")
	}

	log.Info().Msg("notification preference deleted successfully")
	return nil
}

// GetByID retrieves a notification preference by ID
func (npr *notificationPreferenceRepository) GetByID(
	ctx context.Context,
	id pulid.ID,
) (*notification.NotificationPreference, error) {
	dba, err := npr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := npr.l.With().
		Str("operation", "GetByID").
		Str("preferenceID", id.String()).
		Logger()

	pref := new(notification.NotificationPreference)
	err = dba.NewSelect().
		Model(pref).
		Where("np.id = ?", id).
		Scan(ctx)
	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Info().Msg("notification preference not found")
			return nil, errors.NewNotFoundError("notification preference not found")
		}
		log.Error().Err(err).Msg("failed to get notification preference")
		return nil, eris.Wrap(err, "get notification preference")
	}

	log.Info().Msg("notification preference retrieved successfully")
	return pref, nil
}

// GetUserPreferences retrieves notification preferences for a user
func (npr *notificationPreferenceRepository) GetUserPreferences(
	ctx context.Context,
	req *repositories.GetUserPreferencesRequest,
) ([]*notification.NotificationPreference, error) {
	dba, err := npr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := npr.l.With().
		Str("operation", "GetUserPreferences").
		Str("userID", req.UserID.String()).
		Str("resource", string(req.Resource)).
		Logger()

	prefs := make([]*notification.NotificationPreference, 0)

	q := dba.NewSelect().Model(&prefs).
		Where("np.user_id = ?", req.UserID).
		Where("np.organization_id = ?", req.OrganizationID)

	if req.Resource != "" {
		q = q.Where("np.resource = ?", req.Resource)
	}

	if req.IsActive {
		q = q.Where("np.is_active = ?", req.IsActive)
	}

	q = q.Order("np.created_at DESC")

	if err = q.Scan(ctx); err != nil {
		log.Error().Err(err).Msg("failed to get user preferences")
		return nil, eris.Wrap(err, "get user preferences")
	}

	log.Info().Int("count", len(prefs)).Msg("user preferences retrieved successfully")
	return prefs, nil
}

// List retrieves all notification preferences with filtering
func (npr *notificationPreferenceRepository) List(
	ctx context.Context,
	req repositories.ListNotificationPreferencesRequest,
) (*ports.ListResult[*notification.NotificationPreference], error) {
	dba, err := npr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := npr.l.With().
		Str("operation", "List").
		Logger()

	prefs := make([]*notification.NotificationPreference, 0)

	q := dba.NewSelect().Model(&prefs)
	q = npr.filterQuery(q, req.Filter)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to list notification preferences")
		return nil, eris.Wrap(err, "list notification preferences")
	}

	return &ports.ListResult[*notification.NotificationPreference]{
		Items: prefs,
		Total: total,
	}, nil
}
