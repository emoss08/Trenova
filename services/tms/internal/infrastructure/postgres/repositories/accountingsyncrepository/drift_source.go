package accountingsyncrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
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

func NewDriftSource(p DriftSourceParams) repositories.AccountingDriftSource {
	return &driftSource{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-drift-source"),
	}
}

func joinOn(table buncolgen.TableInfo, alias string, conditions ...string) string {
	return "JOIN " + table.As(alias) + " ON " + strings.Join(conditions, " AND ")
}

func (s *driftSource) ScopeStart(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (int64, error) {
	cols := buncolgen.FiscalPeriodColumns
	var start int64
	if err := s.db.DBForContext(ctx).
		NewSelect().
		Model((*fiscalperiod.FiscalPeriod)(nil)).
		ColumnExpr("COALESCE(MIN("+cols.StartDate.Qualified()+"), 0)").
		Where(cols.OrganizationID.Eq(), tenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), tenantInfo.BuID).
		Where(cols.Status.In(), bun.List(fiscalperiod.UnclosedStatuses())).
		Scan(ctx, &start); err != nil {
		return 0, fmt.Errorf("find the start of the earliest open period: %w", err)
	}
	return start, nil
}

func (s *driftSource) newerCreate(dba bun.IDB) *bun.SelectQuery {
	records := buncolgen.AccountingSyncRecordColumns
	newer := func(c buncolgen.Column) buncolgen.Column { return c.WithAlias(newerRecordAlias) }
	return dba.NewSelect().
		TableExpr(buncolgen.AccountingSyncRecordTable.As(newerRecordAlias)).
		ColumnExpr("1").
		Where(newer(records.OrganizationID).EqColumn(records.OrganizationID)).
		Where(newer(records.BusinessUnitID).EqColumn(records.BusinessUnitID)).
		Where(newer(records.ConnectionID).EqColumn(records.ConnectionID)).
		Where(newer(records.ObjectType).EqColumn(records.ObjectType)).
		Where(newer(records.ObjectID).EqColumn(records.ObjectID)).
		Where(newer(records.Status).Eq(), accountingsync.SyncStatusSynced).
		Where(newer(records.Operation).In(), bun.List(accountingsync.DriftCreateOperations())).
		Where(newer(records.Revision).Qualified() + " > " + records.Revision.Qualified())
}

func (s *driftSource) latestSynced(
	q *bun.SelectQuery,
	dba bun.IDB,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
	objectTypes []accountingsync.SyncObjectType,
) *bun.SelectQuery {
	records := buncolgen.AccountingSyncRecordColumns
	return q.
		Where(records.OrganizationID.Eq(), tenantInfo.OrgID).
		Where(records.BusinessUnitID.Eq(), tenantInfo.BuID).
		Where(records.ConnectionID.Eq(), connectionID).
		Where(records.Status.Eq(), accountingsync.SyncStatusSynced).
		Where(records.Operation.In(), bun.List(accountingsync.DriftCreateOperations())).
		Where(records.ObjectType.In(), bun.List(objectTypes)).
		Where(records.ExternalID.IsNotNull()).
		Where("NOT EXISTS (?)", s.newerCreate(dba))
}

func (s *driftSource) ListRecords(
	ctx context.Context,
	req *repositories.ListAccountingDriftRecordsRequest,
) ([]*accountingsync.AccountingSyncRecord, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultDriftBatch
	}
	limit = intutils.Clamp(limit, 1, maxDriftBatch)

	records := buncolgen.AccountingSyncRecordColumns
	dba := s.db.DBForContext(ctx)
	entities := make([]*accountingsync.AccountingSyncRecord, 0, limit)
	query := dba.NewSelect().Model(&entities)
	query = s.latestSynced(
		query,
		dba,
		req.TenantInfo,
		req.ConnectionID,
		accountingsync.DriftObjectTypes(),
	)
	query = query.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.
			Where(records.DocumentDate.IsNull()).
			WhereOr(records.DocumentDate.Gte(), req.DatedFrom)
	})
	if !req.AfterID.IsNil() {
		query = query.Where(records.ID.Gt(), req.AfterID)
	}
	if err := query.
		Order(records.ID.OrderAsc()).
		Limit(limit).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("list records to compare: %w", err)
	}
	return entities, nil
}

