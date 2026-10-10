package invoicesharerepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/pkg/buncolgen/invoicecol"
	"github.com/emoss08/trenova/pkg/buncolgen/tenantcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
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

func New(p Params) invoiceports.InvoiceShareRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.invoice-share-repository"),
	}
}

var recipientConflict = "CONFLICT (" + strings.Join([]string{
	invoicecol.InvoiceShareColumns.InvoiceID.Bare(),
	invoicecol.InvoiceShareColumns.OrganizationID.Bare(),
	invoicecol.InvoiceShareColumns.BusinessUnitID.Bare(),
	invoicecol.InvoiceShareColumns.SharedWithID.Bare(),
}, ", ") + ") DO UPDATE"

func (r *repository) Upsert(ctx context.Context, shares []*invoice.InvoiceShare) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(shares) == 0 {
			return nil
		}

		cols := invoicecol.InvoiceShareColumns
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(&shares).
			On(recipientConflict).
			Set(cols.SharedByID.SetExcluded()).
			Set(cols.Note.SetExcluded()).
			Set(cols.Tab.SetExcluded()).
			Set(cols.LastSharedAt.SetExcluded()).
			Set(cols.UpdatedAt.SetExcluded()).
			Set(cols.ShareCount.IncConflict(1)).
			Exec(ctx); err != nil {
			r.l.Error("failed to upsert invoice shares", zap.Error(err))
			return fmt.Errorf("upsert invoice shares: %w", err)
		}

		return nil
	})
}

func (r *repository) ListByInvoiceID(
	ctx context.Context,
	req *invoiceports.ListInvoiceSharesRequest,
) ([]*invoice.InvoiceShare, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*invoice.InvoiceShare, error) {
		cols := invoicecol.InvoiceShareColumns
		rel := invoicecol.InvoiceShareRelations

		shares := make([]*invoice.InvoiceShare, 0)
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&shares).
			Apply(invoicecol.InvoiceShareApplyTenant(req.TenantInfo)).
			Where(cols.InvoiceID.Eq(), req.InvoiceID).
			Relation(rel.SharedWith, selectUserSummary).
			Relation(rel.SharedBy, selectUserSummary).
			Order(cols.LastSharedAt.OrderDesc(), cols.ID.OrderAsc()).
			Scan(ctx); err != nil {
			r.l.Error("failed to list invoice shares", zap.Error(err))
			return nil, fmt.Errorf("list invoice shares: %w", err)
		}

		return shares, nil
	})
}

func selectUserSummary(q *bun.SelectQuery) *bun.SelectQuery {
	users := tenantcol.UserColumns
	return q.Column(
		users.ID.Bare(),
		users.Name.Bare(),
		users.Username.Bare(),
		users.EmailAddress.Bare(),
		users.Status.Bare(),
		users.ProfilePicURL.Bare(),
		users.ThumbnailURL.Bare(),
	)
}
