package collectionrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/collection"
	"github.com/emoss08/trenova/internal/modules/billing/ports/collectionports"
	"github.com/emoss08/trenova/pkg/buncolgen/collectioncol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// MaxWorklistAccounts bounds how many customers one worklist read considers.
// The worklist is ranked in Go, where the priority rules live, so it reads
// every account it ranks; past this many, the most overdue are kept.
const MaxWorklistAccounts = 5000

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

// GetAccountForUpdate locks the customer's collections account, or returns a
// new, unsaved one when nobody has worked the customer yet.
func (r *repository) GetAccountForUpdate(
	ctx context.Context,
	req collectionports.GetAccountRequest,
) (*collection.Account, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*collection.Account, error) {
		cols := collectioncol.AccountColumns
		account := new(collection.Account)
		err := r.db.DBForContext(ctx).NewSelect().
			Model(account).
			Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
			Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
			Where(cols.CustomerID.Eq(), req.CustomerID).
			For("UPDATE").
			Scan(ctx)
		if errors.Is(err, sql.ErrNoRows) {
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

// SaveAccount inserts a new account or updates one read with
// GetAccountForUpdate, refusing a stale version.
func (r *repository) SaveAccount(
	ctx context.Context,
	account *collection.Account,
) (*collection.Account, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*collection.Account, error) {
		db := r.db.DBForContext(ctx)
		cols := collectioncol.AccountColumns
		if account.ID.IsNil() {
			if _, err := db.NewInsert().Model(account).Returning("*").Exec(ctx); err != nil {
				return nil, dberror.MapRetryableTransactionError(err, "Collections account")
			}
			return account, nil
		}

		previous := account.Version
		account.Version++
		result, err := db.NewUpdate().
			Model(account).
			Column(
				cols.CollectorID.Bare(),
				cols.NextFollowUpOn.Bare(),
				cols.PromiseActivityID.Bare(),
				cols.Version.Bare(),
				cols.UpdatedAt.Bare(),
			).
			WherePK().
			Where(cols.Version.Eq(), previous).
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("save collections account: %w", err)
		}
		if rows, _ := result.RowsAffected(); rows == 0 {
			return nil, errortypes.NewValidationError(
				"version",
				errortypes.ErrVersionMismatch,
				"Someone else updated this account at the same time. Reload it and try again.",
			)
		}
		return account, nil
	})
}

func (r *repository) InsertActivity(ctx context.Context, activity *collection.Activity) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(activity).Exec(ctx); err != nil {
			return fmt.Errorf("insert collections activity: %w", err)
		}
		return nil
	})
}

const activitiesSQL = `
	SELECT cact.*,
		COALESCE(cby.name, '') AS created_by_name,
		COALESCE(col.name, '') AS collector_name,
		COALESCE(inv.number, '') AS invoice_number
	FROM collection_activities AS cact
	LEFT JOIN users AS cby ON cby.id = cact.created_by_id
	LEFT JOIN users AS col ON col.id = cact.collector_id
	LEFT JOIN invoices AS inv
		ON inv.id = cact.invoice_id
		AND inv.organization_id = cact.organization_id
		AND inv.business_unit_id = cact.business_unit_id
	WHERE cact.organization_id = ?
		AND cact.business_unit_id = ?
		AND cact.customer_id = ?
		%s
	ORDER BY cact.created_at DESC, cact.id DESC
	LIMIT ?`

type activityRecord struct {
	collection.Activity `       bun:",extend"`
	CreatedByName       string `bun:"created_by_name"`
	CollectorName       string `bun:"collector_name"`
	InvoiceNumber       string `bun:"invoice_number"`
}

