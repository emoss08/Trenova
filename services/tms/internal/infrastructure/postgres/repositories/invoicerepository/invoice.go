package invoicerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
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

func New(p Params) repositories.InvoiceRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.invoice-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListInvoicesRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"inv",
		req.Filter,
		(*invoice.Invoice)(nil),
	)

	q = q.Relation("Customer").Relation("Shipment")

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListInvoicesRequest,
) (*pagination.ListResult[*invoice.Invoice], error) {
	entities := make([]*invoice.Invoice, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*invoice.Invoice]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetInvoiceByIDRequest,
) (*invoice.Invoice, error) {
	entity := new(invoice.Invoice)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("inv.id = ?", req.ID).
		Where("inv.organization_id = ?", req.TenantInfo.OrgID).
		Where("inv.business_unit_id = ?", req.TenantInfo.BuID).
		Relation("Customer").
		Relation("Shipment").
		Relation("BillingQueueItem").
		Relation("Lines", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("invl.line_number ASC")
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Invoice")
	}

	return entity, nil
}

func (r *repository) GetByBillingQueueItemID(
	ctx context.Context,
	req repositories.GetInvoiceByBillingQueueItemIDRequest,
) (*invoice.Invoice, error) {
	var entity invoice.Invoice
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entity).
		Where("inv.billing_queue_item_id = ?", req.BillingQueueItemID).
		Where("inv.organization_id = ?", req.TenantInfo.OrgID).
		Where("inv.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Invoice")
	}

	return r.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         entity.ID,
		TenantInfo: req.TenantInfo,
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *invoice.Invoice,
) (*invoice.Invoice, error) {
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if _, err := r.db.DBForContext(txCtx).NewInsert().Model(entity).Exec(txCtx); err != nil {
			return fmt.Errorf("insert invoice: %w", err)
		}

		if len(entity.Lines) > 0 {
			for _, line := range entity.Lines {
				line.InvoiceID = entity.ID
				line.OrganizationID = entity.OrganizationID
				line.BusinessUnitID = entity.BusinessUnitID
			}

			if _, err := r.db.DBForContext(txCtx).
				NewInsert().
				Model(&entity.Lines).
				Exec(txCtx); err != nil {
				return fmt.Errorf("insert invoice lines: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return r.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         entity.ID,
		TenantInfo: tenantInfo(entity),
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *invoice.Invoice,
) (*invoice.Invoice, error) {
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Where("inv.id = ?", entity.ID).
		Where("inv.organization_id = ?", entity.OrganizationID).
		Where("inv.business_unit_id = ?", entity.BusinessUnitID).
		Where("inv.version = ?", entity.Version).
		Set("status = ?", entity.Status).
		Set("posted_at = ?", entity.PostedAt).
		Set("due_date = ?", entity.DueDate).
		Set("version = version + 1").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update invoice: %w", err)
	}

	if err = dberror.CheckRowsAffected(result, "Invoice", entity.ID.String()); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         entity.ID,
		TenantInfo: tenantInfo(entity),
	})
}

func tenantInfo(entity *invoice.Invoice) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}
}
