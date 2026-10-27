package collectionrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/receivablesql"
	"github.com/emoss08/trenova/internal/modules/billing/domain/collection"
	"github.com/emoss08/trenova/internal/modules/billing/domain/collection/collectiondb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/collectionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const MaxWorklistAccounts = 5000

var accountErrors = dberror.Mapping{Entity: "Collections account"}

var activityErrors = dberror.Mapping{Entity: "Collections activity"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) collectionports.Repository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.collection-repository")}
}

func (r *repository) GetAccountForUpdate(
	ctx context.Context,
	req collectionports.GetAccountRequest,
) (*collection.Account, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*collection.Account, error) {
		acc := collectiondb.Accounts
		account, err := acc.Where(acc.CustomerID.Eq(req.CustomerID)).
			Lock(keel.ForUpdate).
			One(ctx, r.db.Keel(ctx))
		if dberror.IsNotFoundError(err) {
			return &collection.Account{
				OrganizationID: req.TenantInfo.OrgID,
				BusinessUnitID: req.TenantInfo.BuID,
				CustomerID:     req.CustomerID,
			}, nil
		}
		if err != nil {
			return nil, fmt.Errorf("lock collections account: %w", err)
		}
		return account, nil
	})
}

func (r *repository) SaveAccount(
	ctx context.Context,
	account *collection.Account,
) (*collection.Account, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*collection.Account, error) {
		acc := collectiondb.Accounts
		db := r.db.Keel(ctx)
		if account.ID.IsNil() {
			if err := acc.Create(ctx, db, account); err != nil {
				if dberror.IsRetryableTransactionError(err) {
					return nil, dberror.MapRetryableTransactionError(err, "Collections account")
				}
				return nil, accountErrors.Map(err, account.ID.String())
			}
			return account, nil
		}

		err := acc.Save(
			ctx,
			db,
			account,
			acc.CollectorID,
			acc.NextFollowUpOn,
			acc.PromiseActivityID,
		)
		if dberror.IsVersionConflict(err) {
			return nil, errortypes.NewValidationError(
				"version",
				errortypes.ErrVersionMismatch,
				"Someone else updated this account at the same time. Reload it and try again.",
			)
		}
		if err != nil {
			return nil, fmt.Errorf("save collections account: %w", err)
		}
		return account, nil
	})
}

func (r *repository) InsertActivity(ctx context.Context, activity *collection.Activity) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := collectiondb.Activities.Create(ctx, r.db.Keel(ctx), activity); err != nil {
			return activityErrors.Map(err, activity.ID.String())
		}
		return nil
	})
}

type activityRecord struct {
	ID                  pulid.ID
	OrganizationID      pulid.ID
	BusinessUnitID      pulid.ID
	CustomerID          pulid.ID
	Kind                collection.ActivityKind
	Summary             *string
	ContactName         *string
	InvoiceID           pulid.ID
	PromisedAmountMinor *int64
	PromisedOn          *int64
	CurrencyCode        string
	FollowUpOn          *int64
	CollectorID         pulid.ID
	CreatedByID         pulid.ID
	CreatedAt           int64
	CreatedByName       string
	CollectorName       string
	InvoiceNumber       string
}

