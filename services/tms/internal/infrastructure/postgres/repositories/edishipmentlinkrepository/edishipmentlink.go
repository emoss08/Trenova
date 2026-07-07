package edishipmentlinkrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
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

func New(p Params) repositories.EDIShipmentLinkRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-shipment-link-repository"),
	}
}

func (r *repository) ListShipmentLinks(
	ctx context.Context,
	req *repositories.ListEDIShipmentLinksRequest,
) (*pagination.ListResult[*edi.ShipmentLink], error) {
	entities := make([]*edi.ShipmentLink, 0, req.Filter.Pagination.SafeLimit())
	cols := buncolgen.ShipmentLinkColumns

	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(func(query *bun.SelectQuery) *bun.SelectQuery {
			return querybuilder.ApplyFiltersWithoutTenantScope(query, "esl", req.Filter, (*edi.ShipmentLink)(nil)).
				Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
					return applyShipmentLinkTenantScope(sq, req.Filter.TenantInfo)
				})
		}).
		Order(cols.CreatedAt.OrderDesc()).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*edi.ShipmentLink]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetShipmentLinkByID(
	ctx context.Context,
	req repositories.GetEDIShipmentLinkByIDRequest,
) (*edi.ShipmentLink, error) {
	entity := new(edi.ShipmentLink)
	cols := buncolgen.ShipmentLinkColumns

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(cols.ID.Eq(), req.ID).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyShipmentLinkTenantScope(sq, req.TenantInfo)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDIShipmentLink")
	}

	return entity, nil
}

func (r *repository) GetShipmentLinksByShipmentID(
	ctx context.Context,
	req repositories.GetEDIShipmentLinksByShipmentIDRequest,
) ([]*edi.ShipmentLink, error) {
	entities := make([]*edi.ShipmentLink, 0)
	cols := buncolgen.ShipmentLinkColumns

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.WhereOr(
				buncolgen.Expr("({0} = ? AND {1} = ?)", cols.SourceOrganizationID, cols.SourceShipmentID),
				req.TenantInfo.OrgID,
				req.ShipmentID,
			).
				WhereOr(
					buncolgen.Expr("({0} = ? AND {1} = ?)", cols.TargetOrganizationID, cols.TargetShipmentID),
					req.TenantInfo.OrgID,
					req.ShipmentID,
				)
		}).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *repository) CreateShipmentLink(
	ctx context.Context,
	entity *edi.ShipmentLink,
) (*edi.ShipmentLink, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		return nil, err
	}

	return entity, nil
}

func applyShipmentLinkTenantScope(
	query *bun.SelectQuery,
	tenantInfo pagination.TenantInfo,
) *bun.SelectQuery {
	cols := buncolgen.ShipmentLinkColumns

	return query.
		Where(cols.BusinessUnitID.Eq(), tenantInfo.BuID).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.WhereOr(cols.SourceOrganizationID.Eq(), tenantInfo.OrgID).
				WhereOr(cols.TargetOrganizationID.Eq(), tenantInfo.OrgID)
		})
}
