SELECT cle.customer_id, coalesce(sum(cle.amount_minor), $4)
FROM customer_ledger_entries cle
WHERE cle.organization_id = $1 AND cle.business_unit_id = $2 AND NOT (EXISTS (
  SELECT 1
  FROM invoices inv
  WHERE inv.organization_id = $1 AND inv.business_unit_id = $2 AND inv.id = coalesce(cle.related_invoice_id, CASE WHEN cle.source_object_type = $3 THEN cle.source_object_id END) AND (EXISTS (
    SELECT 1
    FROM factoring_companies fac
    LEFT JOIN factoring_agreements agr ON agr.organization_id = $1 AND agr.business_unit_id = $2 AND agr.id = inv.factoring_agreement_id
    WHERE fac.business_unit_id = $2 AND fac.organization_id = $1 AND fac.id = inv.factoring_company_id AND coalesce(agr.recourse = 'NonRecourse', fac.non_recourse)
  ) AND NOT (EXISTS (
    SELECT 1
    FROM factoring_submission_items fsi
    WHERE fsi.organization_id = $1 AND fsi.business_unit_id = $2 AND fsi.invoice_id = inv.id AND fsi.status IN ('ChargedBack')
  )))
))
GROUP BY cle.customer_id
-- $1 = 'org_1'
-- $2 = 'bu_1'
-- $3 = 'Invoice'
-- $4 = 0
