package accountingsyncrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	backfillEntity           = "Accounting backfill"
	defaultBackfillListLimit = 20
	maxBackfillListLimit     = 100
)

type BackfillParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type backfillRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewBackfillRepository(p BackfillParams) repositories.AccountingBackfillRepository {
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
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, repositories.ErrAccountingBackfillActive
		}
		return nil, fmt.Errorf("create accounting backfill: %w", err)
	}

	return entity, nil
}

func (r *backfillRepository) GetByID(
	ctx context.Context,
	req repositories.GetAccountingBackfillRequest,
) (*accountingsync.AccountingBackfill, error) {
	entity := new(accountingsync.AccountingBackfill)
	cols := buncolgen.AccountingBackfillColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Apply(buncolgen.AccountingBackfillApplyTenant(req.TenantInfo)).
		Where(cols.ID.Eq(), req.ID).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, backfillEntity)
	}

	return entity, nil
}

func (r *backfillRepository) GetActive(
	ctx context.Context,
	req repositories.AccountingSyncConnectionRequest,
) (*accountingsync.AccountingBackfill, error) {
	entity := new(accountingsync.AccountingBackfill)
	cols := buncolgen.AccountingBackfillColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Apply(buncolgen.AccountingBackfillApplyTenant(req.TenantInfo)).
		Where(cols.ConnectionID.Eq(), req.ConnectionID).
		Where(cols.Status.In(), bun.List(inProgressBackfillStatuses())).
		Limit(1).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, backfillEntity)
	}

	return entity, nil
}

func (r *backfillRepository) ListByConnection(
	ctx context.Context,
	req repositories.ListAccountingBackfillsRequest,
) ([]*accountingsync.AccountingBackfill, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultBackfillListLimit
	}
	limit = intutils.Clamp(limit, 1, maxBackfillListLimit)

	cols := buncolgen.AccountingBackfillColumns
	entities := make([]*accountingsync.AccountingBackfill, 0, limit)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(buncolgen.AccountingBackfillApplyTenant(req.TenantInfo)).
		Where(cols.ConnectionID.Eq(), req.ConnectionID).
		Order(cols.CreatedAt.OrderDesc(), cols.ID.OrderDesc()).
		Limit(limit).
		Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *backfillRepository) Update(
	ctx context.Context,
	entity *accountingsync.AccountingBackfill,
) (*accountingsync.AccountingBackfill, error) {
	cols := buncolgen.AccountingBackfillColumns
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		ExcludeColumn(cols.CreatedAt.Bare()).
		WherePK().
		Where(cols.Version.Eq(), ov).
		Exec(ctx)
	if err != nil {
		entity.Version = ov
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, repositories.ErrAccountingBackfillActive
		}
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, backfillEntity, entity.ID.String()); err != nil {
		entity.Version = ov
		return nil, err
	}

	return entity, nil
}
