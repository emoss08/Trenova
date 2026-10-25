SELECT count(*) FILTER (WHERE br.status = 'Imported'), coalesce(sum(br.amount_minor) FILTER (WHERE br.status = 'Imported'), $4), count(*) FILTER (WHERE br.status = 'Matched'), coalesce(sum(br.amount_minor) FILTER (WHERE br.status = 'Matched'), $5), count(*) FILTER (WHERE br.status = 'Exception'), coalesce(sum(br.amount_minor) FILTER (WHERE br.status = 'Exception'), $6), (
  SELECT count(*)
  FROM bank_receipt_work_items wi
  WHERE wi.organization_id = $1 AND wi.business_unit_id = $2 AND wi.status IN ('Open', 'Assigned', 'InReview')
), (
  SELECT count(*)
  FROM bank_receipt_work_items wi2
  WHERE wi2.organization_id = $1 AND wi2.business_unit_id = $2 AND wi2.status IN ('Assigned')
), (
  SELECT count(*)
  FROM bank_receipt_work_items wi3
  WHERE wi3.organization_id = $1 AND wi3.business_unit_id = $2 AND wi3.status IN ('InReview')
), count(*) FILTER (WHERE (br.status = 'Exception' AND ((($7 - br.receipt_date) / $8::bigint) <= $9))), coalesce(sum(br.amount_minor) FILTER (WHERE (br.status = 'Exception' AND ((($7 - br.receipt_date) / $8::bigint) <= $9))), $10), count(*) FILTER (WHERE (br.status = 'Exception' AND ((($7 - br.receipt_date) / $8::bigint) BETWEEN $11 AND $12))), coalesce(sum(br.amount_minor) FILTER (WHERE (br.status = 'Exception' AND ((($7 - br.receipt_date) / $8::bigint) BETWEEN $11 AND $12))), $13), count(*) FILTER (WHERE (br.status = 'Exception' AND ((($7 - br.receipt_date) / $8::bigint) BETWEEN $14 AND $15))), coalesce(sum(br.amount_minor) FILTER (WHERE (br.status = 'Exception' AND ((($7 - br.receipt_date) / $8::bigint) BETWEEN $14 AND $15))), $16), count(*) FILTER (WHERE (br.status = 'Exception' AND ((($7 - br.receipt_date) / $8::bigint) > $17))), coalesce(sum(br.amount_minor) FILTER (WHERE (br.status = 'Exception' AND ((($7 - br.receipt_date) / $8::bigint) > $17))), $18)
FROM bank_receipts br
WHERE br.organization_id = $1 AND br.business_unit_id = $2 AND br.receipt_date <= $3
-- $1 = 'org_1'
-- $2 = 'bu_1'
-- $3 = 1000
-- $4 = 0
-- $5 = 0
-- $6 = 0
-- $7 = 1000
-- $8 = 86400
-- $9 = 0
-- $10 = 0
-- $11 = 1
-- $12 = 3
-- $13 = 0
-- $14 = 4
-- $15 = 7
-- $16 = 0
-- $17 = 7
-- $18 = 0
