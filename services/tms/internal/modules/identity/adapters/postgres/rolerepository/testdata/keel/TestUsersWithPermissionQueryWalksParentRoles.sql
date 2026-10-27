WITH RECURSIVE held (user_id, role_id, parent_role_ids) AS (
  SELECT ura.user_id, r.id, r.parent_role_ids
  FROM user_role_assignments ura
  JOIN roles r ON r.business_unit_id = $1 AND r.organization_id = $5 AND r.id = ura.role_id
  WHERE ura.organization_id = $5 AND (ura.expires_at IS NULL OR ura.expires_at > $6)
  UNION SELECT h2.user_id, parent_role.id, parent_role.parent_role_ids
  FROM roles parent_role
  JOIN held h2 ON (parent_role.id = ANY(h2.parent_role_ids))
  WHERE parent_role.business_unit_id = $1 AND parent_role.organization_id = $5
)
SELECT u.id, u.name, u.username, u.email_address, coalesce(u.profile_pic_url, $4), u.locale
FROM users u
WHERE u.business_unit_id = $1 AND u.status = 'Active' AND EXISTS (
  SELECT 1
  FROM held h
  JOIN resource_permissions rp ON rp.role_id = h.role_id
  WHERE h.user_id = u.id AND rp.resource = $2 AND rp.operations && $3
)
ORDER BY u.name ASC
-- $1 = 'bu_1'
-- $2 = 'agent_proposal'
-- $3 = '{activate,approve,approve_own,archive,assign,cancel,close,lock,manage,reject,reopen,restore,submit,unassign,unlock,update}'
-- $4 = ''
-- $5 = 'org_1'
-- $6 = 100
