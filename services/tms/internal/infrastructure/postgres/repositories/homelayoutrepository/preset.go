package homelayoutrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/homelayout"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// ListPresets returns every preset in the tenant ordered so that resolution can
// take the first match: highest priority first, then newest, which makes the
// winner deterministic when two presets share a priority.
func (r *repository) ListPresets(
	ctx context.Context,
	req *repositories.ListHomeLayoutPresetsRequest,
) ([]*homelayout.Preset, error) {
	log := r.l.With(zap.String("operation", "ListPresets"))

	presets := make([]*homelayout.Preset, 0)
	err := r.db.DB().
		NewSelect().
		Model(&presets).
		Apply(buncolgen.PresetApplyTenant(req.TenantInfo)).
		Order(
			buncolgen.PresetColumns.Priority.OrderDesc(),
			buncolgen.PresetColumns.CreatedAt.OrderDesc(),
		).
		Scan(ctx)
	if err != nil {
		log.Error("failed to list home layout presets", zap.Error(err))
		return nil, err
	}

	return presets, nil
}

func (r *repository) GetPreset(
	ctx context.Context,
	req *repositories.GetHomeLayoutPresetRequest,
) (*homelayout.Preset, error) {
	log := r.l.With(
		zap.String("operation", "GetPreset"),
		zap.String("presetID", req.PresetID.String()),
	)

	entity := new(homelayout.Preset)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.PresetScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.PresetColumns.ID.Eq(), req.PresetID)
		}).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, errortypes.NewNotFoundError("Home screen preset not found")
		}
		log.Error("failed to get home layout preset", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

// CreatePreset inserts the preset, first demoting the incumbent org default in
// the same transaction when this one takes the flag. Without the transaction a
// failed insert — a duplicate name, say — would leave the tenant with no
// default at all.
func (r *repository) CreatePreset(
	ctx context.Context,
	entity *homelayout.Preset,
) (*homelayout.Preset, error) {
	log := r.l.With(zap.String("operation", "CreatePreset"), zap.String("name", entity.Name))

	err := r.db.DB().RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if entity.IsOrgDefault {
			if err := clearOrgDefault(
				c,
				tx,
				entity.OrganizationID,
				entity.BusinessUnitID,
				pulid.Nil,
			); err != nil {
				log.Error("failed to clear home layout org default", zap.Error(err))
				return err
			}
		}

		_, err := tx.NewInsert().Model(entity).Exec(c)
		return err
	})
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, errortypes.NewValidationError(
				"name",
				errortypes.ErrDuplicate,
				"A home screen preset with this name already exists",
			)
		}
		log.Error("failed to create home layout preset", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

// UpdatePreset writes every column from the row the service just loaded, inside
// one transaction with the org-default handover when this preset is promoted.
// Neither OmitZero nor an explicit Set list is usable here: OmitZero would
// silently skip is_org_default, locked, and priority whenever they are being
// set back to false or zero — exactly the demotions an administrator means —
// and bun ignores the model's fields entirely once any Set() is present, which
// would drop name and layout.
func (r *repository) UpdatePreset(
	ctx context.Context,
	entity *homelayout.Preset,
) (*homelayout.Preset, error) {
	log := r.l.With(zap.String("operation", "UpdatePreset"), zap.String("id", entity.ID.String()))

	ov := entity.Version
	entity.Version++

	err := r.db.DB().RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if entity.IsOrgDefault {
			if err := clearOrgDefault(
				c,
				tx,
				entity.OrganizationID,
				entity.BusinessUnitID,
				entity.ID,
			); err != nil {
				log.Error("failed to clear home layout org default", zap.Error(err))
				return err
			}
		}

		result, err := tx.NewUpdate().
			Model(entity).
			WherePK().
			Where(buncolgen.PresetColumns.Version.Eq(), ov).
			Returning("*").
			Exec(c)
		if err != nil {
			return err
		}

		return dberror.CheckRowsAffected(result, "HomeLayoutPreset", entity.ID.String())
	})
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, errortypes.NewValidationError(
				"name",
				errortypes.ErrDuplicate,
				"A home screen preset with this name already exists",
			)
		}
		log.Error("failed to update home layout preset", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) DeletePreset(
	ctx context.Context,
	req *repositories.DeleteHomeLayoutPresetRequest,
) error {
	log := r.l.With(
		zap.String("operation", "DeletePreset"),
		zap.String("presetID", req.PresetID.String()),
	)

	result, err := r.db.DB().
		NewDelete().
		Model((*homelayout.Preset)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.PresetScopeTenantDelete(dq, req.TenantInfo).
				Where(buncolgen.PresetColumns.ID.Eq(), req.PresetID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete home layout preset", zap.Error(err))
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errortypes.NewNotFoundError("Home screen preset not found")
	}

	return nil
}

// clearOrgDefault demotes every other org-default preset in the tenant. It
// bumps version and updated_at on the demoted rows so any editor holding one of
// them open fails its next save on the optimistic lock instead of silently
// re-promoting a preset the administrator just demoted.
func clearOrgDefault(
	ctx context.Context,
	idb bun.IDB,
	orgID, buID, exceptID pulid.ID,
) error {
	_, err := idb.NewUpdate().
		Model((*homelayout.Preset)(nil)).
		Set(buncolgen.PresetColumns.IsOrgDefault.Bare()+" = FALSE").
		Set(buncolgen.PresetColumns.Version.Bare()+" = "+buncolgen.PresetColumns.Version.Bare()+" + 1").
		Set(buncolgen.PresetColumns.UpdatedAt.Bare()+" = ?", timeutils.NowUnix()).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			uq = uq.Where(buncolgen.PresetColumns.OrganizationID.Eq(), orgID).
				Where(buncolgen.PresetColumns.BusinessUnitID.Eq(), buID).
				Where(buncolgen.PresetColumns.IsOrgDefault.IsTrue())
			if exceptID.IsNil() {
				return uq
			}
			return uq.Where(buncolgen.PresetColumns.ID.Ne(), exceptID)
		}).
		Exec(ctx)

	return err
}

