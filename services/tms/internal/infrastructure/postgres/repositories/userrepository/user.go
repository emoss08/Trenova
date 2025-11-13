package userrepository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/m2msync"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/utils/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB      *postgres.Connection
	Logger  *zap.Logger
	M2MSync *m2msync.Syncer
}

type repository struct {
	db      *postgres.Connection
	l       *zap.Logger
	m2mSync *m2msync.Syncer
}

func NewRepository(p Params) repositories.UserRepository {
	return &repository{
		db:      p.DB,
		l:       p.Logger.Named("user-repository"),
		m2mSync: p.M2MSync,
	}
}

func (ur *repository) GetOption(
	ctx context.Context,
	req repositories.GetUserByIDRequest,
) (*tenant.User, error) {
	db, err := ur.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	user := new(tenant.User)
	if err = db.NewSelect().Model(user).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("usr.current_organization_id = ?", req.OrgID).
				Where("usr.business_unit_id = ?", req.BuID).
				Where("usr.id = ?", req.UserID)
		}).
		Scan(ctx); err != nil {
		return nil, err
	}

	return user, nil
}

func (ur *repository) SelectOptions(
	ctx context.Context,
	req repositories.UserSelectOptionsRequest,
) ([]*repositories.UserSelectOptionResponse, error) {
	db, err := ur.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	users := make([]*repositories.UserSelectOptionResponse, 0)
	q := db.NewSelect().Model((*tenant.User)(nil)).
		Column("id", "name", "username", "email_address", "profile_pic_url").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("usr.current_organization_id = ?", req.OrgID).
				Where("usr.business_unit_id = ?", req.BuID)
		})

	if req.Query != "" {
		q = q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				WhereOr("usr.username ILIKE ?", "%"+req.Query+"%").
				WhereOr("usr.name ILIKE ?", "%"+req.Query+"%").
				WhereOr("usr.email_address ILIKE ?", "%"+req.Query+"%")
		})
	}

	if err = q.Scan(ctx, &users); err != nil {
		return nil, err
	}

	return users, nil
}

func (ur *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListUserRequest,
) *bun.SelectQuery {
	fieldConfig := querybuilder.NewFieldConfigBuilder((*tenant.User)(nil)).
		WithAutoMapping().
		WithAllFieldsFilterable().
		WithAllFieldsSortable().
		WithAutoEnumDetection().
		WithRelationshipFields().
		Build()

	qb := querybuilder.NewWithPostgresSearch(q, "usr", fieldConfig, (*tenant.User)(nil))
	qb.WithTraversalSupport(true)

	if len(req.Filter.FieldFilters) > 0 {
		qb.ApplyFilters(req.Filter.FieldFilters)
	}

	if req.Filter.Query != "" {
		searchFields := querybuilder.ExtractSearchFields(fieldConfig)
		qb.ApplyTextSearch(req.Filter.Query, searchFields)
	}

	if len(req.Filter.Sort) > 0 {
		qb.ApplySort(req.Filter.Sort)
	}

	q = qb.GetQuery()

	q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.Where("usr.username != ?", "system").
			Where("usr.current_organization_id = ?", req.Filter.TenantOpts.OrgID).
			Where("usr.business_unit_id = ?", req.Filter.TenantOpts.BuID)
	})

	if req.IncludeRoles {
		q = q.Relation("OrganizationMemberships")
	}

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (ur *repository) List(
	ctx context.Context,
	req *repositories.ListUserRequest,
) (*pagination.ListResult[*tenant.User], error) {
	log := ur.l.With(
		zap.String("operation", "List"),
		zap.String("orgId", req.Filter.TenantOpts.OrgID.String()),
		zap.String("buId", req.Filter.TenantOpts.BuID.String()),
	)

	db, err := ur.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*tenant.User, 0, req.Filter.Limit)

	total, err := db.NewSelect().
		Distinct().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return ur.filterQuery(sq, req)
		}).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan users", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*tenant.User]{
		Items: entities,
		Total: total,
	}, nil
}

func (ur *repository) FindByEmail(
	ctx context.Context,
	emailAddress string,
) (*tenant.User, error) {
	log := ur.l.With(zap.String("operation", "FindByEmail"))

	db, err := ur.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	user := new(tenant.User)
	if err = db.NewSelect().Model(user).Where("usr.email_address = ?", emailAddress).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errortypes.NewValidationError(
				"emailAddress",
				errortypes.ErrNotFound,
				"User not found with the given email address",
			)
		}

		log.Error("failed to find user by email", zap.Error(err))
		return nil, err
	}

	return user, nil
}

