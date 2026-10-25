SELECT i.id, (i.total_amount_minor - i.applied_amount_minor)
FROM invoices i
WHERE i.organization_id = $1 AND i.business_unit_id = $2 AND ((i.status = $3 AND i.bill_type = ANY($4)) AND (i.total_amount_minor > i.applied_amount_minor)) AND (EXISTS (
  SELECT 1
  FROM factoring_companies fac
  LEFT JOIN factoring_agreements agr ON agr.organization_id = $1 AND agr.business_unit_id = $2 AND agr.id = i.factoring_agreement_id
  WHERE fac.business_unit_id = $2 AND fac.organization_id = $1 AND fac.id = i.factoring_company_id AND coalesce(agr.recourse = $5, fac.non_recourse)
) AND NOT (EXISTS (
  SELECT 1
  FROM factoring_submission_items fsi
  WHERE fsi.organization_id = $1 AND fsi.business_unit_id = $2 AND fsi.invoice_id = i.id AND fsi.status = ANY($6)
))) LIMIT 1
-- $1 = 'org_1'
-- $2 = 'bu_1'
-- $3 = 'Posted'
-- $4 = '{Invoice,DebitMemo}'
-- $5 = 'NonRecourse'
-- $6 = '{ChargedBack}'