func (rec *activityRecord) view() *collectionports.ActivityView {
	return &collectionports.ActivityView{
		Activity: &collection.Activity{
			ID:                  rec.ID,
			OrganizationID:      rec.OrganizationID,
			BusinessUnitID:      rec.BusinessUnitID,
			CustomerID:          rec.CustomerID,
			Kind:                rec.Kind,
			Summary:             deref(rec.Summary),
			ContactName:         deref(rec.ContactName),
			InvoiceID:           rec.InvoiceID,
			PromisedAmountMinor: rec.PromisedAmountMinor,
			PromisedOn:          rec.PromisedOn,
			CurrencyCode:        rec.CurrencyCode,
			FollowUpOn:          rec.FollowUpOn,
			CollectorID:         rec.CollectorID,
			CreatedByID:         rec.CreatedByID,
			CreatedAt:           rec.CreatedAt,
		},
		CreatedByName: rec.CreatedByName,
		CollectorName: rec.CollectorName,
		InvoiceNumber: rec.InvoiceNumber,
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
func (r *repository) ListActivities(
	ctx context.Context,
	req collectionports.ListActivitiesRequest,
) ([]*collectionports.ActivityView, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*collectionports.ActivityView, error) {
		cact := collectiondb.Activities
		cby := tenantdb.Users.As("cby")
		col := tenantdb.Users.As("col")
		inv := invoicedb.Invoices.As("inv")

		var before keel.Pred
		if req.Before != nil {
			before = keel.Tuple(cact.CreatedAt, cact.ID).
				Lt(keel.Tuple(req.Before.CreatedAt, req.Before.ID))
		}

		records, err := keel.Into[activityRecord](ctx, r.db.Keel(ctx), cact.
			LeftJoin(cby, cby.ID.EqCol(cact.CreatedByID)).
			LeftJoin(col, col.ID.EqCol(cact.CollectorID)).
			LeftJoin(inv, inv.ID.EqCol(cact.InvoiceID)).
			Pick(
				cact.ID,
				cact.OrganizationID,
				cact.BusinessUnitID,
				cact.CustomerID,
				cact.Kind,
				cact.Summary,
				cact.ContactName,
				cact.InvoiceID,
				cact.PromisedAmountMinor,
				cact.PromisedOn,
				cact.CurrencyCode,
				cact.FollowUpOn,
				cact.CollectorID,
				cact.CreatedByID,
				cact.CreatedAt,
				cby.Name.Coalesce(""),
				col.Name.Coalesce(""),
				inv.Number.Coalesce(""),
			).
			Where(cact.CustomerID.Eq(req.CustomerID), before).
			Order(cact.CreatedAt.Desc(), cact.ID.Desc()).
			Limit(req.Limit))
		if err != nil {
			return nil, fmt.Errorf("list collections activities: %w", err)
		}

		views := make([]*collectionports.ActivityView, 0, len(records))
		for i := range records {
			views = append(views, records[i].view())
		}
		return views, nil
	})
}

type worklistRecord struct {
	CustomerID          pulid.ID
	CustomerName        string
	CustomerCode        string
	CreditStatus        string
	CurrencyCode        string
	OpenMinor           int64
	OverdueMinor        int64
	OldestDays          int
	OpenCount           int
	DisputedCount       int
	CollectorID         pulid.ID
	CollectorName       string
	NextFollowUpOn      *int64
	LastActivityAt      *int64
	LastActivityKind    string
	LastActivitySummary string
	PromiseID           pulid.ID
	PromisedAmountMinor *int64
	PromiseCurrencyCode *string
	PromisedOn          *int64
	PromiseMadeAt       *int64
	PromiseMadeByName   string
	PaidSincePromise    int64
}

type worklistQuery struct {
	asOf        int64
	customerID  pulid.ID
	collectorID pulid.ID
	unassigned  bool
	pattern     string
	limit       int
}

func openInvoicesByCustomer(asOf int64) *keel.CTETable {
	inv := invoicedb.Invoices
	open := receivablesql.OpenMinor(inv)
	overdue := keel.And(inv.DueDate.IsNotNull(), inv.DueDate.Lt(asOf))
	daysPastDue := keel.Int64(asOf).Sub(inv.DueDate).Div(keel.Int64(receivablesql.SecondsPerDay))
	return keel.CTE(
		"open_invoices",
		[]string{
			"customer_id",
			"open_minor",
			"overdue_minor",
			"oldest_days",
			"open_count",
			"disputed_count",
		},
		inv.
			Pick(
				inv.CustomerID,
				open.Sum(),
				keel.CaseInt64().When(overdue, open).Else(keel.Int64(0)).Sum(),
				keel.CaseInt64().When(overdue, daysPastDue).Else(keel.Int64(0)).Max(),
				keel.Count(),
				keel.Count().Filter(inv.DisputeStatus.Eq(invoice.DisputeStatusDisputed)),
			).
			Where(receivablesql.Open(inv), keel.Not(receivablesql.SoldWithoutRecourse(inv))).
			Group(inv.CustomerID),
	)
}

