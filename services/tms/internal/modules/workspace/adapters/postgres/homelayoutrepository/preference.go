package homelayoutrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homelayout"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homelayout/homelayoutdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/homelayoutports"
	"github.com/emoss08/trenova/pkg/dberror"
	"go.uber.org/zap"
)

const preferenceEntityName = "HomePreference"

var preferenceErrors = dberror.Mapping{Entity: preferenceEntityName}

func (r *repository) GetPreference(
	ctx context.Context,
	req *homelayoutports.GetHomePreferenceRequest,
) (*homelayout.Preference, bool, error) {
	return dbtx.Read2(ctx, r.db, func(ctx context.Context) (*homelayout.Preference, bool, error) {
		p := homelayoutdb.Preferences
		entity, err := p.Where(p.UserID.Eq(req.TenantInfo.UserID)).One(ctx, r.db.Keel(ctx))
		if dberror.IsNotFoundError(err) {
			return nil, false, nil
		}
		if err != nil {
			r.l.Error(
				"failed to get home preference",
				zap.String("userID", req.TenantInfo.UserID.String()),
				zap.Error(err),
			)
			return nil, false, err
		}

		return entity, true, nil
	})
}

func (r *repository) CreatePreference(
	ctx context.Context,
	entity *homelayout.Preference,
) (*homelayout.Preference, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*homelayout.Preference, error) {
		if err := homelayoutdb.Preferences.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			if dberror.IsUniqueConstraintViolation(err) {
				return nil, dberror.CreateVersionMismatchError(
					preferenceEntityName,
					entity.UserID.String(),
				)
			}
			r.l.Error(
				"failed to create home preference",
				zap.String("userID", entity.UserID.String()),
				zap.Error(err),
			)
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) UpdatePreference(
	ctx context.Context,
	entity *homelayout.Preference,
) (*homelayout.Preference, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*homelayout.Preference, error) {
		if err := homelayoutdb.Preferences.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to update home preference",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, preferenceErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}