func (ur *repository) GetByID(
	ctx context.Context,
	req repositories.GetUserByIDRequest,
) (*tenant.User, error) {
	log := ur.l.With(zap.Any("request", req))

	db, err := ur.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	u := new(tenant.User)
	q := db.NewSelect().Model(u).Where("usr.id = ?", req.UserID)

	if req.IncludeOrgs {
		q.Relation("OrganizationMemberships")
	}

	if err = q.Scan(ctx); err != nil {
		return nil, err
	}

	return u, nil
}

func (ur *repository) GetNameByID(ctx context.Context, userID pulid.ID) (string, error) {
	db, err := ur.db.DB(ctx)
	if err != nil {
		return "", err
	}

	u := new(tenant.User)
	q := db.NewSelect().Model(u).Where("usr.id = ?", userID)

	if err = q.Scan(ctx); err != nil {
		return "", err
	}

	return u.Name, nil
}

func (ur *repository) GetByIDs(
	ctx context.Context,
	req repositories.GetUsersByIDsRequest,
) ([]*tenant.User, error) {
	if len(req.UserIDs) == 0 {
		return []*tenant.User{}, nil
	}

	db, err := ur.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	users := make([]*tenant.User, 0, len(req.UserIDs))
	q := db.NewSelect().Model(&users).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("usr.id IN (?)", bun.In(req.UserIDs)).
				Where("usr.current_organization_id = ?", req.OrgID).
				Where("usr.business_unit_id = ?", req.BuID)
		})

	if err = q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "User")
	}

	return users, nil
}

func (ur *repository) GetSystemUser(ctx context.Context) (*tenant.User, error) {
	db, err := ur.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	u := new(tenant.User)

	q := db.NewSelect().Model(u).Where("usr.email_address = ?", "system@trenova.app")
	if err = q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "User")
	}

	return u, nil
}

func (ur *repository) Create(ctx context.Context, u *tenant.User) (*tenant.User, error) {
	log := ur.l.With(zap.String("operation", "Create"))

	db, err := ur.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	err = db.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, err = db.NewInsert().Model(u).Exec(ctx); err != nil {
			log.Error("failed to create user", zap.Error(err))
			return err
		}

		if err = ur.syncUserRoles(c, tx, u); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		log.Error("failed to run in transaction", zap.Error(err))
		return nil, err
	}

	return u, nil
}

func (ur *repository) UpdateMe(
	ctx context.Context,
	req *repositories.UpdateMeRequest,
) (*tenant.User, error) {
	log := ur.l.With(zap.String("operation", "UpdateMe"), zap.Any("request", req))

	db, err := ur.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	_, err = db.NewUpdate().
		Model((*tenant.User)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.Where("usr.id = ?", req.UserID).
				Where("usr.current_organization_id = ?", req.OrgID).
				Where("usr.business_unit_id = ?", req.BuID)
		}).
		Set("name = ?", req.Name).
		Set("username = ?", req.Username).
		Set("email_address = ?", req.EmailAddress).
		Set("timezone = ?", req.Timezone).
		Set("time_format = ?", req.TimeFormat).
		Set("updated_at = ?", utils.NowUnix()).
		Exec(ctx)
	if err != nil {
		log.Error("failed to update user", zap.Error(err))
		return nil, err
	}

	u, err := ur.GetByID(ctx, repositories.GetUserByIDRequest{
		UserID:       req.UserID,
		OrgID:        req.OrgID,
		BuID:         req.BuID,
		IncludeRoles: true,
		IncludeOrgs:  true,
	})
	if err != nil {
		log.Error("failed to get user", zap.Error(err))
		return nil, err
	}

	return u, nil
}

func (ur *repository) Update(ctx context.Context, u *tenant.User) (*tenant.User, error) {
	log := ur.l.With(zap.String("operation", "Update"))

	db, err := ur.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	err = db.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := u.Version
		u.Version++

		results, rErr := tx.NewUpdate().
			Model(u).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return uq.Where("usr.id = ?", u.ID).
					Where("usr.version = ?", ov).
					Where("usr.current_organization_id = ?", u.CurrentOrganizationID).
					Where("usr.business_unit_id = ?", u.BusinessUnitID)
			}).
			OmitZero().
			Returning("*").
			Exec(c)

		if rErr != nil {
			return rErr
		}

		roErr := dberror.CheckRowsAffected(results, "User", u.ID.String())
		if roErr != nil {
			return roErr
		}

		if err = ur.syncUserRoles(c, tx, u); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		log.Error("failed to run in transaction", zap.Error(err))
		return nil, err
	}

	return u, nil
}