func worklistStatement(q *worklistQuery) *keel.Query[*customer.Customer] {
	cus := customerdb.Customers
	oi := openInvoicesByCustomer(q.asOf)
	cbp := customerdb.CustomerBillingProfiles
	acc := collectiondb.Accounts
	col := tenantdb.Users.As("col")
	prm := collectiondb.Activities.As("prm")
	pby := tenantdb.Users.As("pby")
	recent := collectiondb.Activities.As("recent")
	cp := customerpaymentdb.Payments.As("cp")
	paidSincePromise := keel.Scalar(cp.Pick(cp.AmountMinor.Sum()).Where(
		cp.CustomerID.EqCol(cus.ID),
		cp.Status.Eq(customerpayment.StatusPosted),
		keel.Int64Of(cp.CreatedAt).GteExpr(prm.CreatedAt),
	))
	openMinor := keel.NumericOf(oi.Col("open_minor")).Coalesce(0)
	overdueMinor := keel.NumericOf(oi.Col("overdue_minor")).Coalesce(0)

	return cus.With(oi).
		LeftJoin(oi, oi.Col("customer_id").EqCol(cus.ID)).
		LeftJoin(cbp, cbp.CustomerID.EqCol(cus.ID)).
		LeftJoin(acc, acc.CustomerID.EqCol(cus.ID)).
		LeftJoin(col, col.ID.EqCol(acc.CollectorID)).
		LeftJoin(prm, prm.ID.EqCol(acc.PromiseActivityID)).
		LeftJoin(pby, pby.ID.EqCol(prm.CreatedByID)).
		LeftJoinLateral(recent.
			Pick(recent.CreatedAt, recent.Kind, recent.Summary).
			Where(recent.CustomerID.EqCol(cus.ID)).
			Order(recent.CreatedAt.Desc(), recent.ID.Desc()).
			Limit(1)).
		Pick(
			cus.ID,
			cus.Name,
			cus.Code,
			keel.CastText(cbp.CreditStatus).Coalesce("Active"),
			cbp.BillingCurrency.Coalesce("USD"),
			openMinor,
			overdueMinor,
			keel.Int64Of(oi.Col("oldest_days")).Coalesce(0),
			keel.Int64Of(oi.Col("open_count")).Coalesce(0),
			keel.Int64Of(oi.Col("disputed_count")).Coalesce(0),
			acc.CollectorID,
			col.Name.Coalesce(""),
			acc.NextFollowUpOn,
			recent.CreatedAt,
			recent.Kind.Coalesce(""),
			recent.Summary.Coalesce(""),
			prm.ID,
			prm.PromisedAmountMinor,
			prm.CurrencyCode,
			prm.PromisedOn,
			prm.CreatedAt,
			pby.Name.Coalesce(""),
			keel.CaseNumeric().
				When(prm.ID.IsNull(), keel.Numeric(0)).
				Else(keel.NumericOf(paidSincePromise).Coalesce(0)),
		).
		Where(
			keel.When(q.customerID.IsNotNil(), cus.ID.Eq(q.customerID)),
			keel.When(q.customerID.IsNil(), keel.Or(
				overdueMinor.Gt(0),
				keel.And(acc.ID.IsNotNull(), openMinor.Gt(0)),
			)),
			keel.When(q.collectorID.IsNotNil(), acc.CollectorID.Eq(q.collectorID)),
			keel.When(q.unassigned, acc.CollectorID.IsNull()),
			keel.When(q.pattern != "", keel.Or(cus.Name.ILike(q.pattern), cus.Code.ILike(q.pattern))),
		).
		Order(overdueMinor.Desc(), cus.ID.Asc()).
		Limit(q.limit)
}

func (r *repository) readWorklist(
	ctx context.Context,
	q *worklistQuery,
) ([]*worklistRecord, error) {
	records, err := keel.Into[worklistRecord](ctx, r.db.Keel(ctx), worklistStatement(q))
	if err != nil {
		return nil, fmt.Errorf("read collections worklist: %w", err)
	}
	return sliceutils.Pointers(records), nil
}

func (r *repository) ListWorklist(
	ctx context.Context,
	req *collectionports.WorklistRequest,
) (*collectionports.WorklistPage, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*collectionports.WorklistPage, error) {
		q := worklistQuery{
			asOf:  req.AsOf,
			limit: MaxWorklistAccounts,
		}
		switch req.Scope {
		case collectionports.CollectorScopeMine:
			q.collectorID = req.TenantInfo.UserID
		case collectionports.CollectorScopeUnassigned:
			q.unassigned = true
		case collectionports.CollectorScopeAll:
		}
		if search := strings.TrimSpace(req.Search); search != "" {
			q.pattern = "%" + escapeLike(search) + "%"
		}

		records, err := r.readWorklist(ctx, &q)
		if err != nil {
			return nil, err
		}
		rows := make([]*collectionports.WorklistRow, 0, len(records))
		for _, rec := range records {
			rows = append(rows, rec.row())
		}
		return &collectionports.WorklistPage{Rows: rows, Total: len(rows)}, nil
	})
}

