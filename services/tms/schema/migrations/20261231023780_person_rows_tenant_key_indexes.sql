--keel:notx
-- The organization keys of role assignments and second factors now carry the business unit, so
-- each gets an index led by both columns: a delete of an organization and a tenant removal find the
-- rows without scanning every tenant's. The organization-only indexes they replace are dropped once
-- the keys are swapped.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_user_role_assignments_fk_organization_id_business_unit_id"
    ON "user_role_assignments"("organization_id", "business_unit_id");

--keel:split

CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_mfa_authenticators_fk_organization_id_business_unit_id"
    ON "mfa_authenticators"("organization_id", "business_unit_id");

--keel:split

CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_mfa_recovery_codes_fk_organization_id_business_unit_id"
    ON "mfa_recovery_codes"("organization_id", "business_unit_id");