func (s *driftSource) ListStates(
	ctx context.Context,
	req *repositories.ListAccountingDriftStatesRequest,
) ([]*repositories.AccountingDriftState, error) {
	states := make([]*repositories.AccountingDriftState, 0, len(req.ObjectIDs))
	if len(req.ObjectIDs) == 0 {
		return states, nil
	}

	var query *bun.SelectQuery
	dba := s.db.DBForContext(ctx)
	switch {
	case req.ObjectType.IsSalesDocument():
		query = s.salesStates(dba, req)
	case req.ObjectType == accountingsync.SyncObjectCustomerPayment:
		query = s.paymentStates(dba, req)
	case req.ObjectType == accountingsync.SyncObjectCreditApplication:
		query = s.applicationStates(dba, req)
	case req.ObjectType.IsPayable() && !req.ObjectType.IsVendor():
		if req.ObjectType.IsDriverSettlement() {
			query = s.driverStates(dba, req)
		} else {
			query = s.carrierStates(dba, req)
		}
	default:
		return nil, fmt.Errorf("drift does not compare %s", req.ObjectType)
	}
	if err := query.Scan(ctx, &states); err != nil {
		return nil, fmt.Errorf("read %s states: %w", req.ObjectType, err)
	}
	return states, nil
}

func (s *driftSource) reflectedMemos(
	dba bun.IDB,
	connectionID pulid.ID,
	sum *buncolgen.Column,
) *bun.SelectQuery {
	inv := buncolgen.InvoiceColumns
	records := buncolgen.AccountingSyncRecordColumns
	memo := func(c buncolgen.Column) buncolgen.Column { return c.WithAlias(reflectedMemoAlias) }
	rec := func(c buncolgen.Column) buncolgen.Column { return c.WithAlias(reflectedRecordAlias) }
	return dba.NewSelect().
		TableExpr(buncolgen.InvoiceTable.As(reflectedMemoAlias)).
		ColumnExpr("COALESCE(SUM("+memo(*sum).Qualified()+"), 0)").
		Join(joinOn(
			buncolgen.AccountingSyncRecordTable,
			reflectedRecordAlias,
			rec(records.OrganizationID).EqColumn(memo(inv.OrganizationID)),
			rec(records.BusinessUnitID).EqColumn(memo(inv.BusinessUnitID)),
			rec(records.ObjectID).EqColumn(memo(inv.ID)),
		)).
		Where(memo(inv.OrganizationID).EqColumn(inv.OrganizationID)).
		Where(memo(inv.BusinessUnitID).EqColumn(inv.BusinessUnitID)).
		Where(memo(inv.ReferenceInvoiceID).EqColumn(inv.ID)).
		Where(memo(inv.Status).Eq(), invoice.StatusPosted).
		Where(rec(records.ConnectionID).Eq(), connectionID).
		Where(rec(records.Status).Eq(), accountingsync.SyncStatusSkipped).
		Where(
			rec(records.ExternalRefs).Qualified()+" ->> ? IS NOT NULL",
			accountingsync.ExternalRefReflectedIn,
		)
}

func (s *driftSource) salesStates(
	dba bun.IDB,
	req *repositories.ListAccountingDriftStatesRequest,
) *bun.SelectQuery {
	inv := buncolgen.InvoiceColumns
	cus := buncolgen.CustomerColumns
	return dba.NewSelect().
		Model((*invoice.Invoice)(nil)).
		ColumnExpr(inv.ID.As("object_id")).
		ColumnExpr(inv.Number.As("number")).
		ColumnExpr(inv.CustomerID.As("party_id")).
		ColumnExpr(cus.Name.As("party_name")).
		ColumnExpr(inv.CurrencyCode.As("currency_code")).
		ColumnExpr("ABS("+inv.TotalAmountMinor.Qualified()+") AS amount_minor").
		ColumnExpr(inv.BalanceDueMinor.As("open_minor")).
		ColumnExpr(inv.AppliedAmountMinor.As("applied_minor")).
		ColumnExpr(
			"(?) AS reflected_minor",
			s.reflectedMemos(dba, req.ConnectionID, &buncolgen.InvoiceColumns.TotalAmountMinor),
		).
		ColumnExpr(inv.Status.Qualified()+" = ? AS voided", invoice.StatusVoided).
		ColumnExpr(inv.Status.As("state")).
		Join(joinOn(
			buncolgen.CustomerTable,
			buncolgen.CustomerTable.Alias,
			cus.ID.EqColumn(inv.CustomerID),
			cus.OrganizationID.EqColumn(inv.OrganizationID),
			cus.BusinessUnitID.EqColumn(inv.BusinessUnitID),
		)).
		Apply(buncolgen.InvoiceApplyTenant(req.TenantInfo)).
		Where(inv.ID.In(), bun.List(req.ObjectIDs))
}

