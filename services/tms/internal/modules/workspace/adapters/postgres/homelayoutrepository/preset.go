package homelayoutrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission/permissiondb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homelayout"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homelayout/homelayoutdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/homelayoutports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const presetEntityName = "HomeLayoutPreset"

var presetErrors = dberror.Mapping{Entity: presetEntityName}

func duplicatePresetName() error {
	return errortypes.NewValidationError(
		"name",
		errortypes.ErrDuplicate,
		"A home screen preset with this name already exists",
	)
}

func presetNotFound() error {
	return errortypes.NewNotFoundError("Home screen preset not found")
}

// ListPresets returns every preset in the tenant ordered so that resolution can
// take the first match: highest priority first, then newest, which makes the
// winner deterministic when two presets share a priority.
func (r *repository) ListPresets(
	ctx context.Context,
	_ *homelayoutports.ListHomeLayoutPresetsRequest,
) ([]*homelayout.Preset, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*homelayout.Preset, error) {
		p := homelayoutdb.Presets
		presets, err := p.Order(p.Priority.Desc(), p.CreatedAt.Desc()).All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list home layout presets", zap.Error(err))
			return nil, err
		}

		return presets, nil
	})
}

func (r *repository) GetPreset(
	ctx context.Context,
	req *homelayoutports.GetHomeLayoutPresetRequest,
) (*homelayout.Preset, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*homelayout.Preset, error) {
		entity, err := homelayoutdb.Presets.Get(ctx, r.db.Keel(ctx), req.PresetID)
		if dberror.IsNotFoundError(err) {
			return nil, presetNotFound()
		}
		if err != nil {
			r.l.Error(
				"failed to get home layout preset",
				zap.String("presetID", req.PresetID.String()),
				zap.Error(err),
			)
			return nil, err
		}

		return entity, nil
	})
}

// CreatePreset inserts the preset, first demoting the incumbent org default in
// the same savepoint when this one takes the flag. Without it a failed insert —
// a duplicate name, say — would leave the tenant with no default at all.
func (r *repository) CreatePreset(
	ctx context.Context,
	entity *homelayout.Preset,
) (*homelayout.Preset, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*homelayout.Preset, error) {
		err := dbtx.Savepoint(ctx, r.db, func(ctx context.Context) error {
			if entity.IsOrgDefault {
				if err := r.clearOrgDefault(ctx, pulid.Nil); err != nil {
					return err
				}
			}

			return homelayoutdb.Presets.Create(ctx, r.db.Keel(ctx), entity)
		})
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, duplicatePresetName()
		}
		if err != nil {
			r.l.Error(
				"failed to create home layout preset",
				zap.String("name", entity.Name),
				zap.Error(err),
			)
			return nil, err
		}

		return entity, nil
	})
}

// UpdatePreset writes every column from the row the service just loaded, inside
// one savepoint with the org-default handover when this preset is promoted.
func (r *repository) UpdatePreset(
	ctx context.Context,
	entity *homelayout.Preset,
) (*homelayout.Preset, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*homelayout.Preset, error) {
		err := dbtx.Savepoint(ctx, r.db, func(ctx context.Context) error {
			if entity.IsOrgDefault {
				if err := r.clearOrgDefault(ctx, entity.ID); err != nil {
					return err
				}
			}

			return homelayoutdb.Presets.Save(ctx, r.db.Keel(ctx), entity)
		})
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, duplicatePresetName()
		}
		if err != nil {
			r.l.Error(
				"failed to update home layout preset",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, presetErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) DeletePreset(
	ctx context.Context,
	req *homelayoutports.DeleteHomeLayoutPresetRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		p := homelayoutdb.Presets
		rows, err := p.Where(p.ID.Eq(req.PresetID)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error(
				"failed to delete home layout preset",
				zap.String("presetID", req.PresetID.String()),
				zap.Error(err),
			)
			return err
		}
		if rows == 0 {
			return presetNotFound()
		}

		return nil
	})
}

// clearOrgDefault demotes every other org-default preset in the tenant. The
// bulk update bumps version and updated_at on the demoted rows so any editor
// holding one of them open fails its next save on the optimistic lock instead
// of silently re-promoting a preset the administrator just demoted.
func (r *repository) clearOrgDefault(ctx context.Context, exceptID pulid.ID) error {
	p := homelayoutdb.Presets
	_, err := p.Where(
		p.IsOrgDefault.IsTrue(),
		keel.When(exceptID.IsNotNil(), p.ID.Ne(exceptID)),
	).Update(ctx, r.db.Keel(ctx), p.IsOrgDefault.To(false))
	if err != nil {
		r.l.Error("failed to clear home layout org default", zap.Error(err))
	}

	return err
}

// ListRoleUserAssignments returns every live role membership in the tenant with
// the role's core responsibility attached, so the service can compute how many
// distinct people any preset reaches — by explicit role or by responsibility —
// in one query instead of one per preset. Expired assignments are excluded
// because they no longer grant the role.
func (r *repository) ListRoleUserAssignments(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]homelayoutports.HomeRoleUserAssignment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]homelayoutports.HomeRoleUserAssignment, error) {
			a := permissiondb.UserRoleAssignments
			role := a.Role()
			assignments, err := keel.Into[homelayoutports.HomeRoleUserAssignment](
				ctx,
				r.db.Keel(ctx),
				a.Pick(a.RoleID, a.UserID, role.CoreResponsibility.Coalesce("")).
					Where(
						role.ID.IsNotNull(),
						keel.Or(a.ExpiresAt.IsNull(), a.ExpiresAt.Gt(timeutils.NowUnix())),
					),
			)
			if err != nil {
				r.l.Error("failed to list role user assignments", zap.Error(err))
				return nil, err
			}
			if assignments == nil {
				assignments = []homelayoutports.HomeRoleUserAssignment{}
			}

			return assignments, nil
		},
	)
}
