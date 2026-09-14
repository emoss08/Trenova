package invoicesharerepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
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

func New(p Params) repositories.InvoiceShareRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.invoice-share-repository"),
	}
}

var recipientConflict = "CONFLICT (" + strings.Join([]string{
	buncolgen.InvoiceShareColumns.InvoiceID.Bare(),
	buncolgen.InvoiceShareColumns.OrganizationID.Bare(),
	buncolgen.InvoiceShareColumns.BusinessUnitID.Bare(),
	buncolgen.InvoiceShareColumns.SharedWithID.Bare(),
}, ", ") + ") DO UPDATE"

func (r *repository) Upsert(ctx context.Context, shares []*invoice.InvoiceShare) error {
	if len(shares) == 0 {
		return nil
	}

	cols := buncolgen.InvoiceShareColumns
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
}

func (r *repository) ListByInvoiceID(
	ctx context.Context,
	req *repositories.ListInvoiceSharesRequest,
) ([]*invoice.InvoiceShare, error) {
	cols := buncolgen.InvoiceShareColumns
	rel := buncolgen.InvoiceShareRelations

	shares := make([]*invoice.InvoiceShare, 0)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&shares).
		Apply(buncolgen.InvoiceShareApplyTenant(req.TenantInfo)).
		Where(cols.InvoiceID.Eq(), req.InvoiceID).
		Relation(rel.SharedWith, selectUserSummary).
		Relation(rel.SharedBy, selectUserSummary).
		Order(cols.LastSharedAt.OrderDesc(), cols.ID.OrderAsc()).
		Scan(ctx); err != nil {
		r.l.Error("failed to list invoice shares", zap.Error(err))
		return nil, fmt.Errorf("list invoice shares: %w", err)
	}

	return shares, nil
}

func selectUserSummary(q *bun.SelectQuery) *bun.SelectQuery {
	users := buncolgen.UserColumns
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
