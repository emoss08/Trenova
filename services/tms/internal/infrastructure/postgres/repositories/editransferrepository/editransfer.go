//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package editransferrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
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

func New(p Params) repositories.EDILoadTenderTransferRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-transfer-repository"),
	}
}

func (r *repository) GetInboundStatusCounts(
	ctx context.Context,
	req repositories.GetEDITransferStatusCountsRequest,
) (map[edi.TransferStatus]int, error) {
	cols := buncolgen.EDITransferColumns
	var rows []struct {
		Status edi.TransferStatus `bun:"status"`
		Count  int                `bun:"count"`
	}
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model((*edi.EDITransfer)(nil)).
		ColumnExpr(cols.Status.Qualified()).
		ColumnExpr("COUNT(*) AS count").
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyTransferTenantFilter(sq, req.TenantInfo, "inbound")
		}).
		GroupExpr(cols.Status.Qualified())
	if req.Since > 0 {
		query = query.Where(cols.SubmittedAt.Gte(), req.Since)
	}
	if err := query.Scan(ctx, &rows); err != nil {
		return nil, err
	}
	counts := make(map[edi.TransferStatus]int, len(rows))
	for _, row := range rows {
		counts[row.Status] = row.Count
	}
	return counts, nil
}

func (r *repository) ListInbound(
	ctx context.Context,
	req *repositories.ListEDITransfersRequest,
) (*pagination.ListResult[*edi.EDITransfer], error) {
	entities := make([]*edi.EDITransfer, 0, req.Filter.Pagination.SafeLimit())
	cols := buncolgen.EDITransferColumns

	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where(cols.TargetOrganizationID.Eq(), req.Filter.TenantInfo.OrgID).
		Where(cols.TargetBusinessUnitID.Eq(), req.Filter.TenantInfo.BuID).
		Order(cols.CreatedAt.OrderDesc()).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*edi.EDITransfer]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) ListOutbound(
	ctx context.Context,
	req *repositories.ListEDITransfersRequest,
) (*pagination.ListResult[*edi.EDITransfer], error) {
	entities := make([]*edi.EDITransfer, 0, req.Filter.Pagination.SafeLimit())
	cols := buncolgen.EDITransferColumns

	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where(cols.SourceOrganizationID.Eq(), req.Filter.TenantInfo.OrgID).
		Where(cols.SourceBusinessUnitID.Eq(), req.Filter.TenantInfo.BuID).
		Where("eltt.inbound_message_id IS NULL").
		Order(cols.CreatedAt.OrderDesc()).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*edi.EDITransfer]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetTransferByID(
	ctx context.Context,
	req repositories.GetEDITransferByIDRequest,
) (*edi.EDITransfer, error) {
	entity := new(edi.EDITransfer)
	cols := buncolgen.EDITransferColumns
	query := r.db.DBForContext(ctx).NewSelect().Model(entity).Where(cols.ID.Eq(), req.ID)

	query = applyTransferTenantFilter(query, req.TenantInfo, req.Direction)

	if err := query.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDITenderTransfer")
	}

	return entity, nil
}

func (r *repository) GetTransfersByIDs(
	ctx context.Context,
	req repositories.GetEDITransfersByIDsRequest,
) ([]*edi.EDITransfer, error) {
	entities := make([]*edi.EDITransfer, 0, len(req.TransferIDs))
	cols := buncolgen.EDITransferColumns
	rel := buncolgen.EDITransferRelations

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(rel.SourcePartner).
		Relation(rel.TargetPartner).
		Where(cols.ID.In(), bun.List(req.TransferIDs)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyTransferTenantFilter(sq, req.TenantInfo, "")
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDITenderTransfer")
	}

	return entities, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *repositories.EDITransferSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDITransfer], error) {
	entities := make([]*edi.EDITransfer, 0, req.SelectQueryRequest.Pagination.SafeLimit())
	cols := buncolgen.EDITransferColumns
	rel := buncolgen.EDITransferRelations

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(rel.SourcePartner).
		Relation(rel.TargetPartner).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyTransferTenantFilter(sq, req.SelectQueryRequest.TenantInfo, "")
		}).
		Order(cols.CreatedAt.OrderDesc()).
		Limit(req.SelectQueryRequest.Pagination.SafeLimit()).
		Offset(req.SelectQueryRequest.Pagination.SafeOffset())

	if req.SelectQueryRequest.Query != "" {
		query = query.Where(
			"eltt.tender_payload->>'bol' ILIKE ?",
			dbhelper.WrapWildcard(req.SelectQueryRequest.Query),
		)
	}

	total, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*edi.EDITransfer]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetTransferForUpdate(
	ctx context.Context,
	req repositories.GetEDITransferForUpdateRequest,
) (*edi.EDITransfer, error) {
	entity := new(edi.EDITransfer)
	cols := buncolgen.EDITransferColumns
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(cols.ID.Eq(), req.ID).
		For("UPDATE")

	query = applyTransferTenantFilter(query, req.TenantInfo, req.Direction)

	if err := query.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDITenderTransfer")
	}

	return entity, nil
}

