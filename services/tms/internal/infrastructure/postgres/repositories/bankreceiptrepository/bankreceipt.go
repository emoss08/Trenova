package bankreceiptrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
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

type summaryRecord struct {
	ImportedCount         int64 `bun:"imported_count"`
	ImportedAmount        int64 `bun:"imported_amount"`
	MatchedCount          int64 `bun:"matched_count"`
	MatchedAmount         int64 `bun:"matched_amount"`
	ExceptionCount        int64 `bun:"exception_count"`
	ExceptionAmount       int64 `bun:"exception_amount"`
	ActiveWorkItemCount   int64 `bun:"active_work_item_count"`
	AssignedWorkItemCount int64 `bun:"assigned_work_item_count"`
	InReviewWorkItemCount int64 `bun:"in_review_work_item_count"`
	CurrentCount          int64 `bun:"current_count"`
	CurrentAmount         int64 `bun:"current_amount"`
	Days1To3Count         int64 `bun:"days_1_to_3_count"`
	Days1To3Amount        int64 `bun:"days_1_to_3_amount"`
	Days4To7Count         int64 `bun:"days_4_to_7_count"`
	Days4To7Amount        int64 `bun:"days_4_to_7_amount"`
	DaysOver7Count        int64 `bun:"days_over_7_count"`
	DaysOver7Amount       int64 `bun:"days_over_7_amount"`
}

func New(p Params) repositories.BankReceiptRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.bank-receipt-repository")}
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetBankReceiptByIDRequest,
) (*bankreceipt.BankReceipt, error) {
	entity := new(bankreceipt.BankReceipt)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("br.id = ?", req.ID).
		Where("br.organization_id = ?", req.TenantInfo.OrgID).
		Where("br.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "BankReceipt")
	}
	return entity, nil
}

func (r *repository) ListByImportBatchID(
	ctx context.Context,
	req repositories.ListBankReceiptsByImportBatchRequest,
) ([]*bankreceipt.BankReceipt, error) {
	items := make([]*bankreceipt.BankReceipt, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("br.import_batch_id = ?", req.BatchID).
		Where("br.organization_id = ?", req.TenantInfo.OrgID).
		Where("br.business_unit_id = ?", req.TenantInfo.BuID).
		Order("br.receipt_date ASC").
		Order("br.created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list bank receipts by import batch: %w", err)
	}
	return items, nil
}

func (r *repository) ListExceptions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*bankreceipt.BankReceipt, error) {
	items := make([]*bankreceipt.BankReceipt, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("br.organization_id = ?", tenantInfo.OrgID).
		Where("br.business_unit_id = ?", tenantInfo.BuID).
		Where("br.status = ?", bankreceipt.StatusException).
		Order("br.receipt_date DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list bank receipt exceptions: %w", err)
	}
	return items, nil
}

