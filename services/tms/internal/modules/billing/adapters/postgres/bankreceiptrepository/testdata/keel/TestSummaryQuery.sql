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
), count(*) FILTER (WHERE (br.status = 'Exception' AND (($7::bigint - br.receipt_date) / $8::bigint) <= $9)), coalesce(sum(br.amount_minor) FILTER (WHERE (br.status = 'Exception' AND (($7::bigint - br.receipt_date) / $8::bigint) <= $10)), $11), count(*) FILTER (WHERE (br.status = 'Exception' AND (($7::bigint - br.receipt_date) / $8::bigint) BETWEEN $12 AND $13)), coalesce(sum(br.amount_minor) FILTER (WHERE (br.status = 'Exception' AND (($7::bigint - br.receipt_date) / $8::bigint) BETWEEN $14 AND $15)), $16), count(*) FILTER (WHERE (br.status = 'Exception' AND (($7::bigint - br.receipt_date) / $8::bigint) BETWEEN $17 AND $18)), coalesce(sum(br.amount_minor) FILTER (WHERE (br.status = 'Exception' AND (($7::bigint - br.receipt_date) / $8::bigint) BETWEEN $19 AND $20)), $21), count(*) FILTER (WHERE (br.status = 'Exception' AND (($7::bigint - br.receipt_date) / $8::bigint) > $22)), coalesce(sum(br.amount_minor) FILTER (WHERE (br.status = 'Exception' AND (($7::bigint - br.receipt_date) / $8::bigint) > $23)), $24)
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
-- $11 = 0
-- $12 = 1
-- $13 = 3
-- $14 = 1
-- $15 = 3
-- $16 = 0
-- $17 = 4
-- $18 = 7
-- $19 = 4
-- $20 = 7
-- $21 = 0
-- $22 = 7
-- $23 = 7
-- $24 = 0
