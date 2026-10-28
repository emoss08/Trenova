package accountingsyncrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync/accountingsyncdb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod/fiscalperioddb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingdriftports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement/carriersettlementdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement/driversettlementdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultDriftBatch     = 200
	maxDriftBatch         = 1000
	defaultDriftCustomers = 50
	maxDriftCustomers     = 200
	newerRecordAlias      = "newer"
	reflectedRecordAlias  = "rrec"
	reflectedMemoAlias    = "rmemo"
)

type DriftSourceParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type driftSource struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewDriftSource(p DriftSourceParams) accountingdriftports.AccountingDriftSource {
	return &driftSource{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-drift-source"),
	}
}

func (s *driftSource) ScopeStart(
	ctx context.Context,
	_ pagination.TenantInfo,
) (int64, error) {
	return dbtx.Read(ctx, s.db, func(ctx context.Context) (int64, error) {
		fp := fiscalperioddb.FiscalPeriods
		start, err := keel.IntoOne[int64](
			ctx,
			s.db.Keel(ctx),
			fp.Pick(keel.Int64Of(fp.StartDate.Min()).Coalesce(0)).
				Where(fp.Status.In(fiscalperiod.UnclosedStatuses()...)),
		)
		if err != nil {
			return 0, fmt.Errorf("find the start of the earliest open period: %w", err)
		}
		return start, nil
	})
}

func latestSynced(
	connectionID pulid.ID,
	objectTypes []accountingsync.SyncObjectType,
) []keel.Pred {
	sr := accountingsyncdb.AccountingSyncRecords
	newer := sr.As(newerRecordAlias)
	creates := accountingsync.DriftCreateOperations()

	return []keel.Pred{
		sr.ConnectionID.Eq(connectionID),
		sr.Status.Eq(accountingsync.SyncStatusSynced),
		sr.Operation.In(creates...),
		sr.ObjectType.In(objectTypes...),
		sr.ExternalID.IsNotNull(),
		keel.NotExists(newer.Where(
			newer.ConnectionID.EqCol(sr.ConnectionID),
			newer.ObjectType.EqCol(sr.ObjectType),
			newer.ObjectID.EqCol(sr.ObjectID),
			newer.Status.Eq(accountingsync.SyncStatusSynced),
			newer.Operation.In(creates...),
			keel.Int64Of(newer.Revision).GtExpr(sr.Revision),
		)),
	}
}

func (s *driftSource) ListRecords(
	ctx context.Context,
	req *accountingdriftports.ListAccountingDriftRecordsRequest,
) ([]*accountingsync.AccountingSyncRecord, error) {
	return dbtx.Read(
		ctx,
		s.db,
		func(ctx context.Context) ([]*accountingsync.AccountingSyncRecord, error) {
			limit := req.Limit
			if limit <= 0 {
				limit = defaultDriftBatch
			}
			limit = intutils.Clamp(limit, 1, maxDriftBatch)

			sr := accountingsyncdb.AccountingSyncRecords
			entities, err := sr.Where(latestSynced(req.ConnectionID, accountingsync.DriftObjectTypes())...).
				Where(
					keel.Or(sr.DocumentDate.IsNull(), sr.DocumentDate.Gte(req.DatedFrom)),
					keel.When(!req.AfterID.IsNil(), sr.ID.Gt(req.AfterID)),
				).
				Order(sr.ID.Asc()).
				Limit(limit).
				List(ctx, s.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("list records to compare: %w", err)
			}
			return entities, nil
		},
	)
}

func (s *driftSource) ListStates(
	ctx context.Context,
	req *accountingdriftports.ListAccountingDriftStatesRequest,
) ([]*accountingdriftports.AccountingDriftState, error) {
	return dbtx.Read(
		ctx,
		s.db,
		func(ctx context.Context) ([]*accountingdriftports.AccountingDriftState, error) {
			if len(req.ObjectIDs) == 0 {
				return []*accountingdriftports.AccountingDriftState{}, nil
			}

			var query keel.Projection
			switch {
			case req.ObjectType.IsSalesDocument():
				query = salesStates(req)
			case req.ObjectType == accountingsync.SyncObjectCustomerPayment:
				query = paymentStates(req)
			case req.ObjectType == accountingsync.SyncObjectCreditApplication:
				query = applicationStates(req)
			case req.ObjectType.IsPayable() && !req.ObjectType.IsVendor():
				if req.ObjectType.IsDriverSettlement() {
					query = driverStates(req)
				} else {
					query = carrierStates(req)
				}
			default:
				return nil, fmt.Errorf("drift does not compare %s", req.ObjectType)
			}

			states, err := keel.Into[accountingdriftports.AccountingDriftState](
				ctx,
				s.db.Keel(ctx),
				query,
			)
			if err != nil {
				return nil, fmt.Errorf("read %s states: %w", req.ObjectType, err)
			}
			return sliceutils.Pointers(states), nil
		},
	)
}

