WITH series (n) AS (
  SELECT generate_series($16::bigint, $17::bigint) AS n
), points (n, period_end) AS (
  SELECT s.n, ($18::bigint - (s.n * $19::bigint))
  FROM series s
), events (invoice_id, settled_at, released_at, amount) AS ((
  SELECT cpa5.invoice_id, cpa5.created_at, cp6.reversed_at, (cpa5.applied_amount_minor + cpa5.short_pay_amount_minor)
  FROM customer_payment_applications cpa5
  JOIN customer_payments cp6 ON cp6.organization_id = $6 AND cp6.business_unit_id = $7 AND cp6.id = cpa5.customer_payment_id
  WHERE cpa5.organization_id = $6 AND cpa5.business_unit_id = $7 AND cpa5.created_at > $20 AND cpa5.created_at <= $21
) UNION ALL (
  SELECT cpa6.invoice_id, cpa6.created_at, cp7.reversed_at, (cpa6.applied_amount_minor + cpa6.short_pay_amount_minor)
  FROM customer_payments cp7
  JOIN customer_payment_applications cpa6 ON cpa6.organization_id = $6 AND cpa6.business_unit_id = $7 AND cpa6.customer_payment_id = cp7.id
  WHERE cp7.organization_id = $6 AND cp7.business_unit_id = $7 AND cp7.reversed_at > $22 AND cp7.reversed_at <= $23 AND cpa6.created_at <= $24
) UNION ALL (
  SELECT cma.invoice_id, cma.created_at, cma.unapplied_at, cma.applied_amount_minor
  FROM credit_memo_applications cma
  WHERE cma.organization_id = $6 AND cma.business_unit_id = $7 AND cma.created_at > $25 AND cma.created_at <= $26
) UNION ALL (
  SELECT cma2.invoice_id, cma2.created_at, cma2.unapplied_at, cma2.applied_amount_minor
  FROM credit_memo_applications cma2
  WHERE cma2.organization_id = $6 AND cma2.business_unit_id = $7 AND cma2.unapplied_at > $27 AND cma2.unapplied_at <= $28 AND cma2.created_at <= $29
) UNION ALL (
  SELECT cle.related_invoice_id, cle.created_at, NULL::bigint, (-cle.amount_minor)
  FROM customer_ledger_entries cle
  WHERE cle.organization_id = $6 AND cle.business_unit_id = $7 AND cle.source_object_type = 'FactoringSubmissionItem' AND cle.related_invoice_id IS NOT NULL AND cle.created_at > $30 AND cle.created_at <= $31
)), candidates (id, invoice_date, due_date, total_amount_minor, open_now_minor) AS ((
  SELECT inv5.id, inv5.invoice_date, inv5.due_date, inv5.total_amount_minor, (inv5.total_amount_minor - inv5.applied_amount_minor)
  FROM invoices inv5
  WHERE inv5.organization_id = $6 AND inv5.business_unit_id = $7 AND inv5.invoice_date <= $32 AND ((inv5.status = 'Posted' AND inv5.bill_type IN ('Invoice', 'DebitMemo')) AND inv5.total_amount_minor > inv5.applied_amount_minor)
) UNION (
  SELECT inv6.id, inv6.invoice_date, inv6.due_date, inv6.total_amount_minor, (inv6.total_amount_minor - inv6.applied_amount_minor)
  FROM invoices inv6
  JOIN events e ON e.invoice_id = inv6.id
  WHERE inv6.organization_id = $6 AND inv6.business_unit_id = $7 AND (inv6.status = 'Posted' AND inv6.bill_type IN ('Invoice', 'DebitMemo')) AND inv6.invoice_date <= $32
)), moves (invoice_id, due_date, changed_at, delta) AS ((
  SELECT c.id, c.due_date, c.invoice_date, c.total_amount_minor
  FROM candidates c
  WHERE c.invoice_date > $33
) UNION ALL (
  SELECT c2.id, c2.due_date, e2.settled_at, (-e2.amount)
  FROM events e2
  JOIN candidates c2 ON c2.id = e2.invoice_id
  WHERE e2.settled_at > $34
) UNION ALL (
  SELECT c3.id, c3.due_date, greatest(e3.settled_at, e3.released_at), e3.amount
  FROM events e3
  JOIN candidates c3 ON c3.id = e3.invoice_id
  WHERE e3.released_at IS NOT NULL AND greatest(e3.settled_at, e3.released_at) > $35
)), changes (invoice_id, due_date, changed_at, delta) AS ((
  SELECT c4.id, c4.due_date, $36::bigint, c4.open_now_minor
  FROM candidates c4
) UNION ALL (
  SELECT m.invoice_id, m.due_date, m.changed_at, m.delta
  FROM moves m
) UNION ALL (
  SELECT m2.invoice_id, m2.due_date, $36::bigint, (-m2.delta)
  FROM moves m2
)), segments (invoice_id, due_date, starts_at, ends_at, open_minor) AS (
  SELECT c5.invoice_id, c5.due_date, c5.changed_at, lead(c5.changed_at, $37) OVER w, sum(sum(c5.delta)) OVER w
  FROM changes c5
  GROUP BY c5.invoice_id, c5.due_date, c5.changed_at
  WINDOW w AS (PARTITION BY c5.invoice_id ORDER BY c5.changed_at ASC)
), open_items (n, invoice_id, due_date, open_minor) AS (
  SELECT o.n, s2.invoice_id, s2.due_date, s2.open_minor
  FROM segments s2
  CROSS JOIN LATERAL (
    SELECT generate_series(CASE WHEN s2.ends_at IS NULL THEN $16::bigint ELSE greatest($16::bigint, (CAST(floor((CAST(($18::bigint - s2.ends_at) AS numeric) / $39::numeric)) AS bigint) + $17::bigint)) END, least($17::bigint, CAST(floor((CAST(($18::bigint - s2.starts_at) AS numeric) / $39::numeric)) AS bigint))) AS n
  ) o
  WHERE s2.open_minor > $38
), bounds (period_start, period_end) AS (
  SELECT $36::bigint AS period_start, $18::bigint AS period_end
)
SELECT coalesce((
  SELECT sum(oi.open_minor)
  FROM open_items oi
  WHERE oi.n = $1
), $2), coalesce((
  SELECT sum(oi2.open_minor)
  FROM open_items oi2
  WHERE oi2.n = $3
), $4), coalesce((
  SELECT sum(oi3.open_minor)
  FROM open_items oi3
  WHERE oi3.n = $3 AND (oi3.due_date IS NULL OR oi3.due_date >= b.period_end)
), $5), coalesce((
  SELECT sum(inv.total_amount_minor)
  FROM invoices inv
  WHERE inv.organization_id = $6 AND inv.business_unit_id = $7 AND (inv.posted_at IS NOT NULL AND (inv.voided_at IS NULL OR inv.voided_at > b.period_end) AND inv.bill_type IN ('Invoice', 'DebitMemo') AND inv.invoice_date > b.period_start AND inv.invoice_date <= b.period_end)
), $8), coalesce((
  SELECT sum(cp.amount_minor)
  FROM customer_payments cp
  WHERE cp.organization_id = $6 AND cp.business_unit_id = $7 AND cp.status = 'Posted' AND (cp.payment_date > b.period_start AND cp.payment_date <= b.period_end)
), $9), coalesce((
  SELECT (sum((CAST(cpa.applied_amount_minor AS numeric) * (CAST((cp2.payment_date - inv2.invoice_date) AS numeric) / $11::numeric))) / nullif(sum(cpa.applied_amount_minor), $12))
  FROM customer_payment_applications cpa
  JOIN customer_payments cp2 ON cp2.organization_id = $6 AND cp2.business_unit_id = $7 AND cp2.id = cpa.customer_payment_id AND cp2.status = 'Posted'
  JOIN invoices inv2 ON inv2.organization_id = $6 AND inv2.business_unit_id = $7 AND inv2.id = cpa.invoice_id
  WHERE cpa.organization_id = $6 AND cpa.business_unit_id = $7 AND cpa.applied_amount_minor > $10 AND (cpa.created_at > b.period_start AND cpa.created_at <= b.period_end)
), $13), coalesce((
  SELECT sum(cpa2.short_pay_amount_minor)
  FROM customer_payment_applications cpa2
  JOIN customer_payments cp3 ON cp3.organization_id = $6 AND cp3.business_unit_id = $7 AND cp3.id = cpa2.customer_payment_id AND cp3.status = 'Posted'
  WHERE cpa2.organization_id = $6 AND cpa2.business_unit_id = $7 AND (cpa2.created_at > b.period_start AND cpa2.created_at <= b.period_end)
), $14), (
  SELECT count(*)
  FROM customer_payment_applications cpa3
  JOIN customer_payments cp4 ON cp4.organization_id = $6 AND cp4.business_unit_id = $7 AND cp4.id = cpa3.customer_payment_id AND cp4.status = 'Posted'
  WHERE cpa3.organization_id = $6 AND cpa3.business_unit_id = $7 AND (cpa3.created_at > b.period_start AND cpa3.created_at <= b.period_end) AND cpa3.short_pay_amount_minor > $15
), (
  SELECT count(*)
  FROM customer_payment_applications cpa4
  JOIN customer_payments cp5 ON cp5.organization_id = $6 AND cp5.business_unit_id = $7 AND cp5.id = cpa4.customer_payment_id AND cp5.status = 'Posted'
  WHERE cpa4.organization_id = $6 AND cpa4.business_unit_id = $7 AND (cpa4.created_at > b.period_start AND cpa4.created_at <= b.period_end)
), (
  SELECT count(*)
  FROM invoices inv3
  WHERE inv3.organization_id = $6 AND inv3.business_unit_id = $7 AND (inv3.status = 'Posted' AND inv3.bill_type IN ('Invoice', 'DebitMemo')) AND inv3.dispute_status = 'Disputed' AND (inv3.invoice_date > b.period_start AND inv3.invoice_date <= b.period_end)
), (
  SELECT count(*)
  FROM invoices inv4
  WHERE inv4.organization_id = $6 AND inv4.business_unit_id = $7 AND (inv4.posted_at IS NOT NULL AND (inv4.voided_at IS NULL OR inv4.voided_at > b.period_end) AND inv4.bill_type IN ('Invoice', 'DebitMemo') AND inv4.invoice_date > b.period_start AND inv4.invoice_date <= b.period_end)
)
FROM bounds b
-- $1 = 1
-- $2 = 0
-- $3 = 0
-- $4 = 0
-- $5 = 0
-- $6 = 'org_1'
-- $7 = 'bu_1'
-- $8 = 0
-- $9 = 0
-- $10 = 0
-- $11 = 86400
-- $12 = 0
-- $13 = 0
-- $14 = 0
-- $15 = 0
-- $16 = 0
-- $17 = 1
-- $18 = 1795449600
-- $19 = 7862400
-- $20 = 1787587200
-- $21 = 9223372036854775807
-- $22 = 1787587200
-- $23 = 9223372036854775807
-- $24 = 1787587200
-- $25 = 1787587200
-- $26 = 9223372036854775807
-- $27 = 1787587200
-- $28 = 9223372036854775807
-- $29 = 1787587200
-- $30 = 1787587200
-- $31 = 9223372036854775807
-- $32 = 1795449600
-- $33 = 1787587200
-- $34 = 1787587200
-- $35 = 1787587200
-- $36 = 1787587200
-- $37 = 1
-- $38 = 0
-- $39 = 7862400
