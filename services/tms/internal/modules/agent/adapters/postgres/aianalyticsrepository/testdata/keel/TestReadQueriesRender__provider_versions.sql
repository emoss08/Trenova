SELECT asv.subject_id, asv.version, coalesce((asv.snapshot #>> $4), $5), coalesce((asv.snapshot #>> $6), $7), coalesce(CAST((asv.snapshot #>> $8) AS boolean), $9), asv.created_at
FROM ai_setting_versions asv
WHERE asv.business_unit_id = $1 AND asv.organization_id = $2 AND asv.kind = 'AIProvider' AND asv.subject_id IN (
  SELECT changed.subject_id
  FROM ai_setting_versions changed
  WHERE changed.business_unit_id = $1 AND changed.organization_id = $2 AND changed.kind = 'AIProvider' AND changed.created_at >= $3
)
ORDER BY asv.subject_id ASC, asv.version ASC
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 1790000000
-- $4 = '{name}'
-- $5 = ''
-- $6 = '{model}'
-- $7 = ''
-- $8 = '{enabled}'
-- $9 = false