func reflectedMemos(
	connectionID pulid.ID,
	amount func(memo *invoicedb.InvoicesTable) keel.Expr,
) keel.Expr {
	inv := invoicedb.Invoices
	memo := inv.As(reflectedMemoAlias)
	rec := accountingsyncdb.AccountingSyncRecords.As(reflectedRecordAlias)

	return keel.Scalar(memo.Join(rec, rec.ObjectID.EqCol(memo.ID)).
		Pick(keel.Int64Of(keel.Sum(amount(memo))).Coalesce(0)).
		Where(
			memo.ReferenceInvoiceID.EqCol(inv.ID),
			memo.Status.Eq(invoice.StatusPosted),
			rec.ConnectionID.Eq(connectionID),
			rec.Status.Eq(accountingsync.SyncStatusSkipped),
			rec.ExternalRefs.Text(accountingsync.ExternalRefReflectedIn).IsNotNull(),
		))
}

func salesStates(req *accountingdriftports.ListAccountingDriftStatesRequest) keel.Projection {
	inv := invoicedb.Invoices
	cus := customerdb.Customers
	return inv.Join(cus, cus.ID.EqCol(inv.CustomerID)).
		Pick(
			inv.ID,
			inv.Number,
			inv.CustomerID,
			cus.Name,
			inv.CurrencyCode,
			inv.TotalAmountMinor.Abs(),
			inv.BalanceDueMinor,
			inv.AppliedAmountMinor,
			reflectedMemos(req.ConnectionID, func(memo *invoicedb.InvoicesTable) keel.Expr {
				return memo.TotalAmountMinor
			}),
			inv.Status.Eq(invoice.StatusVoided),
			inv.Status,
		).
		Where(inv.ID.In(req.ObjectIDs...))
}

func paymentStates(req *accountingdriftports.ListAccountingDriftStatesRequest) keel.Projection {
	cp := customerpaymentdb.Payments
	cus := customerdb.Customers
	return cp.Join(cus, cus.ID.EqCol(cp.CustomerID)).
		Pick(
			cp.ID,
			cp.ReferenceNumber.Coalesce(""),
			cp.CustomerID,
			cus.Name,
			cp.CurrencyCode,
			cp.AmountMinor,
			cp.UnappliedAmountMinor,
			cp.AppliedAmountMinor,
			keel.Int64(0),
			cp.Status.Eq(customerpayment.StatusReversed),
			cp.Status,
		).
		Where(cp.ID.In(req.ObjectIDs...))
}

func applicationStates(req *accountingdriftports.ListAccountingDriftStatesRequest) keel.Projection {
	cma := customerpaymentdb.CreditMemoApplications
	inv := invoicedb.Invoices
	cus := customerdb.Customers
	return cma.Join(inv, inv.ID.EqCol(cma.CreditMemoInvoiceID)).
		Join(cus, cus.ID.EqCol(inv.CustomerID)).
		Pick(
			cma.ID,
			inv.Number,
			inv.CustomerID,
			cus.Name,
			inv.CurrencyCode,
			cma.AppliedAmountMinor,
			keel.Int64(0),
			cma.AppliedAmountMinor,
			keel.Int64(0),
			cma.Status.Eq(customerpayment.CreditApplicationStatusUnapplied),
			cma.Status,
		).
		Where(cma.ID.In(req.ObjectIDs...))
}

func settlementVoided(
	objectType accountingsync.SyncObjectType,
	isVoided, isPaid keel.Pred,
) keel.Expr {
	if objectType.IsBillPayment() {
		return keel.Not(isPaid)
	}
	return isVoided
}

