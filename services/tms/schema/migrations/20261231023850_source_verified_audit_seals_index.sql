--keel:notx
-- The audit verification asks each tenant, once no seal is marked as not verified at its source,
-- whether any seal is marked as verified at its source as its archive states; a partial index
-- answers with one probe instead of a walk of every seal.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_audit_seals_source_verified"
    ON "audit_seals"("organization_id", "business_unit_id", "seq")
    WHERE "hash_version" IN (5, 6);