func (s *driftSource) paymentStates(
	dba bun.IDB,
	req *repositories.ListAccountingDriftStatesRequest,
) *bun.SelectQuery {
	cp := buncolgen.PaymentColumns
	cus := buncolgen.CustomerColumns
	return dba.NewSelect().
		Model((*customerpayment.Payment)(nil)).
		ColumnExpr(cp.ID.As("object_id")).
		ColumnExpr(cp.ReferenceNumber.As("number")).
		ColumnExpr(cp.CustomerID.As("party_id")).
		ColumnExpr(cus.Name.As("party_name")).
		ColumnExpr(cp.CurrencyCode.As("currency_code")).
		ColumnExpr(cp.AmountMinor.As("amount_minor")).
		ColumnExpr(cp.UnappliedAmountMinor.As("open_minor")).
		ColumnExpr(cp.AppliedAmountMinor.As("applied_minor")).
		ColumnExpr(cp.Status.Qualified()+" = ? AS voided", customerpayment.StatusReversed).
		ColumnExpr(cp.Status.As("state")).
		Join(joinOn(
			buncolgen.CustomerTable,
			buncolgen.CustomerTable.Alias,
			cus.ID.EqColumn(cp.CustomerID),
			cus.OrganizationID.EqColumn(cp.OrganizationID),
			cus.BusinessUnitID.EqColumn(cp.BusinessUnitID),
		)).
		Apply(buncolgen.PaymentApplyTenant(req.TenantInfo)).
		Where(cp.ID.In(), bun.List(req.ObjectIDs))
}

func (s *driftSource) applicationStates(
	dba bun.IDB,
	req *repositories.ListAccountingDriftStatesRequest,
) *bun.SelectQuery {
	cma := buncolgen.CreditMemoApplicationColumns
	inv := buncolgen.InvoiceColumns
	cus := buncolgen.CustomerColumns
	return dba.NewSelect().
		Model((*customerpayment.CreditMemoApplication)(nil)).
		ColumnExpr(cma.ID.As("object_id")).
		ColumnExpr(inv.Number.As("number")).
		ColumnExpr(inv.CustomerID.As("party_id")).
		ColumnExpr(cus.Name.As("party_name")).
		ColumnExpr(inv.CurrencyCode.As("currency_code")).
		ColumnExpr(cma.AppliedAmountMinor.As("amount_minor")).
		ColumnExpr(cma.AppliedAmountMinor.As("applied_minor")).
		ColumnExpr(
			cma.Status.Qualified()+" = ? AS voided",
			customerpayment.CreditApplicationStatusUnapplied,
		).
		ColumnExpr(cma.Status.As("state")).
		Join(joinOn(
			buncolgen.InvoiceTable,
			buncolgen.InvoiceTable.Alias,
			inv.ID.EqColumn(cma.CreditMemoInvoiceID),
			inv.OrganizationID.EqColumn(cma.OrganizationID),
			inv.BusinessUnitID.EqColumn(cma.BusinessUnitID),
		)).
		Join(joinOn(
			buncolgen.CustomerTable,
			buncolgen.CustomerTable.Alias,
			cus.ID.EqColumn(inv.CustomerID),
			cus.OrganizationID.EqColumn(inv.OrganizationID),
			cus.BusinessUnitID.EqColumn(inv.BusinessUnitID),
		)).
		Apply(buncolgen.CreditMemoApplicationApplyTenant(req.TenantInfo)).
		Where(cma.ID.In(), bun.List(req.ObjectIDs))
}