func carrierStates(req *accountingdriftports.ListAccountingDriftStatesRequest) keel.Projection {
	cs := carriersettlementdb.CarrierSettlements
	carr := carrierdb.Carriers
	return cs.Join(carr, carr.ID.EqCol(cs.CarrierID)).
		Pick(
			cs.ID,
			cs.SettlementNumber,
			cs.CarrierID,
			carr.Name,
			cs.CurrencyCode,
			cs.NetPayableMinor.Abs(),
			keel.Int64(0),
			keel.Int64(0),
			keel.Int64(0),
			settlementVoided(
				req.ObjectType,
				cs.Status.Eq(carriersettlement.StatusVoided),
				cs.Status.Eq(carriersettlement.StatusPaid),
			),
			cs.Status,
		).
		Where(cs.ID.In(req.ObjectIDs...))
}

func driverStates(req *accountingdriftports.ListAccountingDriftStatesRequest) keel.Projection {
	ds := driversettlementdb.Settlements
	wrk := workerdb.Workers
	return ds.Join(wrk, wrk.ID.EqCol(ds.WorkerID)).
		Pick(
			ds.ID,
			ds.SettlementNumber,
			ds.WorkerID,
			wrk.FirstName.Cat(keel.Text(" "), wrk.LastName).Trim(),
			ds.CurrencyCode,
			ds.NetPayMinor.Abs(),
			keel.Int64(0),
			keel.Int64(0),
			keel.Int64(0),
			settlementVoided(
				req.ObjectType,
				ds.Status.Eq(driversettlement.StatusVoided),
				ds.Status.Eq(driversettlement.StatusPaid),
			),
			ds.Status,
		).
		Where(ds.ID.In(req.ObjectIDs...))
}

func balanceDocuments(
	req *accountingdriftports.ListAccountingDriftBalancesRequest,
) *keel.Query[*accountingsync.AccountingSyncRecord] {
	sr := accountingsyncdb.AccountingSyncRecords
	inv := invoicedb.Invoices
	return sr.Join(inv, inv.ID.EqCol(sr.ObjectID)).
		Where(inv.InvoiceDate.Gte(req.DatedFrom)).
		Where(latestSynced(req.ConnectionID, accountingsync.DriftBalanceObjectTypes())...)
}

func (s *driftSource) ListBalances(
	ctx context.Context,
	req *accountingdriftports.ListAccountingDriftBalancesRequest,
) ([]*accountingdriftports.AccountingDriftBalanceLine, error) {
	return dbtx.Read(
		ctx,
		s.db,
		func(ctx context.Context) ([]*accountingdriftports.AccountingDriftBalanceLine, error) {
			customers := req.Customers
			if customers <= 0 {
				customers = defaultDriftCustomers
			}
			customers = intutils.Clamp(customers, 1, maxDriftCustomers)

			db := s.db.Keel(ctx)
			inv := invoicedb.Invoices
			cus := customerdb.Customers
			sr := accountingsyncdb.AccountingSyncRecords

			ids, err := keel.Into[pulid.ID](ctx, db, balanceDocuments(req).
				Group(inv.CustomerID).
				Pick(inv.CustomerID).
				Where(keel.When(!req.AfterCustomerID.IsNil(), inv.CustomerID.Gt(req.AfterCustomerID))).
				Order(inv.CustomerID.Asc()).
				Limit(customers))
			if err != nil {
				return nil, fmt.Errorf("list customers to balance: %w", err)
			}
			if len(ids) == 0 {
				return []*accountingdriftports.AccountingDriftBalanceLine{}, nil
			}

			lines, err := keel.Into[accountingdriftports.AccountingDriftBalanceLine](
				ctx,
				db,
				balanceDocuments(req).
					Join(cus, cus.ID.EqCol(inv.CustomerID)).
					Pick(
						inv.CustomerID,
						cus.Name,
						inv.CurrencyCode,
						sr.ObjectType,
						inv.ID,
						inv.Number,
						inv.BalanceDueMinor.Add(keel.Int64Of(reflectedMemos(
							req.ConnectionID,
							func(memo *invoicedb.InvoicesTable) keel.Expr { return memo.BalanceDueMinor },
						))),
						sr.ExternalID,
					).
					Where(inv.CustomerID.In(ids...)).
					Order(inv.CustomerID.Asc(), inv.ID.Asc()),
			)
			if err != nil {
				return nil, fmt.Errorf("list balances to compare: %w", err)
			}
			return sliceutils.Pointers(lines), nil
		},
	)
}

