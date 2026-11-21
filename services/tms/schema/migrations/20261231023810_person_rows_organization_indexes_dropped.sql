--keel:notx
-- The organization-only indexes of role assignments and second factors are covered by the
-- (organization_id, business_unit_id) indexes 20261231023780 made for the swapped keys.
DROP INDEX CONCURRENTLY IF EXISTS "idx_user_role_assignments_fk_organization_id";

--keel:split

DROP INDEX CONCURRENTLY IF EXISTS "idx_mfa_authenticators_fk_organization_id";

--keel:split

DROP INDEX CONCURRENTLY IF EXISTS "idx_mfa_recovery_codes_fk_organization_id";
