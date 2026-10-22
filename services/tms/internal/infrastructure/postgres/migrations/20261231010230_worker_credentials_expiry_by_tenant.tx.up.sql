-- Credentials falling due are read for one organization, soonest first. The
-- expiry index serves the nightly sweep across every tenant; this one serves an
-- organization's walk without passing over every other tenant's credentials in
-- the window.
CREATE INDEX IF NOT EXISTS "idx_worker_credentials_expiry_tenant" ON "worker_credentials"("organization_id", "business_unit_id", "expires_at")
WHERE
    "status" = 'Active' AND "expires_at" IS NOT NULL;