// ListActivities pages a customer's history from newest to oldest.
//
//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
func (r *repository) ListActivities(
	ctx context.Context,
	req collectionports.ListActivitiesRequest,
) ([]*collectionports.ActivityView, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*collectionports.ActivityView, error) {
		args := []any{req.TenantInfo.OrgID, req.TenantInfo.BuID, req.CustomerID}
		cursor := ""
		if req.Before != nil {
			cursor = "AND (cact.created_at, cact.id) < (?, ?)"
			args = append(args, req.Before.CreatedAt, req.Before.ID)
		}
		args = append(args, req.Limit)

		records := make([]*activityRecord, 0, req.Limit)
		if err := r.db.DBForContext(ctx).
			NewRaw(fmt.Sprintf(activitiesSQL, cursor), args...).
			Scan(ctx, &records); err != nil {
			return nil, fmt.Errorf("list collections activities: %w", err)
		}

		views := make([]*collectionports.ActivityView, 0, len(records))
		for _, rec := range records {
			activity := rec.Activity
			views = append(views, &collectionports.ActivityView{
				Activity:      &activity,
				CreatedByName: rec.CreatedByName,
				CollectorName: rec.CollectorName,
				InvoiceNumber: rec.InvoiceNumber,
			})
		}
		return views, nil
	})
}

// worklistSQL reads every customer collections should see: anyone with an
// overdue invoice, plus anyone already being worked. %s narrows by collector,
// customer and search.
const worklistSQL = `
	WITH open_invoices AS (
		SELECT inv.customer_id,
			SUM(inv.total_amount_minor - inv.applied_amount_minor) AS open_minor,
			SUM(CASE WHEN inv.due_date IS NOT NULL AND inv.due_date < ?
				THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END) AS overdue_minor,
			MAX(CASE WHEN inv.due_date IS NOT NULL AND inv.due_date < ?
				THEN ((? - inv.due_date) / 86400)::INT ELSE 0 END) AS oldest_days,
			COUNT(*) AS open_count,
			COUNT(*) FILTER (WHERE inv.dispute_status = 'Disputed') AS disputed_count
		FROM invoices AS inv
		WHERE inv.organization_id = ?
			AND inv.business_unit_id = ?
			AND inv.status = 'Posted'
			AND inv.bill_type IN ('Invoice', 'DebitMemo')
			AND inv.total_amount_minor > inv.applied_amount_minor
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
	WHERE cus.organization_id = ?
		AND cus.business_unit_id = ?
		%s
	ORDER BY COALESCE(oi.overdue_minor, 0) DESC, cus.id
	LIMIT ?`

type worklistRecord struct {
	CustomerID          pulid.ID `bun:"customer_id"`
	CustomerName        string   `bun:"customer_name"`
	CustomerCode        string   `bun:"customer_code"`
	CreditStatus        string   `bun:"credit_status"`
	CurrencyCode        string   `bun:"currency_code"`
	OpenMinor           int64    `bun:"open_minor"`
	OverdueMinor        int64    `bun:"overdue_minor"`
	OldestDays          int      `bun:"oldest_days"`
	OpenCount           int      `bun:"open_count"`
	DisputedCount       int      `bun:"disputed_count"`
	CollectorID         pulid.ID `bun:"collector_id"`
	CollectorName       string   `bun:"collector_name"`
	NextFollowUpOn      *int64   `bun:"next_follow_up_on"`
	LastActivityAt      *int64   `bun:"last_activity_at"`
	LastActivityKind    string   `bun:"last_activity_kind"`
	LastActivitySummary string   `bun:"last_activity_summary"`
	PromiseID           pulid.ID `bun:"promise_id"`
	PromisedAmountMinor *int64   `bun:"promised_amount_minor"`
	PromiseCurrencyCode *string  `bun:"promise_currency_code"`
	PromisedOn          *int64   `bun:"promised_on"`
	PromiseMadeAt       *int64   `bun:"promise_made_at"`
	PromiseMadeByName   string   `bun:"promise_made_by_name"`
	PaidSincePromise    int64    `bun:"paid_since_promise"`
}

type worklistQuery struct {
	asOf       int64
	tenantOrg  pulid.ID
	tenantBU   pulid.ID
	conditions []string
	args       []any
	limit      int
}

