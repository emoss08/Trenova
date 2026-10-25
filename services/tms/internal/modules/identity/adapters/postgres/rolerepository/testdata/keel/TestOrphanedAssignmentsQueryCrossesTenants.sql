SELECT DISTINCT ON (ura.organization_id, ura.user_id) ura.user_id, ura.organization_id, o.business_unit_id
FROM user_role_assignments ura
JOIN organizations o ON o.id = ura.organization_id
JOIN users u ON u.id = ura.user_id
WHERE u.current_organization_id <> ura.organization_id AND NOT EXISTS (
  SELECT 1
  FROM user_organization_memberships uom
  WHERE uom.user_id = ura.user_id AND uom.organization_id = ura.organization_id AND (uom.expires_at IS NULL OR uom.expires_at > $1)
)
ORDER BY ura.organization_id ASC, ura.user_id ASC LIMIT 50
-- $1 = 100
