package accountsreceivablerepository

import (
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/pkg/pagination"
)

type openAsOfWindow struct {
	tenantInfo pagination.TenantInfo
	asOf       int64
	step       int64
	points     int
}

func seriesWindow(req customerledgerports.ListARSeriesRequest) openAsOfWindow {
	return openAsOfWindow{
		tenantInfo: req.TenantInfo,
		asOf:       req.AsOfDate,
		step:       secondsPerWeek,
		points:     req.Weeks,
	}
}

func instantWindow(tenantInfo pagination.TenantInfo, asOf int64) openAsOfWindow {
	return openAsOfWindow{tenantInfo: tenantInfo, asOf: asOf, step: secondsPerWeek, points: 1}
}

func (w openAsOfWindow) start() int64 {
	return w.asOf - int64(w.points-1)*w.step
}

func (w openAsOfWindow) args() []any {
	org, bu := tenantArgs(w.tenantInfo)
	return []any{
		w.asOf,
		w.step,
		w.points,
		org,
		bu,
		w.start(),
		customerledger.SourceFactoringItem,
	}
}

const openAsOfCTEs = `
		points AS (
			SELECT n, $1::BIGINT - n * $2::BIGINT AS period_end
			FROM generate_series(0, $3::INT - 1) AS n
		),
		candidates AS (
			SELECT inv.id, inv.organization_id, inv.business_unit_id, inv.invoice_date, inv.due_date, inv.total_amount_minor
			FROM invoices inv
			WHERE inv.organization_id = $4
			  AND inv.business_unit_id = $5
			  AND inv.invoice_date <= $1` + openInvoicePredicate + `
			UNION
			SELECT inv.id, inv.organization_id, inv.business_unit_id, inv.invoice_date, inv.due_date, inv.total_amount_minor
			FROM (
				SELECT cpa.invoice_id
				FROM customer_payment_applications cpa
				WHERE cpa.organization_id = $4
				  AND cpa.business_unit_id = $5
				  AND cpa.created_at > $6
				UNION
				SELECT cpa.invoice_id
				FROM customer_payments cp
				JOIN customer_payment_applications cpa
				  ON cpa.customer_payment_id = cp.id
				 AND cpa.organization_id = cp.organization_id
				 AND cpa.business_unit_id = cp.business_unit_id
				WHERE cp.organization_id = $4
				  AND cp.business_unit_id = $5
				  AND cp.reversed_at > $6
				UNION
				SELECT cma.invoice_id
				FROM credit_memo_applications cma
				WHERE cma.organization_id = $4
				  AND cma.business_unit_id = $5
				  AND cma.created_at > $6
				UNION
				SELECT cma.invoice_id
				FROM credit_memo_applications cma
				WHERE cma.organization_id = $4
				  AND cma.business_unit_id = $5
				  AND cma.unapplied_at > $6
				UNION
				SELECT cle.related_invoice_id
				FROM customer_ledger_entries cle
				WHERE cle.organization_id = $4
				  AND cle.business_unit_id = $5
				  AND cle.source_object_type = $7
				  AND cle.related_invoice_id IS NOT NULL
				  AND cle.created_at > $6
			) moved
			JOIN invoices inv
			  ON inv.id = moved.invoice_id
			 AND inv.organization_id = $4
			 AND inv.business_unit_id = $5
			WHERE inv.status = 'Posted'
			  AND inv.bill_type IN ('Invoice', 'DebitMemo')
			  AND inv.invoice_date <= $1
		),
		settlements AS (
			SELECT c.id AS invoice_id, c.due_date, cpa.created_at AS settled_at, cp.reversed_at AS released_at,
				cpa.applied_amount_minor + cpa.short_pay_amount_minor AS amount
			FROM candidates c
			JOIN customer_payment_applications cpa
			  ON cpa.invoice_id = c.id
			 AND cpa.organization_id = c.organization_id
			 AND cpa.business_unit_id = c.business_unit_id
			JOIN customer_payments cp
			  ON cp.id = cpa.customer_payment_id
			 AND cp.organization_id = cpa.organization_id
			 AND cp.business_unit_id = cpa.business_unit_id
			UNION ALL
			SELECT c.id, c.due_date, cma.created_at, cma.unapplied_at, cma.applied_amount_minor
			FROM candidates c
			JOIN credit_memo_applications cma
			  ON cma.invoice_id = c.id
			 AND cma.organization_id = c.organization_id
			 AND cma.business_unit_id = c.business_unit_id
			UNION ALL
			SELECT c.id, c.due_date, cle.created_at, NULL, -cle.amount_minor
			FROM candidates c
			JOIN customer_ledger_entries cle
			  ON cle.related_invoice_id = c.id
			 AND cle.organization_id = c.organization_id
			 AND cle.business_unit_id = c.business_unit_id
			 AND cle.source_object_type = $7
		),
		changes AS (
			SELECT c.id AS invoice_id, c.due_date, c.invoice_date AS changed_at, c.total_amount_minor AS delta
			FROM candidates c
			UNION ALL
			SELECT s.invoice_id, s.due_date, s.settled_at, -s.amount
			FROM settlements s
			UNION ALL
			SELECT s.invoice_id, s.due_date, GREATEST(s.settled_at, s.released_at), s.amount
			FROM settlements s
			WHERE s.released_at IS NOT NULL
		),
		segments AS (
			SELECT
				ch.invoice_id,
				ch.due_date,
				ch.changed_at AS starts_at,
				LEAD(ch.changed_at) OVER w AS ends_at,
				SUM(SUM(ch.delta)) OVER w AS open_minor
			FROM changes ch
			GROUP BY ch.invoice_id, ch.due_date, ch.changed_at
			WINDOW w AS (PARTITION BY ch.invoice_id ORDER BY ch.changed_at)
		),
		open_items AS (
			SELECT n, seg.invoice_id, seg.due_date, seg.open_minor
			FROM segments seg
			CROSS JOIN LATERAL generate_series(
				CASE
					WHEN seg.ends_at IS NULL THEN 0
					ELSE GREATEST(0, FLOOR(($1::BIGINT - seg.ends_at)::NUMERIC / $2::BIGINT)::INT + 1)
				END,
				LEAST($3::INT - 1, FLOOR(($1::BIGINT - seg.starts_at)::NUMERIC / $2::BIGINT)::INT)
			) AS n
			WHERE seg.open_minor > 0
		)`
