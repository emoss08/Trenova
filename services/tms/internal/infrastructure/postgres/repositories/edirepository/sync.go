package edirepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
)

func (r *repository) ListShipmentLinks(
	ctx context.Context,
	req *repositories.ListEDIShipmentLinksRequest,
) (*pagination.ListResult[*edi.ShipmentLink], error) {
	entities := make([]*edi.ShipmentLink, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(func(query *bun.SelectQuery) *bun.SelectQuery {
			return querybuilder.ApplyFilters(query, "esl", req.Filter, (*edi.ShipmentLink)(nil)).
				Where("esl.business_unit_id = ?", req.Filter.TenantInfo.BuID).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return sq.WhereOr("esl.source_organization_id = ?", req.Filter.TenantInfo.OrgID).
						WhereOr("esl.target_organization_id = ?", req.Filter.TenantInfo.OrgID)
				})
		}).
		Order("esl.created_at DESC").
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
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("esl.id = ?", req.ID).
		Where("esl.business_unit_id = ?", req.TenantInfo.BuID).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.WhereOr("esl.source_organization_id = ?", req.TenantInfo.OrgID).
				WhereOr("esl.target_organization_id = ?", req.TenantInfo.OrgID)
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
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where("esl.business_unit_id = ?", req.TenantInfo.BuID).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.WhereOr(
				"(esl.source_organization_id = ? AND esl.source_shipment_id = ?)",
				req.TenantInfo.OrgID,
				req.ShipmentID,
			).WhereOr(
				"(esl.target_organization_id = ? AND esl.target_shipment_id = ?)",
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

func (r *repository) ListTransferChanges(
	ctx context.Context,
	req *repositories.ListEDITransferChangesRequest,
) (*pagination.ListResult[*edi.TransferChange], error) {
	entities := make([]*edi.TransferChange, 0, req.Filter.Pagination.SafeLimit())
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Join("JOIN edi_shipment_links AS esl ON esl.id = etc.shipment_link_id").
		Where("etc.business_unit_id = ?", req.Filter.TenantInfo.BuID).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.WhereOr("esl.source_organization_id = ?", req.Filter.TenantInfo.OrgID).
				WhereOr("esl.target_organization_id = ?", req.Filter.TenantInfo.OrgID)
		})
	if req.ShipmentLinkID.IsNotNil() {
		query = query.Where("etc.shipment_link_id = ?", req.ShipmentLinkID)
	}
	total, err := querybuilder.ApplyFilters(query, "etc", req.Filter, (*edi.TransferChange)(nil)).
		Order("etc.created_at DESC").
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*edi.TransferChange]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetTransferChangeByID(
	ctx context.Context,
	req repositories.GetEDITransferChangeByIDRequest,
) (*edi.TransferChange, error) {
	entity := new(edi.TransferChange)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Join("JOIN edi_shipment_links AS esl ON esl.id = etc.shipment_link_id").
		Where("etc.id = ?", req.ID).
		Where("etc.business_unit_id = ?", req.TenantInfo.BuID).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.WhereOr("esl.source_organization_id = ?", req.TenantInfo.OrgID).
				WhereOr("esl.target_organization_id = ?", req.TenantInfo.OrgID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDITransferChange")
	}

	return entity, nil
}

func (r *repository) CreateTransferChange(
	ctx context.Context,
	entity *edi.TransferChange,
) (*edi.TransferChange, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) UpdateTransferChange(
	ctx context.Context,
	entity *edi.TransferChange,
) (*edi.TransferChange, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(
		results,
		"EDITransferChange",
		entity.ID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}
