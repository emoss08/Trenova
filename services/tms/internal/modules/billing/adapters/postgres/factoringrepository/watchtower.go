package factoringrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/pkg/buncolgen/customercol"
	"github.com/emoss08/trenova/pkg/buncolgen/factoringcol"
	"github.com/emoss08/trenova/pkg/buncolgen/factoringcompanycol"
	"github.com/emoss08/trenova/pkg/buncolgen/invoicecol"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	maxAttentionRows = 500
	coalesceBlank    = "COALESCE({}, '')"
)

type watchtowerReader struct {
	db kernelports.DBConnection
	l  *zap.Logger
}

func NewWatchtowerReader(p Params) factoringports.WatchtowerReader {
	return &watchtowerReader{
		db: p.DB,
		l:  p.Logger.Named("repository.factoring-watchtower"),
	}
}

func (r *watchtowerReader) ListInvoiceLabels(
	ctx context.Context,
	req *factoringports.ListInvoiceLabelsRequest,
) (map[pulid.ID]*factoringports.InvoiceLabel, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]*factoringports.InvoiceLabel, error) {
			result := make(map[pulid.ID]*factoringports.InvoiceLabel, len(req.InvoiceIDs))
			if len(req.InvoiceIDs) == 0 {
				return result, nil
			}

			inv := invoicecol.InvoiceColumns
			cus := customercol.CustomerColumns
			cusTable := customercol.CustomerTable
			fac := factoringcompanycol.FactoringCompanyColumns
			facTable := factoringcompanycol.FactoringCompanyTable

			rows := make([]*factoringports.InvoiceLabel, 0, len(req.InvoiceIDs))
			if err := r.db.DBForContext(ctx).NewSelect().
				Model((*invoice.Invoice)(nil)).
				ColumnExpr(inv.ID.As("invoice_id")).
				ColumnExpr(inv.Number.As("number")).
				ColumnExpr(inv.CurrencyCode.As("currency_code")).
				ColumnExpr(cus.Name.Expr(coalesceBlank)+" AS customer_name").
				ColumnExpr(fac.Name.Expr(coalesceBlank)+" AS factor_name").
				Join("LEFT JOIN "+cusTable.As(cusTable.Alias)).
				JoinOn(cus.ID.EqColumn(inv.CustomerID)).
				JoinOn(cus.OrganizationID.EqColumn(inv.OrganizationID)).
				JoinOn(cus.BusinessUnitID.EqColumn(inv.BusinessUnitID)).
				Join("LEFT JOIN "+facTable.As(facTable.Alias)).
				JoinOn(fac.ID.EqColumn(inv.FactoringCompanyID)).
				JoinOn(fac.OrganizationID.EqColumn(inv.OrganizationID)).
				JoinOn(fac.BusinessUnitID.EqColumn(inv.BusinessUnitID)).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return invoicecol.InvoiceScopeTenant(sq, req.TenantInfo).
						Where(inv.ID.In(), bun.List(req.InvoiceIDs))
				}).
				Scan(ctx, &rows); err != nil {
				return nil, fmt.Errorf("list factoring invoice labels: %w", err)
			}

			for _, row := range rows {
				result[row.InvoiceID] = row
			}

			return result, nil
		},
	)
}

