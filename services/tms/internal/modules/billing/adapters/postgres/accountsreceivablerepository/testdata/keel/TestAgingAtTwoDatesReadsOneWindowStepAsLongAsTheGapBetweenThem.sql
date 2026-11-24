WITH series (n) AS (
  SELECT generate_series($1::bigint, $16::bigint) AS n
), points (n, period_end) AS (
  SELECT s.n, ($17::bigint - (s.n * $18::bigint))
  FROM series s
), events (invoice_id, settled_at, released_at, amount) AS ((
  SELECT cpa.invoice_id, cpa.created_at, cp.reversed_at, (cpa.applied_amount_minor + cpa.short_pay_amount_minor)
  FROM customer_payment_applications cpa
  JOIN customer_payments cp ON cp.organization_id = $19 AND cp.business_unit_id = $20 AND cp.id = cpa.customer_payment_id
  WHERE cpa.organization_id = $19 AND cpa.business_unit_id = $20 AND cpa.created_at > $21 AND cpa.created_at <= $22
) UNION ALL (
  SELECT cpa2.invoice_id, cpa2.created_at, cp2.reversed_at, (cpa2.applied_amount_minor + cpa2.short_pay_amount_minor)
  FROM customer_payments cp2
  JOIN customer_payment_applications cpa2 ON cpa2.organization_id = $19 AND cpa2.business_unit_id = $20 AND cpa2.customer_payment_id = cp2.id
  WHERE cp2.organization_id = $19 AND cp2.business_unit_id = $20 AND cp2.reversed_at > $23 AND cp2.reversed_at <= $24 AND cpa2.created_at <= $25
) UNION ALL (
  SELECT cma.invoice_id, cma.created_at, cma.unapplied_at, cma.applied_amount_minor
  FROM credit_memo_applications cma
  WHERE cma.organization_id = $19 AND cma.business_unit_id = $20 AND cma.created_at > $26 AND cma.created_at <= $27
) UNION ALL (
  SELECT cma2.invoice_id, cma2.created_at, cma2.unapplied_at, cma2.applied_amount_minor
  FROM credit_memo_applications cma2
  WHERE cma2.organization_id = $19 AND cma2.business_unit_id = $20 AND cma2.unapplied_at > $28 AND cma2.unapplied_at <= $29 AND cma2.created_at <= $30
) UNION ALL (
  SELECT cle.related_invoice_id, cle.created_at, NULL::bigint, (-cle.amount_minor)
  FROM customer_ledger_entries cle
  WHERE cle.organization_id = $19 AND cle.business_unit_id = $20 AND cle.source_object_type = 'FactoringSubmissionItem' AND cle.related_invoice_id IS NOT NULL AND cle.created_at > $31 AND cle.created_at <= $32
)), candidates (id, invoice_date, due_date, total_amount_minor, open_now_minor) AS ((
  SELECT inv.id, inv.invoice_date, inv.due_date, inv.total_amount_minor, (inv.total_amount_minor - inv.applied_amount_minor)
  FROM invoices inv
  WHERE inv.organization_id = $19 AND inv.business_unit_id = $20 AND inv.invoice_date <= $33 AND ((inv.status = 'Posted' AND inv.bill_type IN ('Invoice', 'DebitMemo')) AND inv.total_amount_minor > inv.applied_amount_minor)
) UNION (
  SELECT inv2.id, inv2.invoice_date, inv2.due_date, inv2.total_amount_minor, (inv2.total_amount_minor - inv2.applied_amount_minor)
  FROM invoices inv2
  JOIN events e ON e.invoice_id = inv2.id
  WHERE inv2.organization_id = $19 AND inv2.business_unit_id = $20 AND (inv2.status = 'Posted' AND inv2.bill_type IN ('Invoice', 'DebitMemo')) AND inv2.invoice_date <= $33
)), moves (invoice_id, due_date, changed_at, delta) AS ((
  SELECT c.id, c.due_date, c.invoice_date, c.total_amount_minor
  FROM candidates c
  WHERE c.invoice_date > $34
) UNION ALL (
  SELECT c2.id, c2.due_date, e2.settled_at, (-e2.amount)
  FROM events e2
  JOIN candidates c2 ON c2.id = e2.invoice_id
  WHERE e2.settled_at > $35
) UNION ALL (
  SELECT c3.id, c3.due_date, greatest(e3.settled_at, e3.released_at), e3.amount
  FROM events e3
  JOIN candidates c3 ON c3.id = e3.invoice_id
  WHERE e3.released_at IS NOT NULL AND greatest(e3.settled_at, e3.released_at) > $36
)), changes (invoice_id, due_date, changed_at, delta) AS ((
  SELECT c4.id, c4.due_date, $37::bigint, c4.open_now_minor
  FROM candidates c4
) UNION ALL (
  SELECT m.invoice_id, m.due_date, m.changed_at, m.delta
  FROM moves m
) UNION ALL (
  SELECT m2.invoice_id, m2.due_date, $37::bigint, (-m2.delta)
  FROM moves m2
)), segments (invoice_id, due_date, starts_at, ends_at, open_minor) AS (
  SELECT c5.invoice_id, c5.due_date, c5.changed_at, lead(c5.changed_at, $38) OVER w, sum(sum(c5.delta)) OVER w
  FROM changes c5
  GROUP BY c5.invoice_id, c5.due_date, c5.changed_at
  WINDOW w AS (PARTITION BY c5.invoice_id ORDER BY c5.changed_at ASC)
), open_items (n, invoice_id, due_date, open_minor) AS (
  SELECT o.n, s2.invoice_id, s2.due_date, s2.open_minor
  FROM segments s2
  CROSS JOIN LATERAL (
    SELECT generate_series(CASE WHEN s2.ends_at IS NULL THEN $1::bigint ELSE greatest($1::bigint, (CAST(floor((CAST(($17::bigint - s2.ends_at) AS numeric) / $40::numeric)) AS bigint) + $16::bigint)) END, least($16::bigint, CAST(floor((CAST(($17::bigint - s2.starts_at) AS numeric) / $40::numeric)) AS bigint))) AS n
  ) o
  WHERE s2.open_minor > $39
)
SELECT p.period_end, coalesce(sum(CASE WHEN (oi.due_date IS NULL OR oi.due_date >= p.period_end) THEN oi.open_minor ELSE $1::bigint END), $2), coalesce(sum(CASE WHEN (oi.due_date < p.period_end AND ((p.period_end - oi.due_date) / $3::bigint) BETWEEN $4 AND $5) THEN oi.open_minor ELSE $1::bigint END), $6), coalesce(sum(CASE WHEN (oi.due_date < p.period_end AND ((p.period_end - oi.due_date) / $3::bigint) BETWEEN $7 AND $8) THEN oi.open_minor ELSE $1::bigint END), $9), coalesce(sum(CASE WHEN (oi.due_date < p.period_end AND ((p.period_end - oi.due_date) / $3::bigint) BETWEEN $10 AND $11) THEN oi.open_minor ELSE $1::bigint END), $12), coalesce(sum(CASE WHEN (oi.due_date < p.period_end AND ((p.period_end - oi.due_date) / $3::bigint) > $13) THEN oi.open_minor ELSE $1::bigint END), $14), coalesce(sum(oi.open_minor), $15), count(oi.invoice_id)
FROM points p
LEFT JOIN open_items oi ON oi.n = p.n
GROUP BY 1
ORDER BY p.period_end ASC
-- $1 = 0
-- $2 = 0
-- $3 = 86400
-- $4 = 1
-- $5 = 30
-- $6 = 0
-- $7 = 31
-- $8 = 60
-- $9 = 0
-- $10 = 61
-- $11 = 90
-- $12 = 0
-- $13 = 90
-- $14 = 0
-- $15 = 0
-- $16 = 1
-- $17 = 1795449600
-- $18 = 2678400
-- $19 = 'org_1'
-- $20 = 'bu_1'
-- $21 = 1792771200
-- $22 = 9223372036854775807
-- $23 = 1792771200
-- $24 = 9223372036854775807
-- $25 = 1792771200
-- $26 = 1792771200
-- $27 = 9223372036854775807
-- $28 = 1792771200
-- $29 = 9223372036854775807
-- $30 = 1792771200
-- $31 = 1792771200
-- $32 = 9223372036854775807
-- $33 = 1795449600
-- $34 = 1792771200
-- $35 = 1792771200
-- $36 = 1792771200
-- $37 = 1792771200
-- $38 = 1
-- $39 = 0
-- $40 = 2678400
