--keel:notx
-- Before the operator carries out a staff log pull it asks the TMS whether that staff address
-- holds a live support grant in any organization on the instance. That lookup reads by staff
-- address alone, across organizations, so it needs an index led by the address.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_support_access_grants_live_staff" ON "support_access_grants"("staff_email", "expires_at")
WHERE "revoked_at" IS NULL;
