package userrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/buncolgen/tenantcol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func (ur *repository) UpdateColumns(
	ctx context.Context,
	req tenantports.UpdateUserColumnsRequest,
) (*tenant.User, error) {
	return dbtx.Write(ctx, ur.db, func(ctx context.Context) (*tenant.User, error) {
		entity := req.Entity
		log := ur.l.With(
			zap.String("operation", "UpdateColumns"),
			zap.String("userID", entity.ID.String()),
			zap.Strings("columns", req.Columns),
		)
		cols := tenantcol.UserColumns

		ov := entity.Version
		entity.Version++

		columns := make([]string, 0, len(req.Columns)+2)
		columns = append(columns, req.Columns...)
		columns = append(columns, cols.Version.String(), cols.UpdatedAt.String())

		result, err := ur.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			Column(columns...).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return uq.Where(cols.ID.Eq(), entity.ID).
					Where(cols.Version.Eq(), ov).
					Where(cols.CurrentOrganizationID.Eq(), entity.CurrentOrganizationID).
					Where(cols.BusinessUnitID.Eq(), entity.BusinessUnitID)
			}).
			Returning("*").
			Exec(ctx)
		if err != nil {
			log.Error("failed to update user columns", zap.Error(err))
			return nil, err
		}

		if err = dberror.CheckRowsAffected(result, "User", entity.ID.String()); err != nil {
			return nil, err
		}

		return entity, nil
	})
}
