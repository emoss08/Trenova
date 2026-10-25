SELECT a.primary_worker_id, ((a.created_at / 86400) * 86400), count(*)
FROM assignments a
WHERE a.business_unit_id = $1 AND a.organization_id = $2 AND a.archived_at IS NULL AND (((a.created_at / 86400) * 86400) >= $3) AND (((a.created_at / 86400) * 86400) < $4) AND a.primary_worker_id IN (
  SELECT w.id
  FROM workers w
  WHERE w.business_unit_id = $1 AND w.organization_id = $2 AND w.status = 'Active' AND w.fleet_code_id = $5 AND w.id IN (
    SELECT team_workers.id
    FROM workers team_workers
    WHERE team_workers.business_unit_id = $1 AND team_workers.organization_id = $2 AND (team_workers.manager_id = ANY($6) OR team_workers.fleet_code_id IN (
      SELECT team_fleets.id
      FROM fleet_codes team_fleets
      WHERE team_fleets.business_unit_id = $1 AND team_fleets.organization_id = $2 AND team_fleets.manager_id = ANY($7)
    ))
  )
)
GROUP BY a.primary_worker_id, 2
-- $1 = 'bu_1'
-- $2 = 'org_1'
-- $3 = 1700000000
-- $4 = 1700604800
-- $5 = 'fc_1'
-- $6 = '{usr_1}'
-- $7 = '{usr_1}'