func (r *watchtowerReader) ListCustomersMissingNoa(
	ctx context.Context,
	req *factoringports.ListCustomersMissingNoaRequest,
) ([]*factoringports.CustomerMissingNoa, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*factoringports.CustomerMissingNoa, error) {
			cbp := customercol.CustomerBillingProfileColumns
			cus := customercol.CustomerColumns
			cusTable := customercol.CustomerTable
			fac := factoringcompanycol.FactoringCompanyColumns
			facTable := factoringcompanycol.FactoringCompanyTable
			noa := factoringcol.NoaColumns
			noaTable := factoringcol.NoaTable

			limit := boundedLimit(req.Limit, maxAttentionRows)
			rows := make([]*factoringports.CustomerMissingNoa, 0, min(limit, 64))
			if err := r.db.DBForContext(ctx).NewSelect().
				Model((*customer.CustomerBillingProfile)(nil)).
				ColumnExpr(cbp.CustomerID.As("customer_id")).
				ColumnExpr(cus.Code.As("customer_code")).
				ColumnExpr(cus.Name.As("customer_name")).
				ColumnExpr(cbp.FactoringCompanyID.As("factoring_company_id")).
				ColumnExpr(fac.Name.As("factor_name")).
				ColumnExpr(cbp.UpdatedAt.As("profile_updated_at")).
				ColumnExpr(noa.ID.As("noa_id")).
				ColumnExpr(noa.Status.Expr(coalesceBlank)+" AS noa_status").
				ColumnExpr(noa.SentAt.As("noa_sent_at")).
				ColumnExpr(noa.ExpiresAt.As("noa_expires_at")).
				ColumnExpr(noa.CreatedAt.As("noa_created_at")).
				Join("JOIN "+cusTable.As(cusTable.Alias)).
				JoinOn(cus.ID.EqColumn(cbp.CustomerID)).
				JoinOn(cus.OrganizationID.EqColumn(cbp.OrganizationID)).
				JoinOn(cus.BusinessUnitID.EqColumn(cbp.BusinessUnitID)).
				Join("JOIN "+facTable.As(facTable.Alias)).
				JoinOn(fac.ID.EqColumn(cbp.FactoringCompanyID)).
				JoinOn(fac.OrganizationID.EqColumn(cbp.OrganizationID)).
				JoinOn(fac.BusinessUnitID.EqColumn(cbp.BusinessUnitID)).
				Join("LEFT JOIN "+noaTable.As(noaTable.Alias)).
				JoinOn(noa.CustomerID.EqColumn(cbp.CustomerID)).
				JoinOn(noa.FactoringCompanyID.EqColumn(cbp.FactoringCompanyID)).
				JoinOn(noa.OrganizationID.EqColumn(cbp.OrganizationID)).
				JoinOn(noa.BusinessUnitID.EqColumn(cbp.BusinessUnitID)).
				JoinOn(noa.Status.Ne(), factoring.NoaStatusReleased).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return customercol.CustomerBillingProfileScopeTenant(sq, req.TenantInfo).
						Where(cbp.FactoringCompanyID.IsNotNull()).
						Where(cus.Status.Eq(), domaintypes.StatusActive).
						WhereGroup(" AND ", func(gq *bun.SelectQuery) *bun.SelectQuery {
							return gq.
								Where(noa.ID.IsNull()).
								WhereOr(noa.Status.Ne(), factoring.NoaStatusAcknowledged).
								WhereOr(noa.ExpiresAt.Lte(), req.At)
						})
				}).
				Order(cbp.UpdatedAt.OrderAsc(), cbp.CustomerID.OrderAsc()).
				Limit(int64(limit)).
				Scan(ctx, &rows); err != nil {
				return nil, fmt.Errorf("list customers missing a notice of assignment: %w", err)
			}

			return rows, nil
		},
	)
}

