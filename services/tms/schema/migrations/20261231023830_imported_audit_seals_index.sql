--keel:notx
-- The audit verification asks each tenant whether its chain still holds imported seals that were
-- not verified at their source; a partial index answers with one probe instead of a walk of
-- every seal.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_audit_seals_imported"
    ON "audit_seals"("organization_id", "business_unit_id", "seq")
    WHERE "hash_version" IN (3, 4);