func (r *repository) readWorklist(
	ctx context.Context,
	q *worklistQuery,
) ([]*worklistRecord, error) {
	args := []any{q.asOf, q.asOf, q.asOf, q.tenantOrg, q.tenantBU, q.tenantOrg, q.tenantBU}
	args = append(args, q.args...)
	args = append(args, q.limit)

	filter := ""
	if len(q.conditions) > 0 {
		filter = "AND " + strings.Join(q.conditions, " AND ")
	}
	records := make([]*worklistRecord, 0, 64)
	if err := r.db.DBForContext(ctx).
		NewRaw(fmt.Sprintf(worklistSQL, filter), args...).
		Scan(ctx, &records); err != nil {
		return nil, fmt.Errorf("read collections worklist: %w", err)
	}
	return records, nil
}

// ListWorklist reads every account in scope. Ranking, the due filter, the
// summary and paging are the service's, since they follow the priority rules.
func (r *repository) ListWorklist(
	ctx context.Context,
	req *collectionports.WorklistRequest,
) (*collectionports.WorklistPage, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*collectionports.WorklistPage, error) {
		q := worklistQuery{
			asOf:      req.AsOf,
			tenantOrg: req.TenantInfo.OrgID,
			tenantBU:  req.TenantInfo.BuID,
			conditions: []string{
				"(COALESCE(oi.overdue_minor, 0) > 0 OR (acc.id IS NOT NULL AND COALESCE(oi.open_minor, 0) > 0))",
			},
			limit: MaxWorklistAccounts,
		}
		switch req.Scope {
		case collectionports.CollectorScopeMine:
			q.conditions = append(q.conditions, "acc.collector_id = ?")
			q.args = append(q.args, req.TenantInfo.UserID)
		case collectionports.CollectorScopeUnassigned:
			q.conditions = append(q.conditions, "acc.collector_id IS NULL")
		case collectionports.CollectorScopeAll:
		}
		if search := strings.TrimSpace(req.Search); search != "" {
			q.conditions = append(q.conditions, "(cus.name ILIKE ? OR cus.code ILIKE ?)")
			pattern := "%" + escapeLike(search) + "%"
			q.args = append(q.args, pattern, pattern)
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

// GetWorklistRow reads one customer as the worklist would show it, whether or
// not it is overdue.
func (r *repository) GetWorklistRow(
	ctx context.Context,
	req collectionports.GetAccountRequest,
	asOf int64,
) (*collectionports.WorklistRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*collectionports.WorklistRow, error) {
		records, err := r.readWorklist(ctx, &worklistQuery{
			asOf:       asOf,
			tenantOrg:  req.TenantInfo.OrgID,
			tenantBU:   req.TenantInfo.BuID,
			conditions: []string{"cus.id = ?"},
			args:       []any{req.CustomerID},
			limit:      1,
		})
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

const awaitingPromiseSQL = `
	SELECT acc.customer_id
	FROM collection_accounts AS acc
	JOIN collection_activities AS prm
		ON prm.id = acc.promise_activity_id
		AND prm.organization_id = acc.organization_id
		AND prm.business_unit_id = acc.business_unit_id
	WHERE acc.organization_id = ?
		AND acc.business_unit_id = ?
		AND acc.customer_id IN (?)
		AND prm.promised_on + 86400 > ?
		AND prm.promised_amount_minor > COALESCE((
			SELECT SUM(cp.amount_minor)
			FROM customer_payments AS cp
			WHERE cp.organization_id = acc.organization_id
				AND cp.business_unit_id = acc.business_unit_id
				AND cp.customer_id = acc.customer_id
				AND cp.status = 'Posted'
				AND cp.created_at >= prm.created_at
		), 0)`

// ListAwaitingPromise names the customers waiting on an open promise to pay.
//
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
		ids := make([]pulid.ID, 0, len(req.CustomerIDs))
		if err := r.db.DBForContext(ctx).
			NewRaw(awaitingPromiseSQL,
				req.TenantInfo.OrgID, req.TenantInfo.BuID, bun.List(req.CustomerIDs), req.AsOf).
			Scan(ctx, &ids); err != nil {
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
