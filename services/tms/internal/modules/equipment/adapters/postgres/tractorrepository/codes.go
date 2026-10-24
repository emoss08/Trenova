package tractorrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/tractorports"
	"go.uber.org/zap"
)

func normalizeLookupKeys(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		key := strings.ToLower(strings.TrimSpace(value))
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
	req tractorports.GetTractorsByCodesRequest,
) (map[string]*tractor.Tractor, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[string]*tractor.Tractor, error) {
		codes := normalizeLookupKeys(req.Codes)
		plates := normalizeLookupKeys(req.LicensePlates)
		if len(codes) == 0 && len(plates) == 0 {
			return map[string]*tractor.Tractor{}, nil
		}

		t := tractordb.Tractors
		entities, err := t.Where(keel.Or(
			keel.When(len(codes) > 0, t.Code.Lower().In(codes...)),
			keel.When(len(plates) > 0, t.LicensePlateNumber.Lower().In(plates...)),
		)).All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get tractors by code", zap.Error(err))
			return nil, fmt.Errorf("get tractors by codes: %w", err)
		}

		byCode := make(map[string]*tractor.Tractor, len(entities))
		for _, entity := range entities {
			byCode[strings.ToUpper(strings.TrimSpace(entity.Code))] = entity
		}

		return byCode, nil
	})
}
