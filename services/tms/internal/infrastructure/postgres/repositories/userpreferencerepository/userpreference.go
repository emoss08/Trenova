package userpreferencerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/userpreference"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewRepository(p Params) repositories.UserPreferenceRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.userpreference-repository"),
	}
}

func (r *repository) GetOrCreateByUserID(
	ctx context.Context,
	userID, orgID, buID pulid.ID,
) (*userpreference.UserPreference, error) {
	log := r.l.With(
		zap.String("operation", "GetOrCreateByUserID"),
		zap.String("userID", userID.String()),
	)

	up, err := r.GetByUserID(ctx, userID)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return r.create(ctx, &userpreference.UserPreference{
				UserID:         userID,
				OrganizationID: orgID,
				BusinessUnitID: buID,
				Preferences: userpreference.PreferenceData{
					DismissedNotices: []string{},
					DismissedDialogs: []string{},
					UISettings:       make(map[string]any),
				},
			})
		}

		log.Error("failed to get user preference", zap.Error(err))
		return nil, err
	}

	return up, nil
}

func (r *repository) GetByUserID(
	ctx context.Context,
	userID pulid.ID,
) (*userpreference.UserPreference, error) {
	log := r.l.With(
		zap.String("operation", "GetByUserID"),
		zap.String("userID", userID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	up := new(userpreference.UserPreference)
	err = db.NewSelect().Model(up).Where("user_id = ?", userID).Scan(ctx)
	if err != nil {
		log.Error("failed to scan user preference", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "User Preference")
	}

	return up, nil
}

func (r *repository) create(
	ctx context.Context,
	up *userpreference.UserPreference,
) (*userpreference.UserPreference, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("userID", up.UserID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	_, err = db.NewInsert().Model(up).Returning("*").Exec(ctx)
	if err != nil {
		log.Error("failed to create user preference", zap.Error(err))
		return nil, err
	}

	return up, nil
}

func (r *repository) Update(
	ctx context.Context,
	up *userpreference.UserPreference,
) (*userpreference.UserPreference, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("userID", up.UserID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	ov := up.Version
	up.Version++

	results, err := db.NewUpdate().Model(up).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update user preference", zap.Error(err))
		return nil, err
	}

	roErr := dberror.CheckRowsAffected(results, "User Preference", up.UserID.String())
	if roErr != nil {
		return nil, roErr
	}

	return up, nil
}

func (r *repository) Upsert(
	ctx context.Context,
	up *userpreference.UserPreference,
) (*userpreference.UserPreference, error) {
	log := r.l.With(
		zap.String("operation", "Upsert"),
		zap.String("userID", up.UserID.String()),
	)

	existing, err := r.GetByUserID(ctx, up.UserID)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return r.create(ctx, up)
		}
		log.Error("failed to get existing user preference", zap.Error(err))
		return nil, err
	}

	up.ID = existing.ID
	up.Version = existing.Version
	return r.Update(ctx, up)
}

func (r *repository) Delete(
	ctx context.Context,
	userID pulid.ID,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("userID", userID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	up := &userpreference.UserPreference{}
	results, err := db.NewDelete().Model(up).Where("user_id = ?", userID).Exec(ctx)
	if err != nil {
		log.Error("failed to delete user preference", zap.Error(err))
		return err
	}

	roErr := dberror.CheckRowsAffected(results, "User Preference", userID.String())
	if roErr != nil {
		return roErr
	}

	return nil
}
