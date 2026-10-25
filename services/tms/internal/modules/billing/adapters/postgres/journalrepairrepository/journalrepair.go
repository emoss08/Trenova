package journalrepairrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalsource/journalsourcedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger/customerledgerdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment/invoiceadjustmentdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement/driversettlementdb"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
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

func New(p Params) driversettlementports.JournalRepairRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.journal-repair-repository")}
}

func limitOf(req *driversettlementports.ListJournalRepairRequest) int {
	if req.Limit <= 0 || req.Limit > 1000 {
		return defaultLimit
	}
	return req.Limit
}

func (r *repository) ListUnjournaledAdjustmentMemos(
	ctx context.Context,
	req *driversettlementports.ListJournalRepairRequest,
) ([]*driversettlementports.AdjustmentMemoRepair, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"list adjustment credit memos missing a journal across every organization",
	)
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*driversettlementports.AdjustmentMemoRepair, error) {
			db := r.db.Keel(ctx)
			inv := invoicedb.Invoices
			cle := customerledgerdb.CustomerLedgerEntries.As("cle")
			ledgerLine := cle.AllTenants().Where(
				cle.SourceObjectID.EqCol(inv.ID),
				cle.OrganizationID.EqCol(inv.OrganizationID),
				cle.BusinessUnitID.EqCol(inv.BusinessUnitID),
			)

			memos, err := inv.AllTenants().
				Where(
					inv.IsAdjustmentArtifact.IsTrue(),
					inv.BillType.Eq(billingqueue.BillTypeCreditMemo),
					inv.Status.Eq(invoice.StatusPosted),
					inv.SourceInvoiceAdjustmentID.IsNotNull(),
					keel.NotExists(ledgerLine),
					keel.When(req.OrganizationID.IsNotNil(), inv.OrganizationID.Eq(req.OrganizationID)),
					keel.When(req.AfterID.IsNotNil(), inv.ID.Gt(req.AfterID)),
				).
				Order(inv.ID.Asc()).
				Limit(limitOf(req)).
				List(ctx, db)
			if err != nil {
				return nil, fmt.Errorf("list unjournaled adjustment credit memos: %w", err)
			}
			if len(memos) == 0 {
				return nil, nil
			}

			adjustmentIDs := make([]pulid.ID, 0, len(memos))
			memoIDs := make([]pulid.ID, 0, len(memos))
			for _, memo := range memos {
				adjustmentIDs = append(adjustmentIDs, memo.SourceInvoiceAdjustmentID)
				memoIDs = append(memoIDs, memo.ID)
			}
			ia := invoiceadjustmentdb.InvoiceAdjustments
			adjustments, err := ia.AllTenants().
				Pick(ia.ID, ia.OrganizationID, ia.BusinessUnitID, ia.Kind, ia.OriginalInvoiceID).
				Where(ia.ID.In(adjustmentIDs...)).
				All(ctx, db)
			if err != nil {
				return nil, fmt.Errorf("read adjustments for credit memos: %w", err)
			}
			byID := make(map[pulid.ID]*invoiceadjustment.InvoiceAdjustment, len(adjustments))
			for _, adjustment := range adjustments {
				byID[adjustment.ID] = adjustment
			}

			journaled, err := existingJournalBatches(ctx, db, &existingJournalRequest{
				ObjectType: "Invoice",
				Event:      tenant.JournalSourceEventCreditMemoPosted,
				ObjectIDs:  memoIDs,
			})
			if err != nil {
				return nil, err
			}

			repairs := make([]*driversettlementports.AdjustmentMemoRepair, 0, len(memos))
			for _, memo := range memos {
				adjustment, ok := byID[memo.SourceInvoiceAdjustmentID]
				if !ok || adjustment.OrganizationID != memo.OrganizationID ||
					adjustment.BusinessUnitID != memo.BusinessUnitID {
					repairs = append(repairs, &driversettlementports.AdjustmentMemoRepair{
						Memo:          memo,
						SourceMissing: true,
					})
					continue
				}
				repairs = append(repairs, &driversettlementports.AdjustmentMemoRepair{
					Memo:            memo,
					Kind:            adjustment.Kind,
					SourceInvoiceID: adjustment.OriginalInvoiceID,
					Journaled:       journaled[journalKey(memo.OrganizationID, memo.BusinessUnitID, memo.ID)].IsNotNil(),
				})
			}
			return repairs, nil
		},
	)
}

