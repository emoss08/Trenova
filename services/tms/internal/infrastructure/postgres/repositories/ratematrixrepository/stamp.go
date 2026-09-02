package ratematrixrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// GetLookupStamp summarises the tenant's matrices as count, highest version,
// and latest update, which is enough to tell whether a built lookup is stale.
// Cell replacement bumps the matrix version, so a new sheet moves the stamp
// even though no header field changed.
func (r *repository) GetLookupStamp(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (string, error) {
	cols := buncolgen.RateMatrixColumns

	var stamp string
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*ratematrix.RateMatrix)(nil)).
		ColumnExpr(
			"concat_ws(':', count(*), coalesce(max(?), 0), coalesce(max(?), 0))",
			bun.Ident(cols.Version.Name),
			bun.Ident(cols.UpdatedAt.Name),
		).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RateMatrixScopeTenant(sq, tenantInfo)
		}).
		Scan(ctx, &stamp)
	if err != nil {
		r.l.With(zap.String("operation", "GetLookupStamp")).
			Error("failed to read rate matrix stamp", zap.Error(err))
		return "", err
	}

	return stamp, nil
}
