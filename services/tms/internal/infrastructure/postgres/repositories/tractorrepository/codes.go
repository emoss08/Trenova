package tractorrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func normalizeLookupKeys(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		key := strings.ToUpper(strings.TrimSpace(value))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func (r *repository) GetByCodes(
	ctx context.Context,
	req repositories.GetTractorsByCodesRequest,
) (map[string]*tractor.Tractor, error) {
	codes := normalizeLookupKeys(req.Codes)
	plates := normalizeLookupKeys(req.LicensePlates)
	if len(codes) == 0 && len(plates) == 0 {
		return map[string]*tractor.Tractor{}, nil
	}

	cols := buncolgen.TractorColumns
	entities := make([]*tractor.Tractor, 0, len(codes)+len(plates))

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		ColumnExpr(buncolgen.TractorTable.All()).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.TractorScopeTenant(sq, req.TenantInfo).
				WhereGroup(" AND ", func(oq *bun.SelectQuery) *bun.SelectQuery {
					if len(codes) > 0 {
						oq = oq.Where(cols.Code.Expr("UPPER({}) IN (?)"), bun.In(codes))
					}
					if len(plates) > 0 {
						oq = oq.WhereOr(
							cols.LicensePlateNumber.Expr("UPPER({}) IN (?)"),
							bun.In(plates),
						)
					}
					return oq
				})
		}).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to get tractors by code", zap.Error(err))
		return nil, fmt.Errorf("get tractors by codes: %w", err)
	}

	byCode := make(map[string]*tractor.Tractor, len(entities))
	for _, entity := range entities {
		byCode[strings.ToUpper(strings.TrimSpace(entity.Code))] = entity
	}

	return byCode, nil
}
