//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package edishipmenttenderrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/ediscope"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/pkg/dberror"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var shipmentErrors = dberror.Mapping{Entity: "Shipment"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) ediports.EDIShipmentTenderRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-shipment-tender-repository"),
	}
}

func (r *repository) LockShipment(
	ctx context.Context,
	req ediports.LockEDITenderShipmentRequest,
) (*shipment.Shipment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.Shipment, error) {
		sp := shipmentdb.Shipments
		entity, err := ediscope.Tenant(ctx, sp.Where(sp.ID.Eq(req.ShipmentID)), req.TenantInfo).
			Lock(keel.ForUpdate).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, shipmentErrors.Map(err, req.ShipmentID.String())
		}

		return entity, nil
	})
}

func (r *repository) SetShipmentTenderStatus(
	ctx context.Context,
	req ediports.SetEDIShipmentTenderStatusRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		sp := shipmentdb.Shipments
		n, err := ediscope.Tenant(ctx, sp.Where(sp.ID.Eq(req.ShipmentID)), req.TenantInfo).
			Update(ctx, r.db.Keel(ctx), sp.TenderStatus.To(req.Status))
		if err != nil {
			return shipmentErrors.Map(err, req.ShipmentID.String())
		}

		return dberror.CheckFoundCount(n, "Shipment")
	})
}