func (r *watchtowerReader) ListNoasExpiringSoon(
	ctx context.Context,
	req *factoringports.ListNoasExpiringSoonRequest,
) ([]*factoringports.ExpiringNoa, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*factoringports.ExpiringNoa, error) {
		noa := factoringcol.NoaColumns
		cus := customercol.CustomerColumns
		cusTable := customercol.CustomerTable
		fac := factoringcompanycol.FactoringCompanyColumns
		facTable := factoringcompanycol.FactoringCompanyTable

		limit := boundedLimit(req.Limit, maxAttentionRows)
		rows := make([]*factoringports.ExpiringNoa, 0, min(limit, 64))
		if err := r.db.DBForContext(ctx).NewSelect().
			Model((*factoring.Noa)(nil)).
			ColumnExpr(noa.ID.As("noa_id")).
			ColumnExpr(noa.CustomerID.As("customer_id")).
			ColumnExpr(cus.Name.Expr(coalesceBlank)+" AS customer_name").
			ColumnExpr(noa.FactoringCompanyID.As("factoring_company_id")).
			ColumnExpr(fac.Name.Expr(coalesceBlank)+" AS factor_name").
			ColumnExpr(noa.AcknowledgedAt.As("acknowledged_at")).
			ColumnExpr(noa.ExpiresAt.As("expires_at")).
			Join("LEFT JOIN "+cusTable.As(cusTable.Alias)).
			JoinOn(cus.ID.EqColumn(noa.CustomerID)).
			JoinOn(cus.OrganizationID.EqColumn(noa.OrganizationID)).
			JoinOn(cus.BusinessUnitID.EqColumn(noa.BusinessUnitID)).
			Join("LEFT JOIN "+facTable.As(facTable.Alias)).
			JoinOn(fac.ID.EqColumn(noa.FactoringCompanyID)).
			JoinOn(fac.OrganizationID.EqColumn(noa.OrganizationID)).
			JoinOn(fac.BusinessUnitID.EqColumn(noa.BusinessUnitID)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return factoringcol.NoaScopeTenant(sq, req.TenantInfo).
					Where(noa.Status.Eq(), factoring.NoaStatusAcknowledged).
					Where(noa.ExpiresAt.IsNotNull()).
					Where(noa.ExpiresAt.Gt(), req.At).
					Where(noa.ExpiresAt.Lte(), req.Before)
			}).
			Order(noa.ExpiresAt.OrderAsc(), noa.ID.OrderAsc()).
			Limit(int64(limit)).
			Scan(ctx, &rows); err != nil {
			return nil, fmt.Errorf("list notices of assignment expiring soon: %w", err)
		}

		return rows, nil
	})
}

func (r *watchtowerReader) ListFailedSubmissions(
	ctx context.Context,
	req *factoringports.ListFailedSubmissionsRequest,
) ([]*factoringports.FailedSubmission, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*factoringports.FailedSubmission, error) {
			sub := factoringcol.SubmissionColumns
			fac := factoringcompanycol.FactoringCompanyColumns
			facTable := factoringcompanycol.FactoringCompanyTable

			limit := boundedLimit(req.Limit, maxAttentionRows)
			rows := make([]*factoringports.FailedSubmission, 0, min(limit, 16))
			if err := r.db.DBForContext(ctx).NewSelect().
				Model((*factoring.Submission)(nil)).
				ColumnExpr(sub.ID.As("submission_id")).
				ColumnExpr(sub.Number.As("number")).
				ColumnExpr(sub.FactoringCompanyID.As("factoring_company_id")).
				ColumnExpr(fac.Name.Expr(coalesceBlank)+" AS factor_name").
				ColumnExpr(sub.CurrencyCode.As("currency_code")).
				ColumnExpr(sub.InvoiceCount.As("invoice_count")).
				ColumnExpr(sub.FaceMinor.As("face_minor")).
				ColumnExpr(sub.TransportTarget.As("transport_target")).
				ColumnExpr(sub.LastError.Expr(coalesceBlank)+" AS last_error").
				ColumnExpr(sub.FailedAt.As("failed_at")).
				Join("LEFT JOIN "+facTable.As(facTable.Alias)).
				JoinOn(fac.ID.EqColumn(sub.FactoringCompanyID)).
				JoinOn(fac.OrganizationID.EqColumn(sub.OrganizationID)).
				JoinOn(fac.BusinessUnitID.EqColumn(sub.BusinessUnitID)).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return factoringcol.SubmissionScopeTenant(sq, req.TenantInfo).
						Where(sub.Status.Eq(), factoring.SubmissionStatusFailed).
						Where(sub.FailedAt.IsNotNull())
				}).
				Order(sub.FailedAt.OrderAsc(), sub.ID.OrderAsc()).
				Limit(int64(limit)).
				Scan(ctx, &rows); err != nil {
				return nil, fmt.Errorf("list failed factoring schedules: %w", err)
			}

			return rows, nil
		},
	)
}
