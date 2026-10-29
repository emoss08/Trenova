package invoicesharerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceshare"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/pkg/dberror"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var shareErrors = dberror.Mapping{Entity: "Invoice share"}

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

func (r *repository) Upsert(ctx context.Context, shares []*invoiceshare.InvoiceShare) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(shares) == 0 {
			return nil
		}

		s := invoicedb.InvoiceShares
		_, err := s.Upsert(ctx, r.db.Keel(ctx), shares,
			keel.OnConflict(s.InvoiceID, s.OrganizationID, s.BusinessUnitID, s.SharedWithID).
				Update(s.SharedByID, s.Note, s.Tab, s.LastSharedAt).
				Set(s.ShareCount.Increment(1)),
		)
		if err != nil {
			r.l.Error("failed to upsert invoice shares", zap.Error(err))
			return shareErrors.Map(err, "")
		}

		return nil
	})
}

func (r *repository) ListByInvoiceID(
	ctx context.Context,
	req *invoiceports.ListInvoiceSharesRequest,
) ([]*invoiceshare.InvoiceShare, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*invoiceshare.InvoiceShare, error) {
		s := invoicedb.InvoiceShares
		shares, err := s.Where(s.InvoiceID.Eq(req.InvoiceID)).
			Load(userSummary(s.SharedWith()), userSummary(s.SharedBy())).
			Order(s.LastSharedAt.Desc(), s.ID.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list invoice shares", zap.Error(err))
			return nil, err
		}
		if shares == nil {
			return []*invoiceshare.InvoiceShare{}, nil
		}

		return shares, nil
	})
}

func userSummary(u *tenantdb.UsersTable) *keel.Query[*tenant.User] {
	return u.Pick(
		u.ID,
		u.Name,
		u.Username,
		u.EmailAddress,
		u.Status,
		u.ProfilePicURL,
		u.ThumbnailURL,
	)
}