func (ur *repository) syncUserRoles(
	ctx context.Context,
	tx bun.IDB,
	u *tenant.User,
) error {
	log := ur.l.With(
		zap.String("operation", "syncUserRoles"),
		zap.String("userID", u.ID.String()),
		zap.Int("roleCount", len(u.OrganizationMemberships)),
	)

	config := m2msync.Config{
		Table:       "user_organization_memberships",
		SourceField: "user_id",
		TargetField: "role_id",
		AdditionalFields: map[string]any{
			"organization_id":  u.CurrentOrganizationID,
			"business_unit_id": u.BusinessUnitID,
		},
	}

	if err := ur.m2mSync.SyncEntities(ctx, tx, config, u.ID, u.OrganizationMemberships); err != nil {
		log.Error("failed to sync user roles", zap.Error(err))
		return err
	}

	log.Debug("successfully synced user roles")
	return nil
}

func (ur *repository) ChangePassword(
	ctx context.Context,
	req *repositories.ChangePasswordRequest,
) (*tenant.User, error) {
	log := ur.l.With(
		zap.String("operation", "ChangePassword"),
		zap.Any("request", req),
	)

	db, err := ur.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	q := db.NewUpdate().
		Model((*tenant.User)(nil)).
		Set("password = ?", req.HashedPassword).
		Set("must_change_password = ?", false).
		Where("usr.id = ?", req.UserID)

	if _, err = q.Exec(ctx); err != nil {
		log.Error("failed to change password", zap.Error(err))
		return nil, err
	}

	u, err := ur.GetByID(ctx, repositories.GetUserByIDRequest{
		UserID:       req.UserID,
		OrgID:        req.OrgID,
		BuID:         req.BuID,
		IncludeRoles: false,
		IncludeOrgs:  false,
	})
	if err != nil {
		log.Error("failed to get user", zap.Error(err))
		return nil, err
	}

	return u, nil
}

func (ur *repository) UpdateLastLogin(
	ctx context.Context,
	userID pulid.ID,
) error {
	log := ur.l.With(zap.String("operation", "UpdateLastLogin"))

	db, err := ur.db.DB(ctx)
	if err != nil {
		return err
	}

	q := db.NewUpdate().Model((*tenant.User)(nil)).
		Set("last_login_at = ?", utils.NowUnix()).
		Where("usr.id = ?", userID)

	if _, err = q.Exec(ctx); err != nil {
		log.Error("failed to update last login", zap.Error(err))
		return err
	}

	return nil
}

func (ur *repository) SwitchOrganization(
	ctx context.Context,
	userID, newOrgID pulid.ID,
) (updatedUser *tenant.User, err error) {
	dba, err := ur.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	log := ur.l.With(
		zap.String("operation", "SwitchOrganization"),
		zap.String("userID", userID.String()),
		zap.String("newOrgID", newOrgID.String()),
	)

	updatedUser = new(tenant.User)
	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		u := new(tenant.User)
		if err = tx.NewSelect().
			Model(u).
			Relation("OrganizationMemberships").
			Where("usr.id = ?", userID).
			Scan(c); err != nil {
			return dberror.HandleNotFoundError(err, "User")
		}

		hasAccess := false
		for _, membership := range u.OrganizationMemberships {
			if membership.OrganizationID == newOrgID {
				hasAccess = true
				break
			}
		}

		if !hasAccess {
			return errortypes.NewValidationError(
				"organizationID",
				errortypes.ErrNotFound,
				"You do not have access to this organization",
			)
		}

		result, uErr := tx.NewUpdate().
			Model((*tenant.User)(nil)).
			Set("current_organization_id = ?", newOrgID).
			Set("updated_at = ?", utils.NowUnix()).
			Where("usr.id = ?", userID).
			Exec(c)

		if uErr != nil {
			log.Error("failed to update user organization", zap.Error(uErr))
			return err
		}

		raErr := dberror.CheckRowsAffected(result, "User", userID.String())
		if raErr != nil {
			return err
		}

		return nil
	})
	if err != nil {
		log.Error("failed to switch organization", zap.Error(err))
		return nil, err
	}

	return updatedUser, nil
}
