package collectionrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/factoringsql"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/receivablesql"
	"github.com/emoss08/trenova/internal/modules/billing/domain/collection"
	"github.com/emoss08/trenova/internal/modules/billing/domain/collection/collectiondb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/collectionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
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
			before = keel.DynRow(cact.CreatedAt, cact.ID).Lt(keel.DynRow(req.Before.CreatedAt, req.Before.ID))
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

var worklistSQL = `
	WITH open_invoices AS (
		SELECT inv.customer_id,
			SUM(inv.total_amount_minor - inv.applied_amount_minor) AS open_minor,
			SUM(CASE WHEN inv.due_date IS NOT NULL AND inv.due_date < $1
				THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END) AS overdue_minor,
			MAX(CASE WHEN inv.due_date IS NOT NULL AND inv.due_date < $1
				THEN (($1 - inv.due_date) / 86400)::INT ELSE 0 END) AS oldest_days,
			COUNT(*) AS open_count,
			COUNT(*) FILTER (WHERE inv.dispute_status = 'Disputed') AS disputed_count
		FROM invoices AS inv
		WHERE inv.organization_id = $2
			AND inv.business_unit_id = $3
			AND ` + receivablesql.OpenInvoice + `
			AND NOT EXISTS (
				SELECT 1 FROM factoring_companies AS fac
				LEFT JOIN factoring_agreements AS agr
					ON agr.id = inv.factoring_agreement_id
					AND agr.organization_id = inv.organization_id
					AND agr.business_unit_id = inv.business_unit_id
				WHERE fac.id = inv.factoring_company_id
					AND fac.organization_id = inv.organization_id
					AND fac.business_unit_id = inv.business_unit_id
					AND COALESCE(agr.recourse = 'NonRecourse', fac.non_recourse)
					AND NOT ` + factoringsql.ChargedBackInvoice() + `
			)
		GROUP BY inv.customer_id
	)
	SELECT
		cus.id AS customer_id,
		cus.name AS customer_name,
		cus.code AS customer_code,
		COALESCE(cbp.credit_status::TEXT, 'Active') AS credit_status,
		COALESCE(cbp.billing_currency, 'USD') AS currency_code,
		COALESCE(oi.open_minor, 0) AS open_minor,
		COALESCE(oi.overdue_minor, 0) AS overdue_minor,
		COALESCE(oi.oldest_days, 0) AS oldest_days,
		COALESCE(oi.open_count, 0) AS open_count,
		COALESCE(oi.disputed_count, 0) AS disputed_count,
		acc.collector_id,
		COALESCE(col.name, '') AS collector_name,
		acc.next_follow_up_on,
		recent.created_at AS last_activity_at,
		COALESCE(recent.kind, '') AS last_activity_kind,
		COALESCE(recent.summary, '') AS last_activity_summary,
		prm.id AS promise_id,
		prm.promised_amount_minor,
		prm.currency_code AS promise_currency_code,
		prm.promised_on,
		prm.created_at AS promise_made_at,
		COALESCE(pby.name, '') AS promise_made_by_name,
		CASE WHEN prm.id IS NULL THEN 0 ELSE COALESCE((
			SELECT SUM(cp.amount_minor)
			FROM customer_payments AS cp
			WHERE cp.organization_id = cus.organization_id
				AND cp.business_unit_id = cus.business_unit_id
				AND cp.customer_id = cus.id
				AND cp.status = 'Posted'
				AND cp.created_at >= prm.created_at
		), 0) END AS paid_since_promise
	FROM customers AS cus
	LEFT JOIN open_invoices AS oi ON oi.customer_id = cus.id
	LEFT JOIN customer_billing_profiles AS cbp
		ON cbp.customer_id = cus.id
		AND cbp.organization_id = cus.organization_id
		AND cbp.business_unit_id = cus.business_unit_id
	LEFT JOIN collection_accounts AS acc
		ON acc.customer_id = cus.id
		AND acc.organization_id = cus.organization_id
		AND acc.business_unit_id = cus.business_unit_id
	LEFT JOIN users AS col ON col.id = acc.collector_id
	LEFT JOIN collection_activities AS prm
		ON prm.id = acc.promise_activity_id
		AND prm.organization_id = acc.organization_id
		AND prm.business_unit_id = acc.business_unit_id
	LEFT JOIN users AS pby ON pby.id = prm.created_by_id
	LEFT JOIN LATERAL (
		SELECT a.created_at, a.kind, a.summary
		FROM collection_activities AS a
		WHERE a.organization_id = cus.organization_id
			AND a.business_unit_id = cus.business_unit_id
			AND a.customer_id = cus.id
		ORDER BY a.created_at DESC, a.id DESC
		LIMIT 1
	) AS recent ON TRUE
	WHERE cus.organization_id = $2
		AND cus.business_unit_id = $3`

