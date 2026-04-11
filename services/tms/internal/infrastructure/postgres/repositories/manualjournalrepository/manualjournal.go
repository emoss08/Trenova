package manualjournalrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/manualjournal"
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

func New(p Params) repositories.ManualJournalRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.manual-journal-repository")}
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListManualJournalRequest,
) (*pagination.ListResult[*manualjournal.Request], error) {
	limit := req.Filter.Pagination.SafeLimit()
	items := make([]*manualjournal.Request, 0, limit)

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("mjr.organization_id = ?", req.Filter.TenantInfo.OrgID).
		Where("mjr.business_unit_id = ?", req.Filter.TenantInfo.BuID).
		Order("mjr.created_at DESC").
		Limit(limit).
		Offset(req.Filter.Pagination.SafeOffset())

	if req.Filter.Query != "" {
		query = query.Where("(mjr.request_number ILIKE ? OR mjr.description ILIKE ?)", "%"+req.Filter.Query+"%", "%"+req.Filter.Query+"%")
	}

	total, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("list manual journals: %w", err)
	}

	return &pagination.ListResult[*manualjournal.Request]{Items: items, Total: total}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetManualJournalByIDRequest,
) (*manualjournal.Request, error) {
	entity := new(manualjournal.Request)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("mjr.id = ?", req.ID).
		Where("mjr.organization_id = ?", req.TenantInfo.OrgID).
		Where("mjr.business_unit_id = ?", req.TenantInfo.BuID).
		Relation("Lines", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("mjrl.line_number ASC")
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "ManualJournalRequest")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *manualjournal.Request,
) (*manualjournal.Request, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("mjr_")
	}
	entity.SyncTotals()
	assignLineTenantFields(entity)

	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create manual journal request: %w", err)
	}

	if len(entity.Lines) > 0 {
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(&entity.Lines).Exec(ctx); err != nil {
			return nil, fmt.Errorf("create manual journal request lines: %w", err)
		}
	}

	return r.GetByID(ctx, repositories.GetManualJournalByIDRequest{
		ID:         entity.ID,
		TenantInfo: pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *manualjournal.Request,
) (*manualjournal.Request, error) {
	if entity.Lines != nil {
		entity.SyncTotals()
		assignLineTenantFields(entity)
	}

	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Where("id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Where("version = ?", entity.Version).
		Set("status = ?", entity.Status).
		Set("description = ?", entity.Description).
		Set("reason = ?", entity.Reason).
		Set("accounting_date = ?", entity.AccountingDate).
		Set("requested_fiscal_year_id = ?", entity.RequestedFiscalYearID).
		Set("requested_fiscal_period_id = ?", entity.RequestedFiscalPeriodID).
		Set("currency_code = ?", entity.CurrencyCode).
		Set("total_debit_minor = ?", entity.TotalDebit).
		Set("total_credit_minor = ?", entity.TotalCredit).
		Set("approved_at = ?", entity.ApprovedAt).
		Set("approved_by_id = ?", entity.ApprovedByID).
		Set("rejected_at = ?", entity.RejectedAt).
		Set("rejected_by_id = ?", entity.RejectedByID).
		Set("rejection_reason = ?", entity.RejectionReason).
		Set("cancelled_at = ?", entity.CancelledAt).
		Set("cancelled_by_id = ?", entity.CancelledByID).
		Set("cancel_reason = ?", entity.CancelReason).
		Set("posted_batch_id = ?", entity.PostedBatchID).
		Set("updated_by_id = ?", entity.UpdatedByID).
		Set("version = version + 1").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update manual journal request: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, "ManualJournalRequest", entity.ID.String()); err != nil {
		return nil, err
	}

	if entity.Lines != nil {
		if _, err = r.db.DBForContext(ctx).
			NewDelete().
			Model((*manualjournal.Line)(nil)).
			Where("manual_journal_request_id = ?", entity.ID).
			Where("organization_id = ?", entity.OrganizationID).
			Where("business_unit_id = ?", entity.BusinessUnitID).
			Exec(ctx); err != nil {
			return nil, fmt.Errorf("replace manual journal request lines: %w", err)
		}

		if len(entity.Lines) > 0 {
			if _, err = r.db.DBForContext(ctx).NewInsert().Model(&entity.Lines).Exec(ctx); err != nil {
				return nil, fmt.Errorf("insert manual journal request lines: %w", err)
			}
		}
	}

	return r.GetByID(ctx, repositories.GetManualJournalByIDRequest{
		ID:         entity.ID,
		TenantInfo: pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
	})
}

func assignLineTenantFields(entity *manualjournal.Request) {
	for idx, line := range entity.Lines {
		if line == nil {
			continue
		}

		line.OrganizationID = entity.OrganizationID
		line.BusinessUnitID = entity.BusinessUnitID
		line.ManualJournalRequestID = entity.ID
		line.LineNumber = idx + 1
	}
}
