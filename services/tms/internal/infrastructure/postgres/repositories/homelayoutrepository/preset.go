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

func (r *repository) CreatePreset(
	ctx context.Context,
	entity *homelayout.Preset,
) (*homelayout.Preset, error) {
	log := r.l.With(zap.String("operation", "CreatePreset"), zap.String("name", entity.Name))

	if _, err := r.db.DB().NewInsert().Model(entity).Exec(ctx); err != nil {
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

func (r *repository) UpdatePreset(
	ctx context.Context,
	entity *homelayout.Preset,
) (*homelayout.Preset, error) {
	log := r.l.With(zap.String("operation", "UpdatePreset"), zap.String("id", entity.ID.String()))

	ov := entity.Version
	entity.Version++

	result, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.PresetColumns.Version.Eq(), ov).
		Set(buncolgen.PresetColumns.Description.Set()).
		Set(buncolgen.PresetColumns.RoleIDs.Set()).
		Set(buncolgen.PresetColumns.CoreResponsibility.Set()).
		Set(buncolgen.PresetColumns.IsOrgDefault.Set()).
		Set(buncolgen.PresetColumns.Locked.Set()).
		Set(buncolgen.PresetColumns.Priority.Set()).
		OmitZero().
		Returning("*").
		Exec(ctx)
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

	if err = dberror.CheckRowsAffected(result, "HomeLayoutPreset", entity.ID.String()); err != nil {
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

	return dberror.CheckRowsAffected(result, "HomeLayoutPreset", req.PresetID.String())
}

// ClearOrgDefault demotes every other preset in the tenant. The partial unique
// index is the real guard; this exists so an administrator promoting a second
// preset gets a silent handover instead of a constraint violation.
func (r *repository) ClearOrgDefault(
	ctx context.Context,
	req *repositories.ClearHomeLayoutOrgDefaultRequest,
) error {
	log := r.l.With(zap.String("operation", "ClearOrgDefault"))

	_, err := r.db.DB().
		NewUpdate().
		Model((*homelayout.Preset)(nil)).
		Set(buncolgen.PresetColumns.IsOrgDefault.Bare()+" = FALSE").
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			uq = buncolgen.PresetScopeTenantUpdate(uq, req.TenantInfo).
				Where(buncolgen.PresetColumns.IsOrgDefault.IsTrue())
			if req.ExceptID.IsNil() {
				return uq
			}
			return uq.Where(buncolgen.PresetColumns.ID.Ne(), req.ExceptID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to clear home layout org default", zap.Error(err))
		return err
	}

	return nil
}

// CountUsersForRoles reports how many people a preset would reach, so the admin
// list can say what an assignment costs before it is made. Expired assignments
// are excluded because they no longer grant the role.
func (r *repository) CountUsersForRoles(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	roleIDs []pulid.ID,
) (int, error) {
	if len(roleIDs) == 0 {
		return 0, nil
	}

	log := r.l.With(zap.String("operation", "CountUsersForRoles"))

	now := timeutils.NowUnix()
	count := 0
	err := r.db.DB().
		NewSelect().
		Model((*permission.UserRoleAssignment)(nil)).
		ColumnExpr(buncolgen.CountDistinct(buncolgen.UserRoleAssignmentColumns.UserID, "count")).
		Where(buncolgen.UserRoleAssignmentColumns.OrganizationID.Eq(), tenantInfo.OrgID).
		Where(buncolgen.UserRoleAssignmentColumns.RoleID.In(), bun.List(roleIDs)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where(buncolgen.UserRoleAssignmentColumns.ExpiresAt.IsNull()).
				WhereOr(buncolgen.UserRoleAssignmentColumns.ExpiresAt.Gt(), now)
		}).
		Scan(ctx, &count)
	if err != nil {
		log.Error("failed to count users for roles", zap.Error(err))
		return 0, err
	}

	return count, nil
}
