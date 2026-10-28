package edishipmentlinkrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/ediscope"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edisync"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edisync/edisyncdb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var shipmentLinkErrors = dberror.Mapping{Entity: "EDIShipmentLink"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) editenderports.EDIShipmentLinkRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-shipment-link-repository"),
	}
}

func (r *repository) ListShipmentLinks(
	ctx context.Context,
	req *editenderports.ListEDIShipmentLinksRequest,
) (*pagination.ListResult[*edisync.ShipmentLink], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edisync.ShipmentLink], error) {
			sl := edisyncdb.ShipmentLinks
			return listquery.List(
				ctx,
				r.db.Keel(ctx),
				sl.Where(involvesOrganization(req.Filter.TenantInfo.OrgID)),
				&listquery.Request[edisync.ShipmentLink]{
					Filter: req.Filter,
					Then:   []keel.Order{sl.CreatedAt.Desc()},
				},
			)
		},
	)
}

func (r *repository) GetShipmentLinkByID(
	ctx context.Context,
	req editenderports.GetEDIShipmentLinkByIDRequest,
) (*edisync.ShipmentLink, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edisync.ShipmentLink, error) {
		sl := edisyncdb.ShipmentLinks
		q := sl.Where(sl.ID.Eq(req.ID), involvesOrganization(req.TenantInfo.OrgID))
		entity, err := ediscope.Tenant(ctx, q, req.TenantInfo).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, shipmentLinkErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetShipmentLinksByShipmentID(
	ctx context.Context,
	req editenderports.GetEDIShipmentLinksByShipmentIDRequest,
) ([]*edisync.ShipmentLink, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*edisync.ShipmentLink, error) {
		sl := edisyncdb.ShipmentLinks
		org := req.TenantInfo.OrgID
		return sl.Where(keel.Or(
			keel.And(sl.SourceOrganizationID.Eq(org), sl.SourceShipmentID.Eq(req.ShipmentID)),
			keel.And(sl.TargetOrganizationID.Eq(org), sl.TargetShipmentID.Eq(req.ShipmentID)),
		)).All(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) CreateShipmentLink(
	ctx context.Context,
	entity *edisync.ShipmentLink,
) (*edisync.ShipmentLink, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edisync.ShipmentLink, error) {
		if err := edisyncdb.ShipmentLinks.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, shipmentLinkErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func involvesOrganization(orgID pulid.ID) keel.Pred {
	sl := edisyncdb.ShipmentLinks
	return keel.Or(sl.SourceOrganizationID.Eq(orgID), sl.TargetOrganizationID.Eq(orgID))
}
