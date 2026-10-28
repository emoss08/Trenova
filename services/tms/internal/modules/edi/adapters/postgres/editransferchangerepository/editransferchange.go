package editransferchangerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
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

var transferChangeErrors = dberror.Mapping{Entity: "EDITransferChange"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) editenderports.EDITransferChangeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-transfer-change-repository"),
	}
}

func (r *repository) ListTransferChanges(
	ctx context.Context,
	req *editenderports.ListEDITransferChangesRequest,
) (*pagination.ListResult[*edisync.TransferChange], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edisync.TransferChange], error) {
			tc := edisyncdb.TransferChanges
			q := tc.Where(
				linkedToOrganization(req.Filter.TenantInfo.OrgID),
				keel.When(req.ShipmentLinkID.IsNotNil(), tc.ShipmentLinkID.Eq(req.ShipmentLinkID)),
			)

			return listquery.List(
				ctx,
				r.db.Keel(ctx),
				q,
				&listquery.Request[edisync.TransferChange]{
					Filter: req.Filter,
					Then:   []keel.Order{tc.CreatedAt.Desc()},
				},
			)
		},
	)
}

func (r *repository) GetTransferChangeByID(
	ctx context.Context,
	req editenderports.GetEDITransferChangeByIDRequest,
) (*edisync.TransferChange, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edisync.TransferChange, error) {
		tc := edisyncdb.TransferChanges
		entity, err := tc.Where(tc.ID.Eq(req.ID), linkedToOrganization(req.TenantInfo.OrgID)).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, transferChangeErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CreateTransferChange(
	ctx context.Context,
	entity *edisync.TransferChange,
) (*edisync.TransferChange, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edisync.TransferChange, error) {
		if err := edisyncdb.TransferChanges.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, transferChangeErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CreateTransferChangeIdempotent(
	ctx context.Context,
	entity *edisync.TransferChange,
) (*editenderports.CreateEDITransferChangeIdempotentResult, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*editenderports.CreateEDITransferChangeIdempotentResult, error) {
			db := r.db.Keel(ctx)
			tc := edisyncdb.TransferChanges
			created, err := tc.CreateOrSkip(
				ctx,
				db,
				[]*edisync.TransferChange{entity},
				tc.ShipmentLinkID,
				tc.BusinessUnitID,
				tc.Direction,
				tc.ChangeType,
				tc.IdempotencyKey,
			)
			if err != nil {
				return nil, transferChangeErrors.Map(err, entity.ID.String())
			}
			if created > 0 {
				return &editenderports.CreateEDITransferChangeIdempotentResult{
					TransferChange: entity,
					Created:        true,
				}, nil
			}

			existing, err := tc.Where(
				tc.ShipmentLinkID.Eq(entity.ShipmentLinkID),
				tc.Direction.Eq(entity.Direction),
				tc.ChangeType.Eq(entity.ChangeType),
				tc.IdempotencyKey.Eq(entity.IdempotencyKey),
			).One(ctx, db)
			if err != nil {
				return nil, transferChangeErrors.Map(err, entity.ID.String())
			}

			return &editenderports.CreateEDITransferChangeIdempotentResult{
				TransferChange: existing,
				Created:        false,
			}, nil
		},
	)
}

func (r *repository) UpdateTransferChange(
	ctx context.Context,
	entity *edisync.TransferChange,
) (*edisync.TransferChange, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edisync.TransferChange, error) {
		if err := edisyncdb.TransferChanges.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, transferChangeErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func linkedToOrganization(orgID pulid.ID) keel.Pred {
	tc := edisyncdb.TransferChanges
	sl := edisyncdb.ShipmentLinks
	return keel.Exists(sl.Where(
		sl.ID.EqCol(tc.ShipmentLinkID),
		keel.Or(sl.SourceOrganizationID.Eq(orgID), sl.TargetOrganizationID.Eq(orgID)),
	))
}
