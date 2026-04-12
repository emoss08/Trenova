package bankreceiptworkitemrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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

func New(p Params) repositories.BankReceiptWorkItemRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.bank-receipt-work-item-repository")}
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetBankReceiptWorkItemByIDRequest,
) (*bankreceiptworkitem.WorkItem, error) {
	entity := new(bankreceiptworkitem.WorkItem)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("brwi.id = ?", req.ID).
		Where("brwi.organization_id = ?", req.TenantInfo.OrgID).
		Where("brwi.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "BankReceiptWorkItem")
	}
	return entity, nil
}

func (r *repository) GetActiveByReceiptID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	bankReceiptID pulid.ID,
) (*bankreceiptworkitem.WorkItem, error) {
	entity := new(bankreceiptworkitem.WorkItem)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("brwi.organization_id = ?", tenantInfo.OrgID).
		Where("brwi.business_unit_id = ?", tenantInfo.BuID).
		Where("brwi.bank_receipt_id = ?", bankReceiptID).
		Where("brwi.status IN ('Open','Assigned','InReview')").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, errortypes.NewNotFoundError("bank receipt work item not found")
		}
		return nil, err
	}
	return entity, nil
}

func (r *repository) ListActive(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*bankreceiptworkitem.WorkItem, error) {
	items := make([]*bankreceiptworkitem.WorkItem, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("brwi.organization_id = ?", tenantInfo.OrgID).
		Where("brwi.business_unit_id = ?", tenantInfo.BuID).
		Where("brwi.status IN ('Open','Assigned','InReview')").
		Order("brwi.created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active bank receipt work items: %w", err)
	}
	return items, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *bankreceiptworkitem.WorkItem,
) (*bankreceiptworkitem.WorkItem, error) {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create bank receipt work item: %w", err)
	}
	return r.GetByID(
		ctx,
		repositories.GetBankReceiptWorkItemByIDRequest{
			ID: entity.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *bankreceiptworkitem.WorkItem,
) (*bankreceiptworkitem.WorkItem, error) {
	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Where("id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Where("version = ?", entity.Version).
		Set("status = ?", entity.Status).
		Set("assigned_to_user_id = ?", entity.AssignedToUserID).
		Set("assigned_at = ?", entity.AssignedAt).
		Set("resolution_type = ?", entity.ResolutionType).
		Set("resolution_note = ?", entity.ResolutionNote).
		Set("resolved_by_user_id = ?", entity.ResolvedByUserID).
		Set("resolved_at = ?", entity.ResolvedAt).
		Set("updated_by_id = ?", entity.UpdatedByID).
		Set("version = version + 1").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update bank receipt work item: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, "BankReceiptWorkItem", entity.ID.String()); err != nil {
		return nil, err
	}
	return r.GetByID(
		ctx,
		repositories.GetBankReceiptWorkItemByIDRequest{
			ID: entity.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		},
	)
}
