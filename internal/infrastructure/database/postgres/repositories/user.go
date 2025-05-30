package repositories

import (
	"context"
	"database/sql"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// UserRepositoryParams defines dependencies required for initializing the UserRepository.
// This includes database connection, permission repository, and logger.
type UserRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
	PRepo  repositories.PermissionRepository
}

// userRepository implements the UserRepository interface
// and provides methods to manage user data, including CRUD operations.
type userRepository struct {
	db    db.Connection
	l     *zerolog.Logger
	pRepo repositories.PermissionRepository
}

// NewUserRepository initializes a new instance of userRepository with its dependencies.
//
// Parameters:
//   - p: UserRepositoryParams containing dependencies.
//
// Returns:
//   - repositories.UserRepository: A ready-to-use user repository instance.
func NewUserRepository(p UserRepositoryParams) repositories.UserRepository {
	log := p.Logger.With().
		Str("repository", "user").
		Logger()

	return &userRepository{
		db:    p.DB,
		l:     &log,
		pRepo: p.PRepo,
	}
}

// filterQuery constructs a SQL query to filter users based on the provided options.
// It applies business unit and organization filters, and optionally filters by name or username.
//
// Parameters:
//   - q: The base select query.
//   - req: ListUserRequest containing filter and pagination details.
//
// Returns:
//   - *bun.SelectQuery: The filtered and paginated query.
func (ur *userRepository) filterQuery(
	q *bun.SelectQuery,
	req repositories.ListUserRequest,
) *bun.SelectQuery {
	q = q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.
			Where("usr.business_unit_id = ?", req.Filter.TenantOpts.BuID).
			Where("usr.current_organization_id = ?", req.Filter.TenantOpts.OrgID)
	})

	if req.Filter.Query != "" {
		q = q.Where(
			"usr.name ILIKE ? OR usr.username ILIKE ?",
			"%"+req.Filter.Query+"%",
			"%"+req.Filter.Query+"%",
		)
	}

	if req.IncludeRoles {
		q = q.Relation("Roles")
		q = q.Relation("Roles.Permissions")
	}

	q = q.Order("usr.status ASC", "usr.last_login_at DESC NULLS LAST")

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

// List retrieves a paginated list of users based on the provided options.
// It applies business unit and organization filters, and optionally filters by name or username.
//
// Parameters:
//   - ctx: The context for the operation.
//   - req: ListUserRequest containing filter and pagination details.
//
// Returns:
//   - *ports.ListResult[*user.User]: A paginated list of users.
//   - error: An error if the operation fails.
func (ur *userRepository) List(
	ctx context.Context,
	req repositories.ListUserRequest,
) (*ports.ListResult[*user.User], error) {
	dba, err := ur.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := ur.l.With().Str("operation", "List").
		Str("buID", req.Filter.TenantOpts.BuID.String()).
		Str("userID", req.Filter.TenantOpts.UserID.String()).
		Logger()

	users := make([]*user.User, 0)

	q := dba.NewSelect().Model(&users)
	q = ur.filterQuery(q, req)

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

// FindByEmail searches for a user by their email address.
//
// Parameters:
//   - ctx: The context for the operation.
//   - email: The email address to search for.
//
// Returns:
//   - *user.User: The user found with the given email address.
//   - error: An error if the operation fails.
func (ur *userRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	dba, err := ur.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	u := new(user.User)

	q := dba.NewSelect().Model(u).Where("usr.email_address = ?", email)

	if err = q.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewValidationError(
				"emailAddress",
				errors.ErrNotFound,
				"User with this email address not found",
			)
		}

		ur.l.Error().Err(err).Msgf("failed to find user by email %s", email)
		return nil, eris.Wrapf(err, "failed to find user by email %s", email)
	}

	return u, nil
}

// GetByID retrieves a user by their ID.
//
// Parameters:
//   - ctx: The context for the operation.
//   - opts: GetUserByIDOptions containing user ID and optional flags for including organizations and roles.
//
// Returns:
//   - *user.User: The user found with the given ID.
//   - error: An error if the operation fails.
func (ur *userRepository) GetByID(
	ctx context.Context,
	opts repositories.GetUserByIDOptions,
) (*user.User, error) {
	dba, err := ur.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	u := new(user.User)

	q := dba.NewSelect().Model(u).Where("usr.id = ?", opts.UserID)

	// * Include organizations if needed
	if opts.IncludeOrgs {
		q.Relation("Organizations")
	}

	if err = q.Scan(ctx); err != nil {
		return nil, eris.Wrapf(err, "failed to get user by id %s", opts.UserID)
	}

	// * Include roles and permissions if needed
	if opts.IncludeRoles {
		if err = ur.loadUserRolesAndPermissions(ctx, u, opts.UserID); err != nil {
			return nil, err
		}
	}

	return u, nil
}