func (s *driftSource) ListPendingCustomers(
	ctx context.Context,
	req *accountingdriftports.ListAccountingDriftPendingCustomersRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, s.db, func(ctx context.Context) ([]pulid.ID, error) {
		if len(req.CustomerIDs) == 0 {
			return []pulid.ID{}, nil
		}

		found := make([][]pulid.ID, 4)
		err := keel.Batch(
			ctx,
			s.db.Keel(ctx),
			keel.IntoSlice(waitingInbound(req), &found[0]),
			keel.IntoSlice(unsettledSales(req), &found[1]),
			keel.IntoSlice(unsettledPayments(req), &found[2]),
			keel.IntoSlice(unsettledApplications(req), &found[3]),
		)
		if err != nil {
			return nil, fmt.Errorf("list customers with changes still waiting: %w", err)
		}

		ids := make([]pulid.ID, 0, len(req.CustomerIDs))
		seen := make(map[pulid.ID]struct{}, len(req.CustomerIDs))
		for _, group := range found {
			for _, id := range group {
				if _, dup := seen[id]; dup {
					continue
				}
				seen[id] = struct{}{}
				ids = append(ids, id)
			}
		}
		return ids, nil
	})
}

func waitingInbound(
	req *accountingdriftports.ListAccountingDriftPendingCustomersRequest,
) keel.Projection {
	ic := accountingsyncdb.AccountingInboundChanges
	return ic.Group(ic.PartyObjectID).
		Pick(ic.PartyObjectID).
		Where(
			ic.ConnectionID.Eq(req.ConnectionID),
			ic.Kind.Eq(accountingsync.InboundCustomerPayment),
			ic.Status.In(accountingsync.InboundStatusDetected, accountingsync.InboundStatusProposed),
			ic.PartyObjectID.In(req.CustomerIDs...),
		)
}

func unsettledRecords(
	req *accountingdriftports.ListAccountingDriftPendingCustomersRequest,
	objectTypes ...accountingsync.SyncObjectType,
) *keel.Query[*accountingsync.AccountingSyncRecord] {
	sr := accountingsyncdb.AccountingSyncRecords
	return sr.Where(
		sr.ConnectionID.Eq(req.ConnectionID),
		sr.ObjectType.In(objectTypes...),
		sr.Status.NotIn(accountingsync.FinalSyncStatuses()...),
	)
}

func unsettledSales(
	req *accountingdriftports.ListAccountingDriftPendingCustomersRequest,
) keel.Projection {
	sr := accountingsyncdb.AccountingSyncRecords
	inv := invoicedb.Invoices
	return unsettledRecords(
		req,
		accountingsync.SyncObjectInvoice,
		accountingsync.SyncObjectCreditMemo,
		accountingsync.SyncObjectDebitMemo,
	).
		Join(inv, inv.ID.EqCol(sr.ObjectID)).
		Group(inv.CustomerID).
		Pick(inv.CustomerID).
		Where(inv.CustomerID.In(req.CustomerIDs...))
}

func unsettledPayments(
	req *accountingdriftports.ListAccountingDriftPendingCustomersRequest,
) keel.Projection {
	sr := accountingsyncdb.AccountingSyncRecords
	cp := customerpaymentdb.Payments
	return unsettledRecords(req, accountingsync.SyncObjectCustomerPayment).
		Join(cp, cp.ID.EqCol(sr.ObjectID)).
		Group(cp.CustomerID).
		Pick(cp.CustomerID).
		Where(cp.CustomerID.In(req.CustomerIDs...))
}

func unsettledApplications(
	req *accountingdriftports.ListAccountingDriftPendingCustomersRequest,
) keel.Projection {
	sr := accountingsyncdb.AccountingSyncRecords
	cma := customerpaymentdb.CreditMemoApplications
	inv := invoicedb.Invoices
	return unsettledRecords(req, accountingsync.SyncObjectCreditApplication).
		Join(cma, cma.ID.EqCol(sr.ObjectID)).
		Join(inv, inv.ID.EqCol(cma.CreditMemoInvoiceID)).
		Group(inv.CustomerID).
		Pick(inv.CustomerID).
		Where(inv.CustomerID.In(req.CustomerIDs...))
}