func (r *repository) GetSummary(
	ctx context.Context,
	req repositories.GetBankReceiptSummaryRequest,
) (*repositories.BankReceiptReconciliationSummary, error) {
	rec := new(summaryRecord)
	err := r.db.DBForContext(ctx).NewRaw(`
		SELECT
			COALESCE(SUM(CASE WHEN br.status = 'Imported' THEN 1 ELSE 0 END), 0) AS imported_count,
			COALESCE(SUM(CASE WHEN br.status = 'Imported' THEN br.amount_minor ELSE 0 END), 0) AS imported_amount,
			COALESCE(SUM(CASE WHEN br.status = 'Matched' THEN 1 ELSE 0 END), 0) AS matched_count,
			COALESCE(SUM(CASE WHEN br.status = 'Matched' THEN br.amount_minor ELSE 0 END), 0) AS matched_amount,
			COALESCE(SUM(CASE WHEN br.status = 'Exception' THEN 1 ELSE 0 END), 0) AS exception_count,
			COALESCE(SUM(CASE WHEN br.status = 'Exception' THEN br.amount_minor ELSE 0 END), 0) AS exception_amount,
			COALESCE((SELECT COUNT(*) FROM bank_receipt_work_items wi WHERE wi.organization_id = ? AND wi.business_unit_id = ? AND wi.status IN ('Open','Assigned','InReview')), 0) AS active_work_item_count,
			COALESCE((SELECT COUNT(*) FROM bank_receipt_work_items wi WHERE wi.organization_id = ? AND wi.business_unit_id = ? AND wi.status = 'Assigned'), 0) AS assigned_work_item_count,
			COALESCE((SELECT COUNT(*) FROM bank_receipt_work_items wi WHERE wi.organization_id = ? AND wi.business_unit_id = ? AND wi.status = 'InReview'), 0) AS in_review_work_item_count,
			COALESCE(SUM(CASE WHEN br.status = 'Exception' AND (? - br.receipt_date) / 86400 <= 0 THEN 1 ELSE 0 END), 0) AS current_count,
			COALESCE(SUM(CASE WHEN br.status = 'Exception' AND (? - br.receipt_date) / 86400 <= 0 THEN br.amount_minor ELSE 0 END), 0) AS current_amount,
			COALESCE(SUM(CASE WHEN br.status = 'Exception' AND (? - br.receipt_date) / 86400 BETWEEN 1 AND 3 THEN 1 ELSE 0 END), 0) AS days_1_to_3_count,
			COALESCE(SUM(CASE WHEN br.status = 'Exception' AND (? - br.receipt_date) / 86400 BETWEEN 1 AND 3 THEN br.amount_minor ELSE 0 END), 0) AS days_1_to_3_amount,
			COALESCE(SUM(CASE WHEN br.status = 'Exception' AND (? - br.receipt_date) / 86400 BETWEEN 4 AND 7 THEN 1 ELSE 0 END), 0) AS days_4_to_7_count,
			COALESCE(SUM(CASE WHEN br.status = 'Exception' AND (? - br.receipt_date) / 86400 BETWEEN 4 AND 7 THEN br.amount_minor ELSE 0 END), 0) AS days_4_to_7_amount,
			COALESCE(SUM(CASE WHEN br.status = 'Exception' AND (? - br.receipt_date) / 86400 > 7 THEN 1 ELSE 0 END), 0) AS days_over_7_count,
			COALESCE(SUM(CASE WHEN br.status = 'Exception' AND (? - br.receipt_date) / 86400 > 7 THEN br.amount_minor ELSE 0 END), 0) AS days_over_7_amount
		FROM bank_receipts br
		WHERE br.organization_id = ?
		  AND br.business_unit_id = ?
		  AND br.receipt_date <= ?
	`, req.TenantInfo.OrgID, req.TenantInfo.BuID, req.TenantInfo.OrgID, req.TenantInfo.BuID, req.TenantInfo.OrgID, req.TenantInfo.BuID, req.AsOfDate, req.AsOfDate, req.AsOfDate, req.AsOfDate, req.AsOfDate, req.AsOfDate, req.AsOfDate, req.AsOfDate, req.TenantInfo.OrgID, req.TenantInfo.BuID, req.AsOfDate).Scan(ctx, rec)
	if err != nil {
		return nil, fmt.Errorf("get bank receipt summary: %w", err)
	}
	return &repositories.BankReceiptReconciliationSummary{
		AsOfDate:              req.AsOfDate,
		ImportedCount:         rec.ImportedCount,
		ImportedAmount:        rec.ImportedAmount,
		MatchedCount:          rec.MatchedCount,
		MatchedAmount:         rec.MatchedAmount,
		ExceptionCount:        rec.ExceptionCount,
		ExceptionAmount:       rec.ExceptionAmount,
		ActiveWorkItemCount:   rec.ActiveWorkItemCount,
		AssignedWorkItemCount: rec.AssignedWorkItemCount,
		InReviewWorkItemCount: rec.InReviewWorkItemCount,
		ExceptionAging: repositories.BankReceiptExceptionAging{
			CurrentCount:    rec.CurrentCount,
			CurrentAmount:   rec.CurrentAmount,
			Days1To3Count:   rec.Days1To3Count,
			Days1To3Amount:  rec.Days1To3Amount,
			Days4To7Count:   rec.Days4To7Count,
			Days4To7Amount:  rec.Days4To7Amount,
			DaysOver7Count:  rec.DaysOver7Count,
			DaysOver7Amount: rec.DaysOver7Amount,
		},
	}, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *bankreceipt.BankReceipt,
) (*bankreceipt.BankReceipt, error) {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create bank receipt: %w", err)
	}
	return r.GetByID(
		ctx,
		repositories.GetBankReceiptByIDRequest{
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
	entity *bankreceipt.BankReceipt,
) (*bankreceipt.BankReceipt, error) {
	entity.UpdatedAt = timeutils.NowUnix()
	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Where("id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Where("version = ?", entity.Version).
		Set("status = ?", entity.Status).
		Set("matched_customer_payment_id = ?", entity.MatchedCustomerPaymentID).
		Set("matched_at = ?", entity.MatchedAt).
		Set("matched_by_id = ?", entity.MatchedByID).
		Set("exception_reason = ?", entity.ExceptionReason).
		Set("updated_by_id = ?", entity.UpdatedByID).
		Set("updated_at = ?", entity.UpdatedAt).
		Set("version = version + 1").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update bank receipt: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, "BankReceipt", entity.ID.String()); err != nil {
		return nil, err
	}
	return r.GetByID(
		ctx,
		repositories.GetBankReceiptByIDRequest{
			ID: entity.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		},
	)
}
