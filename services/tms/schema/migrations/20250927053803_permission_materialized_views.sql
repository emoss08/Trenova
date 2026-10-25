-- User effective policies materialized view for fast permission resolution
CREATE MATERIALIZED VIEW user_effective_policies AS
WITH user_roles AS (
    -- Get all roles for each user in each organization
    SELECT
        uom.user_id,
        uom.organization_id,
        uom.business_unit_id,
        unnest(uom.role_ids) AS role_id
    FROM
        user_organization_memberships uom
    WHERE
        uom.expires_at IS NULL
        OR uom.expires_at > EXTRACT(epoch FROM CURRENT_TIMESTAMP)::bigint
),
role_policies AS (
    -- Get all policies from roles (including inherited roles)
    SELECT DISTINCT
        ur.user_id,
        ur.organization_id,
        ur.business_unit_id,
        unnest(r.policy_ids) AS policy_id,
        'role' AS assignment_type,
        ur.role_id AS source_id
    FROM
        user_roles ur
        JOIN roles r ON r.id = ur.role_id
    WHERE
        r.level IN ('system', 'bu', 'org', 'custom')
),
direct_policies AS (
    -- Get directly assigned policies
    SELECT
        uom.user_id,
        uom.organization_id,
        uom.business_unit_id,
        unnest(uom.direct_policies) AS policy_id,
        'direct' AS assignment_type,
        NULL AS source_id
    FROM
        user_organization_memberships uom
    WHERE
        array_length(uom.direct_policies, 1) > 0
        AND (uom.expires_at IS NULL
            OR uom.expires_at > EXTRACT(epoch FROM CURRENT_TIMESTAMP)::bigint)
),
all_user_policies AS (
    -- Combine role and direct policies
    SELECT
        *
    FROM
        role_policies
    UNION ALL
    SELECT
        *
    FROM
        direct_policies
),
policy_details AS (
    -- Join with policy details and apply scoping rules
    SELECT
        aup.user_id,
        aup.organization_id,
        aup.business_unit_id,
        p.id AS policy_id,
        p.name AS policy_name,
        p.effect,
        p.priority,
        p.resources,
        p.scope,
        aup.assignment_type,
        aup.source_id
    FROM
        all_user_policies aup
        JOIN policies p ON p.id = aup.policy_id
    WHERE
        -- Apply business unit scoping
        p.scope ->> 'businessUnitId' = aup.business_unit_id::text
        AND (
            -- Policy applies to all organizations in BU
(p.scope ->> 'organizationIds' = '[]'
                OR p.scope ->> 'organizationIds' IS NULL)
            OR
            -- Policy applies to specific organizations and user's org is included
(p.scope -> 'organizationIds' ? aup.organization_id::text)))
SELECT
    user_id,
    organization_id,
    business_unit_id,
    policy_id,
    policy_name,
    effect,
    priority,
    resources,
    scope,
    assignment_type,
    source_id,
    -- Add computed fields for faster lookups
    array_length(ARRAY (
            SELECT
                jsonb_array_elements_text(resources -> 'resourceType')), 1) AS resource_count,
    -- Cache the permission hash for quick comparison
    md5(user_id::text || organization_id::text || policy_id::text || coalesce(resources::text, '') || coalesce(scope::text, '')) AS permission_hash
FROM
    policy_details;

-- Create indexes on the materialized view
CREATE UNIQUE INDEX idx_user_effective_policies_unique ON user_effective_policies(user_id, organization_id, policy_id);

CREATE INDEX idx_user_effective_policies_user_org ON user_effective_policies(user_id, organization_id);

CREATE INDEX idx_user_effective_policies_effect_priority ON user_effective_policies(effect, priority DESC);

CREATE INDEX idx_user_effective_policies_assignment_type ON user_effective_policies(assignment_type);

CREATE INDEX idx_user_effective_policies_hash ON user_effective_policies(permission_hash);

-- Function to refresh the materialized view
CREATE OR REPLACE FUNCTION refresh_user_effective_policies()
    RETURNS void
    AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY user_effective_policies;
END;
$$
LANGUAGE plpgsql;

-- Function to refresh for specific user/org
CREATE OR REPLACE FUNCTION refresh_user_permissions(p_user_id varchar(100), p_organization_id varchar(100) DEFAULT NULL)
    RETURNS void
    AS $$
BEGIN
    -- For now, refresh the entire view
    -- In the future, we could implement partial refresh logic
    PERFORM
        refresh_user_effective_policies();
END;
$$
LANGUAGE plpgsql;

-- Trigger function to auto-refresh when policies change
CREATE OR REPLACE FUNCTION trigger_refresh_user_policies()
    RETURNS TRIGGER
    AS $$
BEGIN
    -- Refresh the materialized view immediately
    -- Note: REFRESH MATERIALIZED VIEW CONCURRENTLY requires the view to have a unique index
    REFRESH MATERIALIZED VIEW CONCURRENTLY user_effective_policies;

    -- Also send notification for any listeners
    PERFORM
        pg_notify('permission_refresh', json_build_object('table', TG_TABLE_NAME, 'operation', TG_OP, 'timestamp', EXTRACT(epoch FROM CURRENT_TIMESTAMP))::text);

    RETURN COALESCE(NEW, OLD);
END;
$$
LANGUAGE plpgsql;

-- Create triggers for auto-refresh
CREATE TRIGGER trigger_policies_refresh
    AFTER INSERT OR UPDATE OR DELETE ON policies
    FOR EACH ROW
    EXECUTE FUNCTION trigger_refresh_user_policies();

CREATE TRIGGER trigger_roles_refresh
    AFTER INSERT OR UPDATE OR DELETE ON roles
    FOR EACH ROW
    EXECUTE FUNCTION trigger_refresh_user_policies();

CREATE TRIGGER trigger_user_memberships_refresh
    AFTER INSERT OR UPDATE OR DELETE ON user_organization_memberships
    FOR EACH ROW
    EXECUTE FUNCTION trigger_refresh_user_policies();

CREATE TRIGGER trigger_user_roles_refresh
    AFTER INSERT OR UPDATE OR DELETE ON user_organization_roles
    FOR EACH ROW
    EXECUTE FUNCTION trigger_refresh_user_policies();

-- Initial refresh
SELECT
    refresh_user_effective_policies();

