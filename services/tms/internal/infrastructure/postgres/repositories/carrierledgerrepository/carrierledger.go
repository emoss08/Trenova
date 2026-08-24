package carrierledgerrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
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

func New(p Params) repositories.CarrierLedgerRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.carrier-ledger-repository")}
}

func (r *repository) AppendEntries(
	ctx context.Context,
	entries []*carriersettlement.LedgerEntry,
) error {
	records := make([]*carriersettlement.LedgerEntry, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		records = append(records, entry)
	}
	if len(records) == 0 {
		return nil
	}
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(&records).Exec(ctx); err != nil {
		return fmt.Errorf("append carrier ledger entries: %w", err)
	}
	return nil
}

func (r *repository) ListByCarrier(
	ctx context.Context,
	req *repositories.ListCarrierLedgerEntriesRequest,
) ([]*carriersettlement.LedgerEntry, error) {
	cols := buncolgen.LedgerEntryColumns
	limit := req.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	items := make([]*carriersettlement.LedgerEntry, 0, limit)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.LedgerEntryScopeTenant(sq, req.TenantInfo).
				Where(cols.CarrierID.Eq(), req.CarrierID)
		}).
		Order(cols.TransactionDate.OrderDesc()).
		Order(cols.LineNumber.OrderAsc()).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list carrier ledger entries: %w", err)
	}
	return items, nil
}
