package customerpaymentrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

func (r *repository) ListApplicationsByInvoiceIDs(
	ctx context.Context,
	req *repositories.ListApplicationsByInvoiceIDsRequest,
) (map[pulid.ID][]*customerpayment.Application, error) {
	result := make(map[pulid.ID][]*customerpayment.Application, len(req.InvoiceIDs))
	if len(req.InvoiceIDs) == 0 {
		return result, nil
	}

	cols := buncolgen.ApplicationColumns
	rows := make([]*customerpayment.Application, 0, len(req.InvoiceIDs))
	if err := r.db.DBForContext(ctx).NewSelect().
		Model(&rows).
		Relation(buncolgen.ApplicationRelations.Payment).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ApplicationScopeTenant(sq, req.TenantInfo).
				Where(cols.InvoiceID.In(), bun.List(req.InvoiceIDs))
		}).
		Order(cols.CreatedAt.OrderDesc(), cols.LineNumber.OrderAsc()).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("list customer payment applications by invoice: %w", err)
	}

	for _, row := range rows {
		result[row.InvoiceID] = append(result[row.InvoiceID], row)
	}

	return result, nil
}

func (r *repository) CreateCreditMemoApplications(
	ctx context.Context,
	applications []*customerpayment.CreditMemoApplication,
) error {
	if len(applications) == 0 {
		return nil
	}
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(&applications).Exec(ctx); err != nil {
		return fmt.Errorf("create credit memo applications: %w", err)
	}

	return nil
}

func (r *repository) UpdateCreditMemoApplication(
	ctx context.Context,
	application *customerpayment.CreditMemoApplication,
) (*customerpayment.CreditMemoApplication, error) {
	cols := buncolgen.CreditMemoApplicationColumns
	application.UpdatedAt = timeutils.NowUnix()
	res, err := r.db.DBForContext(ctx).NewUpdate().
		Model(application).
		Column(
			cols.Status.Bare(),
			cols.UnappliedAt.Bare(),
			cols.UnappliedByID.Bare(),
			cols.UnappliedReason.Bare(),
			cols.UpdatedAt.Bare(),
		).
		WherePK().
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update credit memo application: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, "CreditMemoApplication", application.ID.String()); err != nil {
		return nil, err
	}

	return application, nil
}

func (r *repository) GetCreditMemoApplicationByID(
	ctx context.Context,
	req repositories.GetCreditMemoApplicationRequest,
) (*customerpayment.CreditMemoApplication, error) {
	cols := buncolgen.CreditMemoApplicationColumns
	entity := new(customerpayment.CreditMemoApplication)
	if err := r.db.DBForContext(ctx).NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CreditMemoApplicationScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "CreditMemoApplication")
	}

	return entity, nil
}

func (r *repository) ListCreditMemoApplicationsByInvoiceIDs(
	ctx context.Context,
	req *repositories.ListApplicationsByInvoiceIDsRequest,
) (map[pulid.ID][]*customerpayment.CreditMemoApplication, error) {
	result := make(map[pulid.ID][]*customerpayment.CreditMemoApplication, len(req.InvoiceIDs))
	if len(req.InvoiceIDs) == 0 {
		return result, nil
	}

	cols := buncolgen.CreditMemoApplicationColumns
	rows := make([]*customerpayment.CreditMemoApplication, 0, len(req.InvoiceIDs))
	if err := r.db.DBForContext(ctx).NewSelect().
		Model(&rows).
		Relation(buncolgen.CreditMemoApplicationRelations.CreditMemo).
		Relation(buncolgen.CreditMemoApplicationRelations.Invoice).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CreditMemoApplicationScopeTenant(sq, req.TenantInfo).
				WhereGroup(" OR ", func(inner *bun.SelectQuery) *bun.SelectQuery {
					return inner.
						Where(cols.InvoiceID.In(), bun.List(req.InvoiceIDs)).
						WhereOr(cols.CreditMemoInvoiceID.In(), bun.List(req.InvoiceIDs))
				})
		}).
		Order(cols.CreatedAt.OrderDesc(), cols.LineNumber.OrderAsc()).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("list credit memo applications by invoice: %w", err)
	}

	for _, row := range rows {
		result[row.InvoiceID] = append(result[row.InvoiceID], row)
		if row.CreditMemoInvoiceID != row.InvoiceID {
			result[row.CreditMemoInvoiceID] = append(result[row.CreditMemoInvoiceID], row)
		}
	}

	return result, nil
}