func (r *repository) GetWorklistRow(
	ctx context.Context,
	req collectionports.GetAccountRequest,
	asOf int64,
) (*collectionports.WorklistRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*collectionports.WorklistRow, error) {
		q := worklistQuery{
			asOf:       asOf,
			customerID: req.CustomerID,
			limit:      1,
		}
		records, err := r.readWorklist(ctx, &q)
		if err != nil {
			return nil, err
		}
		if len(records) == 0 {
			return nil, errortypes.NewNotFoundError("Customer not found")
		}
		return records[0].row(), nil
	})
}

func (rec *worklistRecord) row() *collectionports.WorklistRow {
	row := &collectionports.WorklistRow{
		CustomerID:           rec.CustomerID,
		CustomerName:         rec.CustomerName,
		CustomerCode:         rec.CustomerCode,
		CreditStatus:         rec.CreditStatus,
		CurrencyCode:         rec.CurrencyCode,
		TotalOpenMinor:       rec.OpenMinor,
		OverdueMinor:         rec.OverdueMinor,
		OldestDaysPastDue:    rec.OldestDays,
		OpenInvoiceCount:     rec.OpenCount,
		DisputedInvoiceCount: rec.DisputedCount,
		CollectorID:          rec.CollectorID,
		CollectorName:        rec.CollectorName,
		NextFollowUpOn:       rec.NextFollowUpOn,
		LastActivityAt:       rec.LastActivityAt,
		LastActivityKind:     collection.ActivityKind(rec.LastActivityKind),
		LastActivitySummary:  rec.LastActivitySummary,
	}
	if rec.PromiseID.IsNotNil() && rec.PromisedAmountMinor != nil && rec.PromisedOn != nil &&
		rec.PromiseMadeAt != nil {
		currency := ""
		if rec.PromiseCurrencyCode != nil {
			currency = *rec.PromiseCurrencyCode
		}
		row.Promise = &collectionports.Promise{
			ActivityID:     rec.PromiseID,
			AmountMinor:    *rec.PromisedAmountMinor,
			CurrencyCode:   currency,
			PromisedOn:     *rec.PromisedOn,
			MadeAt:         *rec.PromiseMadeAt,
			MadeByName:     rec.PromiseMadeByName,
			PaidSinceMinor: rec.PaidSincePromise,
		}
	}
	return row
}

//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
func (r *repository) ListAwaitingPromise(
	ctx context.Context,
	req collectionports.AwaitingPromiseRequest,
) (map[pulid.ID]bool, error) {
	awaiting := make(map[pulid.ID]bool, len(req.CustomerIDs))
	if len(req.CustomerIDs) == 0 {
		return awaiting, nil
	}
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]bool, error) {
		acc := collectiondb.Accounts
		prm := collectiondb.Activities.As("prm")
		cp := customerpaymentdb.Payments.As("cp")
		paid := keel.Scalar(cp.Pick(cp.AmountMinor.Sum()).Where(
			cp.CustomerID.EqCol(acc.CustomerID),
			cp.Status.Eq(customerpayment.StatusPosted),
			keel.Int64Of(cp.CreatedAt).GteExpr(prm.CreatedAt),
		))
		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), acc.
			Join(prm, prm.ID.EqCol(acc.PromiseActivityID)).
			Pick(acc.CustomerID).
			Where(
				acc.CustomerID.In(req.CustomerIDs...),
				prm.PromisedOn.Add(keel.Int64(86400)).Gt(req.AsOf),
				keel.NumericOf(prm.PromisedAmountMinor).GtExpr(keel.NumericOf(paid).Coalesce(0)),
			))
		if err != nil {
			return nil, fmt.Errorf("list customers awaiting a promise: %w", err)
		}
		for _, id := range ids {
			awaiting[id] = true
		}
		return awaiting, nil
	})
}

func escapeLike(value string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(value)
}