func settlementVoided(
	objectType accountingsync.SyncObjectType,
	status string,
	voided, paid string,
) (expr string, args []any) {
	if objectType.IsBillPayment() {
		return status + " <> ? AS voided", []any{paid}
	}
	return status + " = ? AS voided", []any{voided}
}

func (s *driftSource) carrierStates(
	dba bun.IDB,
	req *repositories.ListAccountingDriftStatesRequest,
) *bun.SelectQuery {
	cs := buncolgen.CarrierSettlementColumns
	carr := buncolgen.CarrierColumns
	voided, args := settlementVoided(
		req.ObjectType,
		cs.Status.Qualified(),
		string(carriersettlement.StatusVoided),
		string(carriersettlement.StatusPaid),
	)
	return dba.NewSelect().
		Model((*carriersettlement.CarrierSettlement)(nil)).
		ColumnExpr(cs.ID.As("object_id")).
		ColumnExpr(cs.SettlementNumber.As("number")).
		ColumnExpr(cs.CarrierID.As("party_id")).
		ColumnExpr(carr.Name.As("party_name")).
		ColumnExpr(cs.CurrencyCode.As("currency_code")).
		ColumnExpr("ABS("+cs.NetPayableMinor.Qualified()+") AS amount_minor").
		ColumnExpr(voided, args...).
		ColumnExpr(cs.Status.As("state")).
		Join(joinOn(
			buncolgen.CarrierTable,
			buncolgen.CarrierTable.Alias,
			carr.ID.EqColumn(cs.CarrierID),
			carr.OrganizationID.EqColumn(cs.OrganizationID),
			carr.BusinessUnitID.EqColumn(cs.BusinessUnitID),
		)).
		Apply(buncolgen.CarrierSettlementApplyTenant(req.TenantInfo)).
		Where(cs.ID.In(), bun.List(req.ObjectIDs))
}

func (s *driftSource) driverStates(
	dba bun.IDB,
	req *repositories.ListAccountingDriftStatesRequest,
) *bun.SelectQuery {
	ds := buncolgen.SettlementColumns
	wrk := buncolgen.WorkerColumns
	voided, args := settlementVoided(
		req.ObjectType,
		ds.Status.Qualified(),
		string(driversettlement.StatusVoided),
		string(driversettlement.StatusPaid),
	)
	return dba.NewSelect().
		Model((*driversettlement.Settlement)(nil)).
		ColumnExpr(ds.ID.As("object_id")).
		ColumnExpr(ds.SettlementNumber.As("number")).
		ColumnExpr(ds.WorkerID.As("party_id")).
		ColumnExpr(
			"TRIM(CONCAT("+wrk.FirstName.Qualified()+", ' ', "+
				wrk.LastName.Qualified()+")) AS party_name",
		).
		ColumnExpr(ds.CurrencyCode.As("currency_code")).
		ColumnExpr("ABS("+ds.NetPayMinor.Qualified()+") AS amount_minor").
		ColumnExpr(voided, args...).
		ColumnExpr(ds.Status.As("state")).
		Join(joinOn(
			buncolgen.WorkerTable,
			buncolgen.WorkerTable.Alias,
			wrk.ID.EqColumn(ds.WorkerID),
			wrk.OrganizationID.EqColumn(ds.OrganizationID),
			wrk.BusinessUnitID.EqColumn(ds.BusinessUnitID),
		)).
		Apply(buncolgen.SettlementApplyTenant(req.TenantInfo)).
		Where(ds.ID.In(), bun.List(req.ObjectIDs))
}

func (s *driftSource) balanceDocuments(
	dba bun.IDB,
	req *repositories.ListAccountingDriftBalancesRequest,
) *bun.SelectQuery {
	records := buncolgen.AccountingSyncRecordColumns
	inv := buncolgen.InvoiceColumns
	query := dba.NewSelect().
		Model((*accountingsync.AccountingSyncRecord)(nil)).
		Join(joinOn(
			buncolgen.InvoiceTable,
			buncolgen.InvoiceTable.Alias,
			inv.ID.EqColumn(records.ObjectID),
			inv.OrganizationID.EqColumn(records.OrganizationID),
			inv.BusinessUnitID.EqColumn(records.BusinessUnitID),
		)).
		Where(inv.InvoiceDate.Gte(), req.DatedFrom)
	return s.latestSynced(
		query,
		dba,
		req.TenantInfo,
		req.ConnectionID,
		accountingsync.DriftBalanceObjectTypes(),
	)
}