// loadUserRolesAndPermissions loads the roles and permissions for a user
//
// Parameters:
//   - ctx: The context for the operation.
//   - u: The user to load roles and permissions for.
//   - userID: The ID of the user to load roles and permissions for.
//
// Returns:
//   - error: An error if the operation fails.
func (ur *userRepository) loadUserRolesAndPermissions(
	ctx context.Context,
	u *user.User,
	userID pulid.ID,
) error {
	log := ur.l.With().
		Str("operation", "loadUserRolesAndPermissions").
		Str("userId", userID.String()).
		Logger()

	// * Get roles and permissions from the permission repository
	rolesAndPerms, err := ur.pRepo.GetRolesAndPermissions(ctx, userID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get user roles and permissions")
		return eris.Wrap(err, "get user roles and permissions")
	}

	// * Use complete roles when available (preferred because permissions are properly associated)
	if len(rolesAndPerms.CompleteRoles) > 0 {
		u.Roles = rolesAndPerms.CompleteRoles
		log.Debug().Int("count", len(u.Roles)).Msg("using complete roles with permissions")
		return nil
	}

	// * Fallback: Create roles from names and assign all permissions to each role
	u.Roles = make([]*permission.Role, 0, len(rolesAndPerms.Roles))

	for _, roleName := range rolesAndPerms.Roles {
		if roleName == nil {
			continue
		}

		u.Roles = append(u.Roles, &permission.Role{
			Name:        *roleName,
			Permissions: rolesAndPerms.Permissions,
		})
	}

	log.Debug().Int("roleCount", len(u.Roles)).
		Int("permissionCount", len(rolesAndPerms.Permissions)).
		Msg("created roles from names with shared permissions")

	return nil
}

// UpdateLastLogin updates the last login time for a user
//
// Parameters:
//   - ctx: The context for the operation.
//   - userID: The ID of the user to update the last login time for.
//
// Returns:
//   - error: An error if the operation fails.
func (ur *userRepository) UpdateLastLogin(ctx context.Context, userID pulid.ID) error {
	dba, err := ur.db.DB(ctx)
	if err != nil {
		return err
	}

	log := ur.l.With().
		Str("operation", "UpdateLastLogin").
		Str("userID", userID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		q := tx.NewUpdate().Model((*user.User)(nil)).
			Set("last_login_at = ?", timeutils.NowUnix()).
			Where("usr.id = ?", userID)

		if _, err = q.Exec(c); err != nil {
			log.Error().Str("userID", userID.String()).Err(err).Msg("failed to update last login")
			return err
		}

		return nil
	})

	return err
}

// Create creates a new user
//
// Parameters:
//   - ctx: The context for the operation.
//   - u: The user to create.
//
// Returns:
//   - *user.User: The created user.
//   - error: An error if the operation fails.
func (ur *userRepository) Create(ctx context.Context, u *user.User) (*user.User, error) {
	dba, err := ur.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := ur.l.With().Str("operation", "Create").
		Str("orgID", u.CurrentOrganizationID.String()).
		Str("buID", u.BusinessUnitID.String()).
		Str("userID", u.ID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, iErr := tx.NewInsert().Model(u).Exec(c); iErr != nil {
			return iErr
		}

		return nil
	})

	if err != nil {
		log.Error().
			Err(err).
			Interface("user", u).
			Msg("failed to create user")
		return nil, err
	}

	return u, nil
}

// Update updates a user
//
// Parameters:
//   - ctx: The context for the operation.
//   - u: The user to update.
//
// Returns:
//   - *user.User: The updated user.
//   - error: An error if the operation fails.
func (ur *userRepository) Update(ctx context.Context, u *user.User) (*user.User, error) {
	dba, err := ur.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := ur.l.With().Str("operation", "Update").
		Str("orgID", u.CurrentOrganizationID.String()).
		Str("buID", u.BusinessUnitID.String()).
		Str("userID", u.ID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := u.Version

		u.Version++

		results, rErr := tx.NewUpdate().
			Model(u).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return uq.Where("usr.id = ?", u.ID).
					Where("usr.version = ?", ov).
					Where("usr.current_organization_id = ?", u.CurrentOrganizationID).
					Where("usr.business_unit_id = ?", u.BusinessUnitID).
					Where("usr.version = ?", ov)
			}).
			Returning("*").
			Exec(c)
		if rErr != nil {
			return rErr
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			return roErr
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				"User has been modified since it was last read",
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Interface("user", u).Msg("failed to update user")
		return nil, err
	}

	return u, nil
}
