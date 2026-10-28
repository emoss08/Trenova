package edicarrierinvoicerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edicarrierinvoice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edicarrierinvoice/edicarrierinvoicedb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var carrierInvoiceErrors = dberror.Mapping{Entity: "CarrierInvoice"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) ediports.EDICarrierInvoiceRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-carrier-invoice-repository"),
	}
}

func (r *repository) ListCarrierInvoices(
	ctx context.Context,
	req *ediports.ListEDICarrierInvoicesRequest,
) (*pagination.ListResult[*edicarrierinvoice.CarrierInvoice], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edicarrierinvoice.CarrierInvoice], error) {
			ci := edicarrierinvoicedb.CarrierInvoices
			q := ci.Where(
				keel.When(
					req.ReconciliationStatus != "",
					ci.ReconciliationStatus.Eq(req.ReconciliationStatus),
				),
				keel.When(req.PartnerID.IsNotNil(), ci.EDIPartnerID.Eq(req.PartnerID)),
				keel.When(req.ShipmentID.IsNotNil(), ci.ShipmentID.Eq(req.ShipmentID)),
			)

			return listquery.List(
				ctx,
				r.db.Keel(ctx),
				q,
				&listquery.Request[edicarrierinvoice.CarrierInvoice]{
					Filter: req.Filter,
					Shape: func(q *keel.Query[*edicarrierinvoice.CarrierInvoice]) *keel.Query[*edicarrierinvoice.CarrierInvoice] {
						return q.Load(ci.Partner())
					},
					Then: []keel.Order{ci.CreatedAt.Desc()},
				},
			)
		},
	)
}

func (r *repository) GetCarrierInvoiceByID(
	ctx context.Context,
	req ediports.GetEDICarrierInvoiceByIDRequest,
) (*edicarrierinvoice.CarrierInvoice, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*edicarrierinvoice.CarrierInvoice, error) {
			ci := edicarrierinvoicedb.CarrierInvoices
			entity, err := ci.Where(ci.ID.Eq(req.ID)).Load(ci.Partner()).One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, carrierInvoiceErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) GetCarrierInvoiceByNumber(
	ctx context.Context,
	req *ediports.GetEDICarrierInvoiceByNumberRequest,
) (*edicarrierinvoice.CarrierInvoice, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*edicarrierinvoice.CarrierInvoice, error) {
			ci := edicarrierinvoicedb.CarrierInvoices
			entity, err := ci.Where(
				ci.EDIPartnerID.Eq(req.PartnerID),
				ci.InvoiceNumberKey.Eq(edicarrierinvoice.InvoiceNumberKey(req.InvoiceNumber)),
				ci.DuplicateOfID.IsNull(),
			).One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, carrierInvoiceErrors.Map(err, req.InvoiceNumber)
			}

			return entity, nil
		},
	)
}

func (r *repository) CreateCarrierInvoice(
	ctx context.Context,
	entity *edicarrierinvoice.CarrierInvoice,
) (*edicarrierinvoice.CarrierInvoice, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*edicarrierinvoice.CarrierInvoice, error) {
			if err := edicarrierinvoicedb.CarrierInvoices.Create(ctx, r.db.Keel(ctx), entity); err != nil {
				return nil, carrierInvoiceErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) UpdateCarrierInvoice(
	ctx context.Context,
	entity *edicarrierinvoice.CarrierInvoice,
) (*edicarrierinvoice.CarrierInvoice, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*edicarrierinvoice.CarrierInvoice, error) {
			if err := edicarrierinvoicedb.CarrierInvoices.Save(ctx, r.db.Keel(ctx), entity); err != nil {
				return nil, carrierInvoiceErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}
