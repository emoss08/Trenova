package edirepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
)

func (r *repository) ListInbound(
	ctx context.Context,
	req *repositories.ListEDITransfersRequest,
) (*pagination.ListResult[*edi.EDITransfer], error) {
	entities := make([]*edi.EDITransfer, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where("eltt.target_organization_id = ?", req.Filter.TenantInfo.OrgID).
		Where("eltt.target_business_unit_id = ?", req.Filter.TenantInfo.BuID).
		Order("eltt.created_at DESC").
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
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where("eltt.source_organization_id = ?", req.Filter.TenantInfo.OrgID).
		Where("eltt.source_business_unit_id = ?", req.Filter.TenantInfo.BuID).
		Order("eltt.created_at DESC").
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
	query := r.db.DBForContext(ctx).NewSelect().Model(entity).Where("eltt.id = ?", req.ID)

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
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("eltt.id = ?", req.ID).
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
	switch direction {
	case "inbound":
		return query.
			Where("eltt.target_organization_id = ?", tenantInfo.OrgID).
			Where("eltt.target_business_unit_id = ?", tenantInfo.BuID)
	case "outbound":
		return query.
			Where("eltt.source_organization_id = ?", tenantInfo.OrgID).
			Where("eltt.source_business_unit_id = ?", tenantInfo.BuID)
	default:
		return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.WhereOr(
				"(eltt.target_organization_id = ? AND eltt.target_business_unit_id = ?)",
				tenantInfo.OrgID,
				tenantInfo.BuID,
			).WhereOr(
				"(eltt.source_organization_id = ? AND eltt.source_business_unit_id = ?)",
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
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
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
	if err = dberror.CheckRowsAffected(results, "EDITenderTransfer", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) SetApprovalWorkflowRunID(
	ctx context.Context,
	req repositories.SetEDITransferApprovalWorkflowRunIDRequest,
) (*edi.EDITransfer, error) {
	entity := new(edi.EDITransfer)
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Set("approval_workflow_run_id = ?", req.RunID).
		Set("updated_at = extract(epoch from current_timestamp)::bigint").
		Where("id = ?", req.ID).
		Where("target_organization_id = ?", req.TenantInfo.OrgID).
		Where("target_business_unit_id = ?", req.TenantInfo.BuID).
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
