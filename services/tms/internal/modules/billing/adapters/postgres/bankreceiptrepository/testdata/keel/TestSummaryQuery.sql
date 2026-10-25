SELECT count(*) FILTER (WHERE br.status = $4), coalesce(sum(br.amount_minor) FILTER (WHERE br.status = $4), $5), count(*) FILTER (WHERE br.status = $6), coalesce(sum(br.amount_minor) FILTER (WHERE br.status = $6), $7), count(*) FILTER (WHERE br.status = $8), coalesce(sum(br.amount_minor) FILTER (WHERE br.status = $8), $9), (
  SELECT count(*)
  FROM bank_receipt_work_items wi
  WHERE wi.organization_id = $1 AND wi.business_unit_id = $2 AND wi.status = ANY($10)
), (
  SELECT count(*)
  FROM bank_receipt_work_items wi2
  WHERE wi2.organization_id = $1 AND wi2.business_unit_id = $2 AND wi2.status = ANY($11)
), (
  SELECT count(*)
  FROM bank_receipt_work_items wi3
  WHERE wi3.organization_id = $1 AND wi3.business_unit_id = $2 AND wi3.status = ANY($12)
), count(*) FILTER (WHERE (br.status = $8 AND ((($13 - br.receipt_date) / $14::bigint) <= $15))), coalesce(sum(br.amount_minor) FILTER (WHERE (br.status = $8 AND ((($13 - br.receipt_date) / $14::bigint) <= $15))), $16), count(*) FILTER (WHERE (br.status = $8 AND ((($13 - br.receipt_date) / $14::bigint) BETWEEN $17 AND $18))), coalesce(sum(br.amount_minor) FILTER (WHERE (br.status = $8 AND ((($13 - br.receipt_date) / $14::bigint) BETWEEN $17 AND $18))), $19), count(*) FILTER (WHERE (br.status = $8 AND ((($13 - br.receipt_date) / $14::bigint) BETWEEN $20 AND $21))), coalesce(sum(br.amount_minor) FILTER (WHERE (br.status = $8 AND ((($13 - br.receipt_date) / $14::bigint) BETWEEN $20 AND $21))), $22), count(*) FILTER (WHERE (br.status = $8 AND ((($13 - br.receipt_date) / $14::bigint) > $23))), coalesce(sum(br.amount_minor) FILTER (WHERE (br.status = $8 AND ((($13 - br.receipt_date) / $14::bigint) > $23))), $24)
FROM bank_receipts br
WHERE br.organization_id = $1 AND br.business_unit_id = $2 AND br.receipt_date <= $3
-- $1 = 'org_1'
-- $2 = 'bu_1'
-- $3 = 1000
-- $4 = 'Imported'
-- $5 = 0
-- $6 = 'Matched'
-- $7 = 0
-- $8 = 'Exception'
-- $9 = 0
-- $10 = '{Open,Assigned,InReview}'
-- $11 = '{Assigned}'
-- $12 = '{InReview}'
-- $13 = 1000
-- $14 = 86400
-- $15 = 0
-- $16 = 0
-- $17 = 1
-- $18 = 3
-- $19 = 0
-- $20 = 4
-- $21 = 7
-- $22 = 0
-- $23 = 7
-- $24 = 0