const worklistFilterSQL = `
		AND ($5::text IS NOT NULL
			OR COALESCE(oi.overdue_minor, 0) > 0
			OR (acc.id IS NOT NULL AND COALESCE(oi.open_minor, 0) > 0))
		AND ($5::text IS NULL OR cus.id = $5)
		AND ($6::text IS NULL OR acc.collector_id = $6)
		AND (NOT $7::boolean OR acc.collector_id IS NULL)
		AND ($8::text IS NULL OR cus.name ILIKE $8 OR cus.code ILIKE $8)
	ORDER BY COALESCE(oi.overdue_minor, 0) DESC, cus.id
	LIMIT $4`

var worklistQuerySQL = worklistSQL + worklistFilterSQL

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
	tenantOrg   pulid.ID
	tenantBU    pulid.ID
	customerID  *string
	collectorID *string
	unassigned  bool
	pattern     *string
	limit       int
}

func (r *repository) readWorklist(
	ctx context.Context,
	q *worklistQuery,
) ([]*worklistRecord, error) {
	records, err := keel.QueryAll(
		ctx,
		r.db.Keel(ctx),
		pgx.RowToAddrOfStructByName[worklistRecord],
		worklistQuerySQL,
		q.asOf,
		q.tenantOrg.String(),
		q.tenantBU.String(),
		q.limit,
		q.customerID,
		q.collectorID,
		q.unassigned,
		q.pattern,
	)
	if err != nil {
		return nil, fmt.Errorf("read collections worklist: %w", err)
	}
	return records, nil
}

func (r *repository) ListWorklist(
	ctx context.Context,
	req *collectionports.WorklistRequest,
) (*collectionports.WorklistPage, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*collectionports.WorklistPage, error) {
		q := worklistQuery{
			asOf:      req.AsOf,
			tenantOrg: req.TenantInfo.OrgID,
			tenantBU:  req.TenantInfo.BuID,
			limit:     MaxWorklistAccounts,
		}
		switch req.Scope {
		case collectionports.CollectorScopeMine:
			collector := req.TenantInfo.UserID.String()
			q.collectorID = &collector
		case collectionports.CollectorScopeUnassigned:
			q.unassigned = true
		case collectionports.CollectorScopeAll:
		}
		if search := strings.TrimSpace(req.Search); search != "" {
			pattern := "%" + escapeLike(search) + "%"
			q.pattern = &pattern
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
			asOf:      asOf,
			tenantOrg: req.TenantInfo.OrgID,
			tenantBU:  req.TenantInfo.BuID,
			limit:     1,
		}
		customerID := req.CustomerID.String()
		q.customerID = &customerID
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
			keel.DynOf(cp.CreatedAt).Gte(prm.CreatedAt),
		))
		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), acc.
			Join(prm, prm.ID.EqCol(acc.PromiseActivityID)).
			Pick(acc.CustomerID).
			Where(
				acc.CustomerID.In(req.CustomerIDs...),
				keel.DynOf(prm.PromisedOn).Add(86400).Gt(req.AsOf),
				keel.DynOf(prm.PromisedAmountMinor).Gt(keel.DynOf(paid).Coalesce(0)),
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