// ListRoleUserAssignments returns every live role membership in the tenant with
// the role's core responsibility attached, so the service can compute how many
// distinct people any preset reaches — by explicit role or by responsibility —
// in one query instead of one per preset. Expired assignments are excluded
// because they no longer grant the role.
func (r *repository) ListRoleUserAssignments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]repositories.HomeRoleUserAssignment, error) {
	log := r.l.With(zap.String("operation", "ListRoleUserAssignments"))

	now := timeutils.NowUnix()
	assignments := make([]repositories.HomeRoleUserAssignment, 0)
	err := r.db.DB().
		NewSelect().
		Model((*permission.UserRoleAssignment)(nil)).
		ColumnExpr(buncolgen.UserRoleAssignmentColumns.RoleID.Qualified()+" AS role_id").
		ColumnExpr(buncolgen.UserRoleAssignmentColumns.UserID.Qualified()+" AS user_id").
		ColumnExpr(
			buncolgen.RoleColumns.CoreResponsibility.Qualified()+" AS core_responsibility",
		).
		Join("JOIN roles AS r ON "+buncolgen.RoleColumns.ID.Qualified()+" = "+
			buncolgen.UserRoleAssignmentColumns.RoleID.Qualified()).
		Where(buncolgen.UserRoleAssignmentColumns.OrganizationID.Eq(), tenantInfo.OrgID).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where(buncolgen.UserRoleAssignmentColumns.ExpiresAt.IsNull()).
				WhereOr(buncolgen.UserRoleAssignmentColumns.ExpiresAt.Gt(), now)
		}).
		Scan(ctx, &assignments)
	if err != nil {
		log.Error("failed to list role user assignments", zap.Error(err))
		return nil, err
	}

	return assignments, nil
}
