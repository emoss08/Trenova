package userrepository

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
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

func New(p Params) repositories.UserRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.user-repository"),
	}
}

func (ur *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListUsersRequest,
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
		return sq.Where("usr.current_organization_id = ?", req.Filter.TenantInfo.OrgID).
			Where("usr.username != ?", "system").
			Where("usr.business_unit_id = ?", req.Filter.TenantInfo.BuID)
	})

	q.Relation("Assignments").Relation("Assignments.Role")

	if req.IncludeMemberships {
		q = q.Relation("Memberships").Relation("Memberships.Organization")
	}

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (ur *repository) List(
	ctx context.Context,
	req *repositories.ListUsersRequest,
) (*pagination.ListResult[*tenant.User], error) {
	log := ur.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*tenant.User, 0, req.Filter.Pagination.SafeLimit())

	q := ur.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return ur.filterQuery(sq, req)
		})

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count users", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*tenant.User]{
		Items: entities,
		Total: total,
	}, nil
}

func (ur *repository) GetByID(
	ctx context.Context,
	req repositories.GetUserByIDRequest,
) (*tenant.User, error) {
	entity := new(tenant.User)
	q := ur.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("usr.current_organization_id = ?", req.TenantInfo.OrgID).
				Where("usr.business_unit_id = ?", req.TenantInfo.BuID).
				Where("usr.id = ?", req.TenantInfo.UserID)
		})

	if req.IncludeMemberships {
		q = q.Relation("Memberships").
			Relation("Memberships.Organization").
			Relation("Memberships.Organization.State")
	}

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "User")
	}

	return entity, nil
}

func (ur *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*tenant.User], error) {
	return dbhelper.SelectOptions[*tenant.User](ctx, ur.db.DB(), req, &dbhelper.SelectOptionsConfig{
		Columns: []string{
			"id",
			"name",
			"username",
			"email_address",
			"status",
			"profile_pic_url",
			"thumbnail_url",
		},
		OrgColumn: "usr.current_organization_id",
		BuColumn:  "usr.business_unit_id",
		QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("usr.status = ?", domaintypes.StatusActive).
				Where("usr.username != ?", "system")
		},
		EntityName: "User",
		SearchColumns: []string{
			"usr.username",
			"usr.name",
			"usr.email_address",
		},
	})
}

func (ur *repository) FindByEmail(ctx context.Context, emailAddress string) (*tenant.User, error) {
	user := new(tenant.User)

	if err := ur.db.DB().
		NewSelect().
		Model(user).
		Where("usr.email_address = ?", emailAddress).
		Scan(ctx); err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, errortypes.NewValidationError(
				"emailAddress",
				errortypes.ErrNotFound,
				"User not found with the given email address",
			)
		}

		return nil, err
	}

	return user, nil
}

func (ur *repository) UpdateLastLoginAt(ctx context.Context, userID pulid.ID) error {
	log := ur.l.With(
		zap.String("operation", "UpdateLastLoginAt"),
		zap.String("userID", userID.String()),
	)

	if _, err := ur.db.DB().
		NewUpdate().
		Model((*tenant.User)(nil)).
		Set("last_login_at = ?", timeutils.NowUnix()).
		Where("usr.id = ?", userID).
		Exec(ctx); err != nil {
		log.Error("failed to update last login at in database", zap.Error(err))
		return err
	}

	return nil
}

