package repositories

import (
	"context"
	"database/sql"

	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/trenova-app/transport/internal/core/domain/user"
	"github.com/trenova-app/transport/internal/core/ports"
	"github.com/trenova-app/transport/internal/core/ports/db"
	"github.com/trenova-app/transport/internal/core/ports/repositories"
	"github.com/trenova-app/transport/internal/pkg/errors"
	"github.com/trenova-app/transport/internal/pkg/logger"
	"github.com/trenova-app/transport/pkg/types/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type UserRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type userRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewUserRepository(p UserRepositoryParams) repositories.UserRepository {
	log := p.Logger.With().
		Str("repository", "user").
		Logger()

	return &userRepository{
		db: p.DB,
		l:  &log,
	}
}

func (ur *userRepository) filterQuery(q *bun.SelectQuery, opts *ports.LimitOffsetQueryOptions) *bun.SelectQuery {
	q = q.Where("usr.business_unit_id = ?", opts.TenantOpts.BuID).
		Where("usr.current_organization_id = ?", opts.TenantOpts.OrgID).
		Limit(opts.Limit).
		Offset(opts.Offset)

	if opts.Query != "" {
		q = q.Where("usr.name ILIKE ? OR usr.username ILIKE ?", "%"+opts.Query+"%", "%"+opts.Query+"%")
	}

	return q
}

func (ur *userRepository) List(ctx context.Context, opts *ports.LimitOffsetQueryOptions) (*ports.ListResult[*user.User], error) {
	dba, err := ur.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := ur.l.With().Str("operation", "List").
		Str("buID", opts.TenantOpts.BuID.String()).
		Str("userID", opts.TenantOpts.UserID.String()).
		Logger()

	users := make([]*user.User, 0)

	q := dba.NewSelect().Model(&users)
	q = ur.filterQuery(q, opts)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan users")
		return nil, eris.Wrap(err, "scan users")
	}

	return &ports.ListResult[*user.User]{
		Items: users,
		Total: total,
	}, nil
}

func (ur *userRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	dba, err := ur.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	u := new(user.User)

	q := dba.NewSelect().Model(u).Where("usr.email_address = ?", email)

	if err = q.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewValidationError("emailAddress", errors.ErrNotFound, "User with this email address not found")
		}

		ur.l.Error().Err(err).Msgf("failed to find user by email %s", email)
		return nil, eris.Wrapf(err, "failed to find user by email %s", email)
	}

	return u, nil
}

func (ur *userRepository) UpdateLastLogin(ctx context.Context, userID pulid.ID) error {
	dba, err := ur.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	return dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		u := new(user.User)

		results, rErr := tx.NewUpdate().Model(u).Where("usr.id = ?", userID).Exec(c)
		if rErr != nil {
			ur.l.Error().Err(rErr).Msgf("failed to update last login for user %s", userID)
			return eris.Wrapf(rErr, "failed to update last login for user %s", userID)
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			ur.l.Error().Err(roErr).Msgf("failed to get rows affected for user %s", userID)
			return eris.Wrapf(roErr, "failed to get rows affected for user %s", userID)
		}

		if rows == 0 {
			return errors.NewValidationError("id", errors.ErrNotFound, "User not found")
		}

		ur.l.Info().Msgf("updated last login for user %s", userID)
		return nil
	})
}

func (ur *userRepository) GetByID(ctx context.Context, opts *repositories.GetUserByIDOptions) (*user.User, error) {
	dba, err := ur.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	u := new(user.User)

	q := dba.NewSelect().Model(u).Where("usr.id = ?", opts.UserID)

	// Include roles and permissions if needed
	if opts.IncludeRoles {
		q.Relation("Roles").Relation("Roles.Permissions")
	}

	// Include organizations if needed
	if opts.IncludeOrgs {
		q.Relation("Organizations")
	}

	if err = q.Scan(ctx); err != nil {
		return nil, eris.Wrapf(err, "failed to get user by id %s", opts.UserID)
	}

	return u, nil
}
