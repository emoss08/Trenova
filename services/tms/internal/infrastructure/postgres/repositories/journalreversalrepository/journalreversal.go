package journalreversalrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/journalreversal"
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

func New(p Params) repositories.JournalReversalRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.journal-reversal-repository")}
}

type reversalRecord struct {
	bun.BaseModel `bun:"table:journal_reversals,alias:jr"`

	ID                      pulid.ID `bun:"id,pk"`
	OrganizationID          pulid.ID `bun:"organization_id,pk"`
	BusinessUnitID          pulid.ID `bun:"business_unit_id,pk"`
	OriginalJournalEntryID  pulid.ID `bun:"original_journal_entry_id"`
	ReversalJournalEntryID  pulid.ID `bun:"reversal_journal_entry_id,nullzero"`
	PostedBatchID           pulid.ID `bun:"posted_batch_id,nullzero"`
	Status                  string   `bun:"status"`
	RequestedAccountingDate int64    `bun:"requested_accounting_date"`
	ResolvedFiscalYearID    pulid.ID `bun:"resolved_fiscal_year_id"`
	ResolvedFiscalPeriodID  pulid.ID `bun:"resolved_fiscal_period_id"`
	ReasonCode              string   `bun:"reason_code"`
	ReasonText              string   `bun:"reason_text"`
	RequestedByID           pulid.ID `bun:"requested_by_id"`
	ApprovedByID            pulid.ID `bun:"approved_by_id,nullzero"`
	ApprovedAt              *int64   `bun:"approved_at,nullzero"`
	RejectedByID            pulid.ID `bun:"rejected_by_id,nullzero"`
	RejectedAt              *int64   `bun:"rejected_at,nullzero"`
	RejectionReason         string   `bun:"rejection_reason,nullzero"`
	CancelledByID           pulid.ID `bun:"cancelled_by_id,nullzero"`
	CancelledAt             *int64   `bun:"cancelled_at,nullzero"`
	CancelReason            string   `bun:"cancel_reason,nullzero"`
	PostedByID              pulid.ID `bun:"posted_by_id,nullzero"`
	PostedAt                *int64   `bun:"posted_at,nullzero"`
	Version                 int64    `bun:"version"`
	CreatedAt               int64    `bun:"created_at"`
	UpdatedAt               int64    `bun:"updated_at"`
}

func (r *repository) List(ctx context.Context, req *repositories.ListJournalReversalsRequest) (*pagination.ListResult[*journalreversal.Reversal], error) {
	records := make([]*reversalRecord, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&records).
		Where("organization_id = ?", req.Filter.TenantInfo.OrgID).
		Where("business_unit_id = ?", req.Filter.TenantInfo.BuID).
		Order("created_at DESC").
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*journalreversal.Reversal, 0, len(records))
	for _, rec := range records {
		items = append(items, mapReversal(rec))
	}
	return &pagination.ListResult[*journalreversal.Reversal]{Items: items, Total: total}, nil
}

func (r *repository) GetByID(ctx context.Context, req repositories.GetJournalReversalByIDRequest) (*journalreversal.Reversal, error) {
	rec := new(reversalRecord)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(rec).
		Where("jr.id = ?", req.ID).
		Where("jr.organization_id = ?", req.TenantInfo.OrgID).
		Where("jr.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "JournalReversal")
	}
	return mapReversal(rec), nil
}

func (r *repository) Create(ctx context.Context, entity *journalreversal.Reversal) (*journalreversal.Reversal, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("jrev_")
	}
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(toRecord(entity)).Exec(ctx); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, repositories.GetJournalReversalByIDRequest{ID: entity.ID, TenantInfo: pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}})
}