func (ur *repository) GetOrganizations(
	ctx context.Context,
	userID pulid.ID,
) ([]*tenant.OrganizationMembership, error) {
	log := ur.l.With(
		zap.String("operation", "GetOrganizations"),
		zap.String("userID", userID.String()),
	)

	entities := make([]*tenant.OrganizationMembership, 0)
	err := ur.db.DB().
		NewSelect().
		Model(&entities).
		Relation("Organization").
		Relation("Organization.State").
		Relation("User").
		Where("uom.user_id = ?", userID).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("uom.expires_at IS NULL").
				WhereOr("uom.expires_at > ?", timeutils.NowUnix())
		}).
		Order("uom.is_default DESC").
		Scan(ctx)
	if err != nil {
		log.Error("failed to get user organizations", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (ur *repository) GetOrganizationsByBusinessUnit(
	ctx context.Context,
	businessUnitID pulid.ID,
) ([]*tenant.Organization, error) {
	orgs := make([]*tenant.Organization, 0)
	err := ur.db.DB().
		NewSelect().
		Model(&orgs).
		Relation("State").
		Where("org.business_unit_id = ?", businessUnitID).
		Order("org.name ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return orgs, nil
}

func (ur *repository) GetOrganizationByID(
	ctx context.Context,
	organizationID pulid.ID,
) (*tenant.Organization, error) {
	org := new(tenant.Organization)
	if err := ur.db.DB().
		NewSelect().
		Model(org).
		Where("org.id = ?", organizationID).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Organization")
	}

	return org, nil
}

func (ur *repository) ListOrganizationMemberships(
	ctx context.Context,
	userID, businessUnitID pulid.ID,
) ([]*tenant.OrganizationMembership, error) {
	memberships := make([]*tenant.OrganizationMembership, 0)
	err := ur.db.DBForContext(ctx).
		NewSelect().
		Model(&memberships).
		Relation("Organization").
		Relation("Organization.State").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("uom.user_id = ?", userID).
				Where("uom.business_unit_id = ?", businessUnitID)
		}).
		Order("uom.is_default DESC, uom.joined_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return memberships, nil
}

func (ur *repository) ReplaceOrganizationMemberships(
	ctx context.Context,
	req repositories.ReplaceOrganizationMembershipsRequest,
) ([]*tenant.OrganizationMembership, error) {
	if len(req.OrganizationIDs) == 0 {
		_, err := ur.db.DBForContext(ctx).
			NewDelete().
			Model((*tenant.OrganizationMembership)(nil)).
			Where("user_id = ?", req.UserID).
			Where("business_unit_id = ?", req.BusinessUnitID).
			Exec(ctx)
		if err != nil {
			return nil, err
		}

		return []*tenant.OrganizationMembership{}, nil
	}

	desiredOrgSet := make(map[pulid.ID]struct{}, len(req.OrganizationIDs))
	for _, orgID := range req.OrganizationIDs {
		desiredOrgSet[orgID] = struct{}{}
	}

	orgCount, err := ur.db.DBForContext(ctx).
		NewSelect().
		Model((*tenant.Organization)(nil)).
		Where("org.id IN (?)", bun.List(req.OrganizationIDs)).
		Where("org.business_unit_id = ?", req.BusinessUnitID).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	if orgCount != len(req.OrganizationIDs) {
		return nil, errortypes.NewAuthorizationError(
			"organizations must belong to the current business unit",
		)
	}

	currentMemberships, err := ur.ListOrganizationMemberships(ctx, req.UserID, req.BusinessUnitID)
	if err != nil {
		return nil, err
	}

	currentByOrg := make(map[pulid.ID]*tenant.OrganizationMembership, len(currentMemberships))
	var currentDefaultOrgID pulid.ID
	for _, membership := range currentMemberships {
		currentByOrg[membership.OrganizationID] = membership
		if membership.IsDefault {
			currentDefaultOrgID = membership.OrganizationID
		}
	}

	err = ur.db.WithTx(ctx, ports.TxOptions{
		LockTimeout: 250 * time.Millisecond,
	}, func(txCtx context.Context, tx bun.Tx) error {
		lockedMemberships := make([]*tenant.OrganizationMembership, 0)
		if lmErr := tx.NewSelect().
			Model(&lockedMemberships).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Where("uom.user_id = ?", req.UserID).
					Where("uom.business_unit_id = ?", req.BusinessUnitID)
			}).
			Order("uom.organization_id ASC").
			For("UPDATE NOWAIT").
			Scan(txCtx); lmErr != nil {
			return lmErr
		}

		currentByOrg = make(map[pulid.ID]*tenant.OrganizationMembership, len(lockedMemberships))
		currentDefaultOrgID = ""
		for _, membership := range lockedMemberships {
			currentByOrg[membership.OrganizationID] = membership
			if membership.IsDefault {
				currentDefaultOrgID = membership.OrganizationID
			}
		}

		for orgID, membership := range currentByOrg {
			if _, ok := desiredOrgSet[orgID]; ok {
				continue
			}
			if _, deleteErr := tx.NewDelete().Model((*tenant.OrganizationMembership)(nil)).
				Where("id = ?", membership.ID).
				Exec(txCtx); deleteErr != nil {
				return deleteErr
			}
		}

		for _, orgID := range req.OrganizationIDs {
			if _, ok := currentByOrg[orgID]; ok {
				continue
			}
			newMembership := &tenant.OrganizationMembership{
				UserID:         req.UserID,
				BusinessUnitID: req.BusinessUnitID,
				OrganizationID: orgID,
				GrantedByID:    req.ActorID,
				IsDefault:      false,
			}
			if _, insertErr := tx.NewInsert().Model(newMembership).Exec(txCtx); insertErr != nil {
				return insertErr
			}
		}

		newDefaultOrgID := currentDefaultOrgID
		if _, ok := desiredOrgSet[newDefaultOrgID]; !ok {
			newDefaultOrgID = req.OrganizationIDs[0]
		}

		if _, updateDefaultErr := tx.NewUpdate().
			Model((*tenant.OrganizationMembership)(nil)).
			Set("is_default = (organization_id = ?)", newDefaultOrgID).
			Where("user_id = ?", req.UserID).
			Where("business_unit_id = ?", req.BusinessUnitID).
			Exec(txCtx); updateDefaultErr != nil {
			return updateDefaultErr
		}

		return nil
	})
	if err != nil {
		return nil, dberror.MapRetryableTransactionError(
			err,
			"The user's organization memberships are busy. Retry the request.",
		)
	}

	return ur.ListOrganizationMemberships(ctx, req.UserID, req.BusinessUnitID)
}

