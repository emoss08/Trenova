ALTER TABLE "customer_email_profiles"
    ADD COLUMN IF NOT EXISTS "from_email" VARCHAR(255);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_customer_email_profiles_from_email"
    ON "customer_email_profiles" ("from_email", "organization_id", "business_unit_id")
    WHERE "from_email" IS NOT NULL;
