package journalrepairrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customerledger"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/journalsource"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const defaultLimit = 200

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.JournalRepairRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.journal-repair-repository")}
}

func limitOf(req *repositories.ListJournalRepairRequest) int {
	if req.Limit <= 0 || req.Limit > 1000 {
		return defaultLimit
	}
	return req.Limit
}

func (r *repository) ListUnjournaledAdjustmentMemos(
	ctx context.Context,
	req *repositories.ListJournalRepairRequest,
) ([]*repositories.AdjustmentMemoRepair, error) {
	dba := r.db.DBForContext(ctx)
	cols := buncolgen.InvoiceColumns
	ledgerCols := buncolgen.CustomerLedgerEntryColumns

	ledgerLine := dba.NewSelect().
		Model((*customerledger.CustomerLedgerEntry)(nil)).
		ColumnExpr("1").
		Where(ledgerCols.SourceObjectID.EqColumn(cols.ID)).
		Where(ledgerCols.OrganizationID.EqColumn(cols.OrganizationID)).
		Where(ledgerCols.BusinessUnitID.EqColumn(cols.BusinessUnitID))

	memos := make([]*invoice.Invoice, 0, limitOf(req))
	q := dba.NewSelect().
		Model(&memos).
		Where(cols.IsAdjustmentArtifact.IsTrue()).
		Where(cols.BillType.Eq(), billingqueue.BillTypeCreditMemo).
		Where(cols.Status.Eq(), invoice.StatusPosted).
		Where(cols.SourceInvoiceAdjustmentID.IsNotNull()).
		Where("NOT EXISTS (?)", ledgerLine).
		Order(cols.ID.OrderAsc()).
		Limit(limitOf(req))
	if req.OrganizationID.IsNotNil() {
		q = q.Where(cols.OrganizationID.Eq(), req.OrganizationID)
	}
	if req.AfterID.IsNotNil() {
		q = q.Where(cols.ID.Gt(), req.AfterID)
	}
	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("list unjournaled adjustment credit memos: %w", err)
	}
	if len(memos) == 0 {
		return nil, nil
	}

	adjustmentIDs := make([]pulid.ID, 0, len(memos))
	for _, memo := range memos {
		adjustmentIDs = append(adjustmentIDs, memo.SourceInvoiceAdjustmentID)
	}
	adjustments := make([]*invoiceadjustment.InvoiceAdjustment, 0, len(adjustmentIDs))
	adjCols := buncolgen.InvoiceAdjustmentColumns
	if err := dba.NewSelect().
		Model(&adjustments).
		Column(adjCols.ID.String(), adjCols.OrganizationID.String(), adjCols.BusinessUnitID.String(),
			adjCols.Kind.String(), adjCols.OriginalInvoiceID.String()).
		Where(adjCols.ID.In(), bun.List(adjustmentIDs)).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("read adjustments for credit memos: %w", err)
	}
	byID := make(map[pulid.ID]*invoiceadjustment.InvoiceAdjustment, len(adjustments))
	for _, adjustment := range adjustments {
		byID[adjustment.ID] = adjustment
	}

	memoIDs := make([]pulid.ID, 0, len(memos))
	for _, memo := range memos {
		memoIDs = append(memoIDs, memo.ID)
	}
	journaled, err := existingJournalBatches(ctx, dba, &existingJournalRequest{
		ObjectType: "Invoice",
		Event:      tenant.JournalSourceEventCreditMemoPosted,
		ObjectIDs:  memoIDs,
	})
	if err != nil {
		return nil, err
	}

	repairs := make([]*repositories.AdjustmentMemoRepair, 0, len(memos))
	for _, memo := range memos {
		adjustment, ok := byID[memo.SourceInvoiceAdjustmentID]
		if !ok || adjustment.OrganizationID != memo.OrganizationID ||
			adjustment.BusinessUnitID != memo.BusinessUnitID {
			continue
		}
		repairs = append(repairs, &repositories.AdjustmentMemoRepair{
			Memo:            memo,
			Kind:            adjustment.Kind,
			SourceInvoiceID: adjustment.OriginalInvoiceID,
			Journaled:       journaled[journalKey(memo.OrganizationID, memo.BusinessUnitID, memo.ID)].IsNotNil(),
		})
	}
	return repairs, nil
}