func (ur *repository) UpdateCurrentOrganization(
	ctx context.Context,
	userID, orgID, buID pulid.ID,
) error {
	log := ur.l.With(
		zap.String("operation", "UpdateCurrentOrganization"),
		zap.String("userID", userID.String()),
		zap.String("orgID", orgID.String()),
	)

	if _, err := ur.db.DB().
		NewUpdate().
		Model((*tenant.User)(nil)).
		Set("current_organization_id = ?", orgID).
		Set("business_unit_id = ?", buID).
		Set("updated_at = ?", timeutils.NowUnix()).
		Where("usr.id = ?", userID).
		Exec(ctx); err != nil {
		log.Error("failed to update current organization", zap.Error(err))
		return err
	}

	return nil
}

func (ur *repository) IsPlatformAdmin(ctx context.Context, userID pulid.ID) (bool, error) {
	log := ur.l.With(
		zap.String("operation", "IsPlatformAdmin"),
		zap.String("userID", userID.String()),
	)

	var isPlatformAdmin bool
	err := ur.db.DB().
		NewSelect().
		Model((*tenant.User)(nil)).
		Column("is_platform_admin").
		Where("id = ?", userID).
		Scan(ctx, &isPlatformAdmin)
	if err != nil {
		log.Error("failed to check if user is platform admin", zap.Error(err))
		return false, err
	}

	return isPlatformAdmin, nil
}

func (ur *repository) GetUserOrganizationSummaries(
	ctx context.Context,
	userID pulid.ID,
) ([]repositories.OrgSummary, error) {
	log := ur.l.With(
		zap.String("operation", "GetUserOrganizationSummaries"),
		zap.String("userID", userID.String()),
	)

	var summaries []repositories.OrgSummary
	err := ur.db.DB().
		NewSelect().
		Model((*tenant.OrganizationMembership)(nil)).
		ColumnExpr("org.id, org.name").
		Join("JOIN organizations AS org ON org.id = uom.organization_id").
		Where("uom.user_id = ?", userID).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("uom.expires_at IS NULL").
				WhereOr("uom.expires_at > ?", timeutils.NowUnix())
		}).
		Order("org.name ASC").
		Scan(ctx, &summaries)
	if err != nil {
		log.Error("failed to get user organization summaries", zap.Error(err))
		return nil, err
	}

	return summaries, nil
}

func (ur *repository) Update(ctx context.Context, entity *tenant.User) (*tenant.User, error) {
	log := ur.l.With(
		zap.String("operation", "Update"),
		zap.String("userID", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	_, err := ur.db.DB().
		NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.Where("usr.id = ?", entity.ID).
				Where("usr.version = ?", ov).
				Where("usr.current_organization_id = ?", entity.CurrentOrganizationID).
				Where("usr.business_unit_id = ?", entity.BusinessUnitID)
		}).
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update user", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (ur *repository) UpdatePassword(
	ctx context.Context,
	req repositories.UpdateUserPasswordRequest,
) error {
	log := ur.l.With(
		zap.String("operation", "UpdatePassword"),
		zap.String("userID", req.UserID.String()),
	)

	_, err := ur.db.DB().
		NewUpdate().
		Model((*tenant.User)(nil)).
		Set("password = ?", req.Password).
		Set("must_change_password = ?", req.MustChangePassword).
		Set("updated_at = ?", timeutils.NowUnix()).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.Where("usr.id = ?", req.UserID).
				Where("usr.current_organization_id = ?", req.OrganizationID).
				Where("usr.business_unit_id = ?", req.BusinessUnitID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to update user password", zap.Error(err))
		return err
	}

	return nil
}

func (ur *repository) GetByIDs(
	ctx context.Context,
	req repositories.GetUsersByIDsRequest,
) ([]*tenant.User, error) {
	log := ur.l.With(
		zap.String("operation", "GetByIDs"),
		zap.Any("request", req),
	)

	entities := make([]*tenant.User, 0, len(req.UserIDs))
	err := ur.db.DB().
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("usr.current_organization_id = ?", req.TenantInfo.OrgID).
				Where("usr.business_unit_id = ?", req.TenantInfo.BuID).
				Where("usr.id IN (?)", bun.List(req.UserIDs))
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get users", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "User")
	}

	return entities, nil
}

func (ur *repository) BulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateUserStatusRequest,
) ([]*tenant.User, error) {
	log := ur.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	entities := make([]*tenant.User, 0, len(req.UserIDs))
	results, err := ur.db.DB().
		NewUpdate().
		Model(&entities).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.Where("usr.current_organization_id = ?", req.TenantInfo.OrgID).
				Where("usr.business_unit_id = ?", req.TenantInfo.BuID).
				Where("usr.id IN (?)", bun.List(req.UserIDs))
		}).
		Set("status = ?", req.Status).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to bulk update user status", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckBulkRowsAffected(results, "User", req.UserIDs); err != nil {
		return nil, err
	}

	return entities, nil
}

func (ur *repository) GetSystemUser(ctx context.Context, columns ...string) (*tenant.User, error) {
	user := new(tenant.User)

	if err := ur.db.DB().NewSelect().
		Column(columns...).
		Model(user).
		Where("usr.username = ?", "system").
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "User")
	}

	return user, nil
}