func applyTransferTenantFilter(
	query *bun.SelectQuery,
	tenantInfo pagination.TenantInfo,
	direction string,
) *bun.SelectQuery {
	cols := buncolgen.EDITransferColumns

	switch direction {
	case "inbound":
		return query.
			Where(cols.TargetOrganizationID.Eq(), tenantInfo.OrgID).
			Where(cols.TargetBusinessUnitID.Eq(), tenantInfo.BuID)
	case "outbound":
		return query.
			Where(cols.SourceOrganizationID.Eq(), tenantInfo.OrgID).
			Where(cols.SourceBusinessUnitID.Eq(), tenantInfo.BuID)
	default:
		return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.WhereOr(
				buncolgen.Expr("({0} = ? AND {1} = ?)", cols.TargetOrganizationID, cols.TargetBusinessUnitID),
				tenantInfo.OrgID,
				tenantInfo.BuID,
			).
				WhereOr(
					buncolgen.Expr("({0} = ? AND {1} = ?)", cols.SourceOrganizationID, cols.SourceBusinessUnitID),
					tenantInfo.OrgID,
					tenantInfo.BuID,
				)
		})
	}
}

func (r *repository) GetActionableInboundTransferByExternalReference(
	ctx context.Context,
	req repositories.GetActionableInboundEDITransferByExternalReferenceRequest,
) (*edi.EDITransfer, error) {
	entity := new(edi.EDITransfer)
	cols := buncolgen.EDITransferColumns

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(cols.TargetOrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(cols.TargetBusinessUnitID.Eq(), req.TenantInfo.BuID).
		Where(cols.TargetPartnerID.Eq(), req.PartnerID).
		Where("eltt.inbound_message_id IS NOT NULL").
		Where(
			"eltt.tender_payload->'ratingDetail'->>'externalShipmentId' = ?",
			req.ExternalReference,
		).
		Where(cols.Status.In(), bun.List([]edi.TransferStatus{
			edi.TransferStatusSubmitted,
			edi.TransferStatusMappingRequired,
			edi.TransferStatusPendingApproval,
		})).
		Order(cols.CreatedAt.OrderDesc()).
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDITenderTransfer")
	}
	return entity, nil
}

func (r *repository) ListActionableInboundTransfersByPartner(
	ctx context.Context,
	req repositories.ListActionableInboundEDITransfersByPartnerRequest,
) ([]*edi.EDITransfer, error) {
	entities := make([]*edi.EDITransfer, 0)
	cols := buncolgen.EDITransferColumns

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where(cols.TargetOrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(cols.TargetBusinessUnitID.Eq(), req.TenantInfo.BuID).
		Where(cols.TargetPartnerID.Eq(), req.PartnerID).
		Where(cols.Status.In(), bun.List(req.Statuses)).
		Order(cols.CreatedAt.OrderAsc())
	if len(req.ExcludeIDs) > 0 {
		query = query.Where(cols.ID.NotIn(), bun.List(req.ExcludeIDs))
	}

	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *repository) CreateTransfer(
	ctx context.Context,
	entity *edi.EDITransfer,
) (*edi.EDITransfer, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) UpdateTransfer(
	ctx context.Context,
	entity *edi.EDITransfer,
) (*edi.EDITransfer, error) {
	ov := entity.Version
	entity.Version++
	cols := buncolgen.EDITransferColumns

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(cols.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(
		results,
		"EDITenderTransfer",
		entity.ID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) SetApprovalWorkflowRunID(
	ctx context.Context,
	req repositories.SetEDITransferApprovalWorkflowRunIDRequest,
) (*edi.EDITransfer, error) {
	entity := new(edi.EDITransfer)
	cols := buncolgen.EDITransferColumns
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Set(cols.ApprovalWorkflowRunID.Set(), req.RunID).
		Set(cols.UpdatedAt.SetExpr("extract(epoch from current_timestamp)::bigint")).
		Where(cols.ID.Eq(), req.ID).
		Where(cols.TargetOrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(cols.TargetBusinessUnitID.Eq(), req.TenantInfo.BuID).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "EDITenderTransfer", req.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) ListInboundCursor(
	ctx context.Context,
	req *repositories.ListEDITransfersRequest,
) (*pagination.CursorListResult[*edi.EDITransfer], error) {
	return r.listCursor(ctx, req, "inbound")
}

func (r *repository) ListOutboundCursor(
	ctx context.Context,
	req *repositories.ListEDITransfersRequest,
) (*pagination.CursorListResult[*edi.EDITransfer], error) {
	return r.listCursor(ctx, req, "outbound")
}

func (r *repository) listCursor(
	ctx context.Context,
	req *repositories.ListEDITransfersRequest,
	direction string,
) (*pagination.CursorListResult[*edi.EDITransfer], error) {
	dba := r.db.DBForContext(ctx)
	scope := func(sq *bun.SelectQuery) *bun.SelectQuery {
		sq = applyTransferTenantFilter(sq, req.Filter.TenantInfo, direction)
		if direction == "outbound" {
			sq = sq.Where("eltt.inbound_message_id IS NULL")
		}
		return sq
	}

	total, err := dba.
		NewSelect().
		Model((*edi.EDITransfer)(nil)).
		Apply(scope).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	return dbhelper.CursorList(ctx, dbhelper.CursorListParams[*edi.EDITransfer]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: &total,
		Query: func(entities *[]*edi.EDITransfer) *bun.SelectQuery {
			rel := buncolgen.EDITransferRelations
			return dba.
				NewSelect().
				Model(entities).
				ColumnExpr(buncolgen.EDITransferTable.All()).
				Relation(rel.SourcePartner).
				Relation(rel.TargetPartner)
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			sq, applyErr := querybuilder.ApplyCursorFiltersWithoutTenantScope(
				sq,
				"eltt",
				req.Filter,
				req.Cursor,
				(*edi.EDITransfer)(nil),
			)
			if applyErr != nil {
				return sq, applyErr
			}
			return scope(sq), nil
		},
	})
}