func (r *repository) ListUnjournaledDriverPayments(
	ctx context.Context,
	req *repositories.ListJournalRepairRequest,
) ([]*repositories.DriverPaymentRepair, error) {
	dba := r.db.DBForContext(ctx)
	cols := buncolgen.SettlementColumns
	settlements := make([]*driversettlement.Settlement, 0, limitOf(req))
	q := dba.NewSelect().
		Model(&settlements).
		Where(cols.Status.Eq(), driversettlement.StatusPaid).
		Where(cols.NetPayMinor.Ne(), 0).
		Where(cols.PaidJournalBatchID.IsNull()).
		Where(cols.PaidAt.IsNotNull()).
		Order(cols.ID.OrderAsc()).
		Limit(limitOf(req))
	if req.OrganizationID.IsNotNil() {
		q = q.Where(cols.OrganizationID.Eq(), req.OrganizationID)
	}
	if req.AfterID.IsNotNil() {
		q = q.Where(cols.ID.Gt(), req.AfterID)
	}
	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("list unjournaled driver settlement payments: %w", err)
	}
	if len(settlements) == 0 {
		return nil, nil
	}

	settlementIDs := make([]pulid.ID, 0, len(settlements))
	for _, settlement := range settlements {
		settlementIDs = append(settlementIDs, settlement.ID)
	}
	batches, err := existingJournalBatches(ctx, dba, &existingJournalRequest{
		ObjectType: "DriverSettlement",
		Event:      tenant.JournalSourceEventDriverSettlementPaid,
		ObjectIDs:  settlementIDs,
	})
	if err != nil {
		return nil, err
	}

	repairs := make([]*repositories.DriverPaymentRepair, 0, len(settlements))
	for _, settlement := range settlements {
		repairs = append(repairs, &repositories.DriverPaymentRepair{
			Settlement: settlement,
			JournalBatchID: batches[journalKey(
				settlement.OrganizationID,
				settlement.BusinessUnitID,
				settlement.ID,
			)],
		})
	}
	return repairs, nil
}

type existingJournalRequest struct {
	ObjectType string
	Event      tenant.JournalSourceEventType
	ObjectIDs  []pulid.ID
}

type journalSourceKey struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	ObjectID       string
}

func journalKey(orgID, buID, objectID pulid.ID) journalSourceKey {
	return journalSourceKey{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ObjectID:       objectID.String(),
	}
}

func existingJournalBatches(
	ctx context.Context,
	dba bun.IDB,
	req *existingJournalRequest,
) (map[journalSourceKey]pulid.ID, error) {
	objectIDs := make([]string, 0, len(req.ObjectIDs))
	for _, id := range req.ObjectIDs {
		objectIDs = append(objectIDs, id.String())
	}

	cols := buncolgen.SourceColumns
	sources := make([]*journalsource.Source, 0, len(objectIDs))
	if err := dba.NewSelect().
		Model(&sources).
		Column(
			cols.OrganizationID.String(),
			cols.BusinessUnitID.String(),
			cols.SourceObjectID.String(),
			cols.JournalBatchID.String(),
		).
		Where(cols.SourceObjectType.Eq(), req.ObjectType).
		Where(cols.SourceEventType.Eq(), string(req.Event)).
		Where(cols.SourceObjectID.In(), bun.List(objectIDs)).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("read existing %s journals: %w", req.Event, err)
	}

	batches := make(map[journalSourceKey]pulid.ID, len(sources))
	for _, source := range sources {
		batches[journalSourceKey{
			OrganizationID: source.OrganizationID,
			BusinessUnitID: source.BusinessUnitID,
			ObjectID:       source.SourceObjectID,
		}] = source.JournalBatchID
	}
	return batches, nil
}

func (r *repository) SetDriverSettlementPaidBatch(
	ctx context.Context,
	params *repositories.SetDriverSettlementPaidBatchParams,
) error {
	cols := buncolgen.SettlementColumns
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*driversettlement.Settlement)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.SettlementScopeTenantUpdate(uq, params.TenantInfo).
				Where(cols.ID.Eq(), params.SettlementID).
				Where(cols.PaidJournalBatchID.IsNull())
		}).
		Set(cols.PaidJournalBatchID.Set(), params.BatchID).
		Set(cols.Version.Inc(1)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("record driver settlement payment journal: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if affected != 1 {
		return errortypes.NewConflictError(
			"The driver settlement already has a payment journal",
		)
	}
	return nil
}