func (r *repository) Update(ctx context.Context, entity *journalreversal.Reversal) (*journalreversal.Reversal, error) {
	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(toRecord(entity)).
		Where("id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Where("version = ?", entity.Version).
		Set("reversal_journal_entry_id = ?", entity.ReversalJournalEntryID).
		Set("posted_batch_id = ?", entity.PostedBatchID).
		Set("status = ?", entity.Status).
		Set("resolved_fiscal_year_id = ?", entity.ResolvedFiscalYearID).
		Set("resolved_fiscal_period_id = ?", entity.ResolvedFiscalPeriodID).
		Set("approved_by_id = ?", entity.ApprovedByID).
		Set("approved_at = ?", entity.ApprovedAt).
		Set("rejected_by_id = ?", entity.RejectedByID).
		Set("rejected_at = ?", entity.RejectedAt).
		Set("rejection_reason = ?", entity.RejectionReason).
		Set("cancelled_by_id = ?", entity.CancelledByID).
		Set("cancelled_at = ?", entity.CancelledAt).
		Set("cancel_reason = ?", entity.CancelReason).
		Set("posted_by_id = ?", entity.PostedByID).
		Set("posted_at = ?", entity.PostedAt).
		Set("version = version + 1").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, repositories.GetJournalReversalByIDRequest{ID: entity.ID, TenantInfo: pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}})
}

func toRecord(entity *journalreversal.Reversal) *reversalRecord {
	return &reversalRecord{
		ID:                      entity.ID,
		OrganizationID:          entity.OrganizationID,
		BusinessUnitID:          entity.BusinessUnitID,
		OriginalJournalEntryID:  entity.OriginalJournalEntryID,
		ReversalJournalEntryID:  entity.ReversalJournalEntryID,
		PostedBatchID:           entity.PostedBatchID,
		Status:                  entity.Status.String(),
		RequestedAccountingDate: entity.RequestedAccountingDate,
		ResolvedFiscalYearID:    entity.ResolvedFiscalYearID,
		ResolvedFiscalPeriodID:  entity.ResolvedFiscalPeriodID,
		ReasonCode:              entity.ReasonCode,
		ReasonText:              entity.ReasonText,
		RequestedByID:           entity.RequestedByID,
		ApprovedByID:            entity.ApprovedByID,
		ApprovedAt:              entity.ApprovedAt,
		RejectedByID:            entity.RejectedByID,
		RejectedAt:              entity.RejectedAt,
		RejectionReason:         entity.RejectionReason,
		CancelledByID:           entity.CancelledByID,
		CancelledAt:             entity.CancelledAt,
		CancelReason:            entity.CancelReason,
		PostedByID:              entity.PostedByID,
		PostedAt:                entity.PostedAt,
		Version:                 entity.Version,
	}
}

func mapReversal(rec *reversalRecord) *journalreversal.Reversal {
	return &journalreversal.Reversal{
		ID:                      rec.ID,
		OrganizationID:          rec.OrganizationID,
		BusinessUnitID:          rec.BusinessUnitID,
		OriginalJournalEntryID:  rec.OriginalJournalEntryID,
		ReversalJournalEntryID:  rec.ReversalJournalEntryID,
		PostedBatchID:           rec.PostedBatchID,
		Status:                  journalreversal.Status(rec.Status),
		RequestedAccountingDate: rec.RequestedAccountingDate,
		ResolvedFiscalYearID:    rec.ResolvedFiscalYearID,
		ResolvedFiscalPeriodID:  rec.ResolvedFiscalPeriodID,
		ReasonCode:              rec.ReasonCode,
		ReasonText:              rec.ReasonText,
		RequestedByID:           rec.RequestedByID,
		ApprovedByID:            rec.ApprovedByID,
		ApprovedAt:              rec.ApprovedAt,
		RejectedByID:            rec.RejectedByID,
		RejectedAt:              rec.RejectedAt,
		RejectionReason:         rec.RejectionReason,
		CancelledByID:           rec.CancelledByID,
		CancelledAt:             rec.CancelledAt,
		CancelReason:            rec.CancelReason,
		PostedByID:              rec.PostedByID,
		PostedAt:                rec.PostedAt,
		Version:                 rec.Version,
		CreatedAt:               rec.CreatedAt,
		UpdatedAt:               rec.UpdatedAt,
	}
}