func (s *driftSource) ListBalances(
	ctx context.Context,
	req *repositories.ListAccountingDriftBalancesRequest,
) ([]*repositories.AccountingDriftBalanceLine, error) {
	customers := req.Customers
	if customers <= 0 {
		customers = defaultDriftCustomers
	}
	customers = intutils.Clamp(customers, 1, maxDriftCustomers)

	inv := buncolgen.InvoiceColumns
	cus := buncolgen.CustomerColumns
	records := buncolgen.AccountingSyncRecordColumns
	dba := s.db.DBForContext(ctx)

	ids := make([]pulid.ID, 0, customers)
	page := s.balanceDocuments(dba, req).
		ColumnExpr("DISTINCT " + inv.CustomerID.Qualified())
	if !req.AfterCustomerID.IsNil() {
		page = page.Where(inv.CustomerID.Gt(), req.AfterCustomerID)
	}
	if err := page.
		OrderExpr(inv.CustomerID.Qualified()+" ASC").
		Limit(customers).
		Scan(ctx, &ids); err != nil {
		return nil, fmt.Errorf("list customers to balance: %w", err)
	}

	lines := make([]*repositories.AccountingDriftBalanceLine, 0, len(ids))
	if len(ids) == 0 {
		return lines, nil
	}
	if err := s.balanceDocuments(dba, req).
		ColumnExpr(inv.CustomerID.As("customer_id")).
		ColumnExpr(cus.Name.As("customer_name")).
		ColumnExpr(inv.CurrencyCode.As("currency_code")).
		ColumnExpr(records.ObjectType.As("object_type")).
		ColumnExpr(inv.ID.As("object_id")).
		ColumnExpr(inv.Number.As("number")).
		ColumnExpr(
			inv.BalanceDueMinor.Qualified()+" + (?) AS open_minor",
			s.reflectedMemos(dba, req.ConnectionID, &inv.BalanceDueMinor),
		).
		ColumnExpr(records.ExternalID.As("external_id")).
		Join(joinOn(
			buncolgen.CustomerTable,
			buncolgen.CustomerTable.Alias,
			cus.ID.EqColumn(inv.CustomerID),
			cus.OrganizationID.EqColumn(inv.OrganizationID),
			cus.BusinessUnitID.EqColumn(inv.BusinessUnitID),
		)).
		Where(inv.CustomerID.In(), bun.List(ids)).
		OrderExpr(inv.CustomerID.Qualified()+" ASC").
		OrderExpr(inv.ID.Qualified()+" ASC").
		Scan(ctx, &lines); err != nil {
		return nil, fmt.Errorf("list balances to compare: %w", err)
	}
	return lines, nil
}

func (s *driftSource) ListPendingCustomers(
	ctx context.Context,
	req *repositories.ListAccountingDriftPendingCustomersRequest,
) ([]pulid.ID, error) {
	ids := make([]pulid.ID, 0, len(req.CustomerIDs))
	if len(req.CustomerIDs) == 0 {
		return ids, nil
	}
	cols := buncolgen.AccountingInboundChangeColumns
	if err := s.db.DBForContext(ctx).
		NewSelect().
		Model((*accountingsync.AccountingInboundChange)(nil)).
		ColumnExpr("DISTINCT "+cols.PartyObjectID.Qualified()).
		Apply(buncolgen.AccountingInboundChangeApplyTenant(req.TenantInfo)).
		Where(cols.ConnectionID.Eq(), req.ConnectionID).
		Where(cols.Kind.Eq(), accountingsync.InboundCustomerPayment).
		Where(cols.Status.In(), bun.List([]accountingsync.InboundChangeStatus{
			accountingsync.InboundStatusDetected,
			accountingsync.InboundStatusProposed,
		})).
		Where(cols.PartyObjectID.In(), bun.List(req.CustomerIDs)).
		Scan(ctx, &ids); err != nil {
		return nil, fmt.Errorf("list customers with payments waiting: %w", err)
	}
	return ids, nil
}
