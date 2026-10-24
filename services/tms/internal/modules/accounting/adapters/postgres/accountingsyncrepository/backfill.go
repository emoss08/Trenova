package accountingsyncrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync/accountingsyncdb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/intutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	backfillEntity           = "Accounting backfill"
	defaultBackfillListLimit = 20
	maxBackfillListLimit     = 100
)

var backfillErrors = dberror.Mapping{Entity: backfillEntity}

type BackfillParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type backfillRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewBackfillRepository(p BackfillParams) accountingsyncports.AccountingBackfillRepository {
	return &backfillRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-backfill-repository"),
	}
}

func inProgressBackfillStatuses() []accountingsync.BackfillStatus {
	statuses := make([]accountingsync.BackfillStatus, 0, 3)
	for _, status := range accountingsync.AllBackfillStatuses() {
		if status.InProgress() {
			statuses = append(statuses, status)
		}
	}
	return statuses
}

func (r *backfillRepository) Create(
	ctx context.Context,
	entity *accountingsync.AccountingBackfill,
) (*accountingsync.AccountingBackfill, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingBackfill, error) {
			err := accountingsyncdb.AccountingBackfills.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				if dberror.IsUniqueConstraintViolation(err) {
					return nil, accountingsyncports.ErrAccountingBackfillActive
				}
				return nil, fmt.Errorf("create accounting backfill: %w", err)
			}

			return entity, nil
		},
	)
}

func (r *backfillRepository) GetByID(
	ctx context.Context,
	req accountingsyncports.GetAccountingBackfillRequest,
) (*accountingsync.AccountingBackfill, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingBackfill, error) {
			entity, err := accountingsyncdb.AccountingBackfills.Get(ctx, r.db.Keel(ctx), req.ID)
			if err != nil {
				return nil, backfillErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *backfillRepository) GetActive(
	ctx context.Context,
	req accountingsyncports.AccountingSyncConnectionRequest,
) (*accountingsync.AccountingBackfill, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingBackfill, error) {
			ab := accountingsyncdb.AccountingBackfills
			entity, err := ab.Where(
				ab.ConnectionID.Eq(req.ConnectionID),
				ab.Status.In(inProgressBackfillStatuses()...),
			).First(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, backfillErrors.Map(err, "")
			}

			return entity, nil
		},
	)
}

func (r *backfillRepository) ListByConnection(
	ctx context.Context,
	req accountingsyncports.ListAccountingBackfillsRequest,
) ([]*accountingsync.AccountingBackfill, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountingsync.AccountingBackfill, error) {
			limit := req.Limit
			if limit <= 0 {
				limit = defaultBackfillListLimit
			}
			limit = intutils.Clamp(limit, 1, maxBackfillListLimit)

			ab := accountingsyncdb.AccountingBackfills
			return ab.Where(ab.ConnectionID.Eq(req.ConnectionID)).
				Order(ab.CreatedAt.Desc(), ab.ID.Desc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx))
		},
	)
}

func (r *backfillRepository) Update(
	ctx context.Context,
	entity *accountingsync.AccountingBackfill,
) (*accountingsync.AccountingBackfill, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingBackfill, error) {
			ov := entity.Version
			err := accountingsyncdb.AccountingBackfills.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				entity.Version = ov
				if dberror.IsUniqueConstraintViolation(err) {
					return nil, accountingsyncports.ErrAccountingBackfillActive
				}
				return nil, backfillErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}
