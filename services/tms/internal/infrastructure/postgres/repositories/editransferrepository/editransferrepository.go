//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package editransferrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
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
