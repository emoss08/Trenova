package invoicedisputerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     ports.DBConnection
	Logger *zap.Logger
}

type repository struct {
	db ports.DBConnection
	l  *zap.Logger
}

func New(p Params) repositories.InvoiceDisputeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("repository.invoice-dispute"),
	}
}

func (r *repository) Create(
	ctx context.Context,
	entity *invoice.InvoiceDispute,
) (*invoice.InvoiceDispute, error) {
	entity.SyncMinor()
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		return nil, fmt.Errorf("create invoice dispute: %w", err)
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *invoice.InvoiceDispute,
) (*invoice.InvoiceDispute, error) {
	entity.SyncMinor()
	previousVersion := entity.Version
	entity.Version++

	res, err := r.db.DBForContext(ctx).NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.InvoiceDisputeColumns.Version.Eq(), previousVersion).
		Returning("*").
		Exec(ctx)
	if err != nil {
		entity.Version = previousVersion
		return nil, fmt.Errorf("update invoice dispute: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, "InvoiceDispute", entity.ID.String()); err != nil {
		entity.Version = previousVersion
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetInvoiceDisputeByIDRequest,
) (*invoice.InvoiceDispute, error) {
	cols := buncolgen.InvoiceDisputeColumns
	entity := new(invoice.InvoiceDispute)
	if err := r.db.DBForContext(ctx).NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.InvoiceDisputeScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "InvoiceDispute")
	}

	return entity, nil
}

func (r *repository) GetOpenByInvoiceID(
	ctx context.Context,
	req repositories.GetOpenInvoiceDisputeRequest,
) (*invoice.InvoiceDispute, error) {
	cols := buncolgen.InvoiceDisputeColumns
	entity := new(invoice.InvoiceDispute)
	if err := r.db.DBForContext(ctx).NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.InvoiceDisputeScopeTenant(sq, req.TenantInfo).
				Where(cols.InvoiceID.Eq(), req.InvoiceID).
				Where(cols.Status.Eq(), invoice.DisputeCaseStatusOpen)
		}).
		Limit(1).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "InvoiceDispute")
	}

	return entity, nil
}

func (r *repository) ListByInvoiceIDs(
	ctx context.Context,
	req *repositories.ListInvoiceDisputesByInvoiceIDsRequest,
) (map[pulid.ID][]*invoice.InvoiceDispute, error) {
	result := make(map[pulid.ID][]*invoice.InvoiceDispute, len(req.InvoiceIDs))
	if len(req.InvoiceIDs) == 0 {
		return result, nil
	}

	cols := buncolgen.InvoiceDisputeColumns
	rows := make([]*invoice.InvoiceDispute, 0, len(req.InvoiceIDs))
	if err := r.db.DBForContext(ctx).NewSelect().
		Model(&rows).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.InvoiceDisputeScopeTenant(sq, req.TenantInfo).
				Where(cols.InvoiceID.In(), bun.List(req.InvoiceIDs))
		}).
		Order(cols.OpenedAt.OrderDesc(), cols.ID.OrderDesc()).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("list invoice disputes by invoice: %w", err)
	}

	for _, row := range rows {
		result[row.InvoiceID] = append(result[row.InvoiceID], row)
	}

	return result, nil
}
