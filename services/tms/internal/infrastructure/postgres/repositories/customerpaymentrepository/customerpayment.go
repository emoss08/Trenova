package customerpaymentrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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

func New(p Params) repositories.CustomerPaymentRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.customer-payment-repository")}
}

func (r *repository) GetByID(ctx context.Context, req repositories.GetCustomerPaymentByIDRequest) (*customerpayment.Payment, error) {
	entity := new(customerpayment.Payment)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("cp.id = ?", req.ID).
		Where("cp.organization_id = ?", req.TenantInfo.OrgID).
		Where("cp.business_unit_id = ?", req.TenantInfo.BuID).
		Relation("Applications", func(q *bun.SelectQuery) *bun.SelectQuery { return q.Order("cpa.line_number ASC") }).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "CustomerPayment")
	}
	return entity, nil
}

func (r *repository) FindMatchCandidates(ctx context.Context, req repositories.FindCustomerPaymentMatchCandidatesRequest) ([]*customerpayment.Payment, error) {
	items := make([]*customerpayment.Payment, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("cp.organization_id = ?", req.TenantInfo.OrgID).
		Where("cp.business_unit_id = ?", req.TenantInfo.BuID).
		Where("cp.status = ?", customerpayment.StatusPosted).
		Where("cp.reference_number = ?", req.ReferenceNumber).
		Where("cp.amount_minor = ?", req.AmountMinor).
		Where("NOT EXISTS (SELECT 1 FROM bank_receipts br WHERE br.organization_id = cp.organization_id AND br.business_unit_id = cp.business_unit_id AND br.matched_customer_payment_id = cp.id AND br.status = 'Matched')").
		Order("cp.created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("find customer payment match candidates: %w", err)
	}
	return items, nil
}

func (r *repository) FindSuggestedMatchCandidates(ctx context.Context, req repositories.FindCustomerPaymentMatchCandidatesRequest) ([]*customerpayment.Payment, error) {
	items := make([]*customerpayment.Payment, 0)
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("cp.organization_id = ?", req.TenantInfo.OrgID).
		Where("cp.business_unit_id = ?", req.TenantInfo.BuID).
		Where("cp.status = ?", customerpayment.StatusPosted).
		Where("NOT EXISTS (SELECT 1 FROM bank_receipts br WHERE br.organization_id = cp.organization_id AND br.business_unit_id = cp.business_unit_id AND br.matched_customer_payment_id = cp.id AND br.status = 'Matched')")
	if req.ReferenceNumber != "" {
		query = query.Where("cp.reference_number = ? OR cp.reference_number ILIKE ?", req.ReferenceNumber, "%"+req.ReferenceNumber+"%")
	}
	if req.AmountMinor > 0 {
		query = query.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("cp.amount_minor = ?", req.AmountMinor).WhereOr("ABS(cp.amount_minor - ?) <= 100", req.AmountMinor)
		})
	}
	if req.ReceiptDate > 0 {
		query = query.Where("ABS(cp.payment_date - ?) <= 604800", req.ReceiptDate)
	}
	err := query.Order("cp.created_at DESC").Limit(10).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("find suggested customer payment matches: %w", err)
	}
	return items, nil
}

func (r *repository) Create(ctx context.Context, entity *customerpayment.Payment) (*customerpayment.Payment, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("cpay_")
	}
	entity.SyncAmounts()
	assignApplicationFields(entity)
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create customer payment: %w", err)
	}
	if len(entity.Applications) > 0 {
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(&entity.Applications).Exec(ctx); err != nil {
			return nil, fmt.Errorf("create customer payment applications: %w", err)
		}
	}
	return r.GetByID(ctx, repositories.GetCustomerPaymentByIDRequest{ID: entity.ID, TenantInfo: pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}})
}

func (r *repository) Update(ctx context.Context, entity *customerpayment.Payment) (*customerpayment.Payment, error) {
	entity.SyncAmounts()
	assignApplicationFields(entity)
	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Where("id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Where("version = ?", entity.Version).
		Set("applied_amount_minor = ?", entity.AppliedAmountMinor).
		Set("unapplied_amount_minor = ?", entity.UnappliedAmountMinor).
		Set("status = ?", entity.Status).
		Set("posted_batch_id = ?", entity.PostedBatchID).
		Set("reversal_batch_id = ?", entity.ReversalBatchID).
		Set("reversed_by_id = ?", entity.ReversedByID).
		Set("reversed_at = ?", entity.ReversedAt).
		Set("reversal_reason = ?", entity.ReversalReason).
		Set("updated_by_id = ?", entity.UpdatedByID).
		Set("version = version + 1").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update customer payment: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, "CustomerPayment", entity.ID.String()); err != nil {
		return nil, err
	}
	if entity.Applications != nil {
		if _, err = r.db.DBForContext(ctx).
			NewDelete().
			Model((*customerpayment.Application)(nil)).
			Where("customer_payment_id = ?", entity.ID).
			Where("organization_id = ?", entity.OrganizationID).
			Where("business_unit_id = ?", entity.BusinessUnitID).
			Exec(ctx); err != nil {
			return nil, fmt.Errorf("replace customer payment applications: %w", err)
		}
		if len(entity.Applications) > 0 {
			if _, err = r.db.DBForContext(ctx).NewInsert().Model(&entity.Applications).Exec(ctx); err != nil {
				return nil, fmt.Errorf("insert customer payment applications: %w", err)
			}
		}
	}
	return r.GetByID(ctx, repositories.GetCustomerPaymentByIDRequest{ID: entity.ID, TenantInfo: pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}})
}

func assignApplicationFields(entity *customerpayment.Payment) {
	for idx, app := range entity.Applications {
		if app == nil {
			continue
		}
		app.OrganizationID = entity.OrganizationID
		app.BusinessUnitID = entity.BusinessUnitID
		app.CustomerPaymentID = entity.ID
		app.LineNumber = idx + 1
	}
}