func (r *repository) ListUnjournaledDriverPayments(
	ctx context.Context,
	req *driversettlementports.ListJournalRepairRequest,
) ([]*driversettlementports.DriverPaymentRepair, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"list paid driver settlements missing a journal across every organization",
	)
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*driversettlementports.DriverPaymentRepair, error) {
			db := r.db.Keel(ctx)
			ds := driversettlementdb.Settlements
			settlements, err := ds.AllTenants().
				Where(
					ds.Status.Eq(driversettlement.StatusPaid),
					ds.NetPayMinor.Ne(0),
					ds.PaidJournalBatchID.IsNull(),
					ds.PaidAt.IsNotNull(),
					keel.When(req.OrganizationID.IsNotNil(), ds.OrganizationID.Eq(req.OrganizationID)),
					keel.When(req.AfterID.IsNotNil(), ds.ID.Gt(req.AfterID)),
				).
				Order(ds.ID.Asc()).
				Limit(limitOf(req)).
				List(ctx, db)
			if err != nil {
				return nil, fmt.Errorf("list unjournaled driver settlement payments: %w", err)
			}
			if len(settlements) == 0 {
				return nil, nil
			}

			settlementIDs := make([]pulid.ID, 0, len(settlements))
			for _, settlement := range settlements {
				settlementIDs = append(settlementIDs, settlement.ID)
			}
			batches, err := existingJournalBatches(ctx, db, &existingJournalRequest{
				ObjectType: "DriverSettlement",
				Event:      tenant.JournalSourceEventDriverSettlementPaid,
				ObjectIDs:  settlementIDs,
			})
			if err != nil {
				return nil, err
			}

			repairs := make([]*driversettlementports.DriverPaymentRepair, 0, len(settlements))
			for _, settlement := range settlements {
				repairs = append(repairs, &driversettlementports.DriverPaymentRepair{
					Settlement: settlement,
					JournalBatchID: batches[journalKey(
						settlement.OrganizationID,
						settlement.BusinessUnitID,
						settlement.ID,
					)],
				})
			}
			return repairs, nil
		},
	)
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
	db *keel.DB,
	req *existingJournalRequest,
) (map[journalSourceKey]pulid.ID, error) {
	objectIDs := make([]string, 0, len(req.ObjectIDs))
	for _, id := range req.ObjectIDs {
		objectIDs = append(objectIDs, id.String())
	}

	js := journalsourcedb.Sources
	sources, err := js.AllTenants().
		Pick(js.OrganizationID, js.BusinessUnitID, js.SourceObjectID, js.JournalBatchID).
		Where(
			js.SourceObjectType.Eq(req.ObjectType),
			js.SourceEventType.Eq(string(req.Event)),
			js.SourceObjectID.In(objectIDs...),
		).
		All(ctx, db)
	if err != nil {
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
	params *driversettlementports.SetDriverSettlementPaidBatchParams,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		ds := driversettlementdb.Settlements
		affected, err := ds.Where(ds.ID.Eq(params.SettlementID), ds.PaidJournalBatchID.IsNull()).
			Update(ctx, r.db.Keel(ctx), ds.PaidJournalBatchID.To(params.BatchID))
		if err != nil {
			return fmt.Errorf("record driver settlement payment journal: %w", err)
		}
		if affected != 1 {
			return errortypes.NewConflictError(
				"The driver settlement already has a payment journal",
			)
		}
		return nil
	})
}
