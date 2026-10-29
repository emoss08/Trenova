package invoicedisputerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicedispute"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicedispute/invoicedisputedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var disputeErrors = dberror.Mapping{Entity: "InvoiceDispute"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) invoiceports.InvoiceDisputeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("repository.invoice-dispute"),
	}
}

func (r *repository) Create(
	ctx context.Context,
	entity *invoicedispute.InvoiceDispute,
) (*invoicedispute.InvoiceDispute, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*invoicedispute.InvoiceDispute, error) {
		entity.SyncMinor()
		if err := invoicedisputedb.InvoiceDisputes.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, disputeErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *invoicedispute.InvoiceDispute,
) (*invoicedispute.InvoiceDispute, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*invoicedispute.InvoiceDispute, error) {
		entity.SyncMinor()
		if err := invoicedisputedb.InvoiceDisputes.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, disputeErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req invoiceports.GetInvoiceDisputeByIDRequest,
) (*invoicedispute.InvoiceDispute, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*invoicedispute.InvoiceDispute, error) {
		entity, err := invoicedisputedb.InvoiceDisputes.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, disputeErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetOpenByInvoiceID(
	ctx context.Context,
	req invoiceports.GetOpenInvoiceDisputeRequest,
) (*invoicedispute.InvoiceDispute, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*invoicedispute.InvoiceDispute, error) {
		d := invoicedisputedb.InvoiceDisputes
		entity, err := d.Where(
			d.InvoiceID.Eq(req.InvoiceID),
			d.Status.Eq(invoicedispute.DisputeCaseStatusOpen),
		).First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, disputeErrors.Map(err, req.InvoiceID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListByInvoiceIDs(
	ctx context.Context,
	req *invoiceports.ListInvoiceDisputesByInvoiceIDsRequest,
) (map[pulid.ID][]*invoicedispute.InvoiceDispute, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID][]*invoicedispute.InvoiceDispute, error) {
			result := make(map[pulid.ID][]*invoicedispute.InvoiceDispute, len(req.InvoiceIDs))
			if len(req.InvoiceIDs) == 0 {
				return result, nil
			}

			d := invoicedisputedb.InvoiceDisputes
			rows, err := d.Where(d.InvoiceID.In(req.InvoiceIDs...)).
				Order(d.OpenedAt.Desc(), d.ID.Desc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, err
			}

			for _, row := range rows {
				result[row.InvoiceID] = append(result[row.InvoiceID], row)
			}

			return result, nil
		},
	)
}
