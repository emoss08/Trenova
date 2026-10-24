ALTER TABLE "carriers"
    ADD COLUMN IF NOT EXISTS "intel_risk_level" varchar(20),
    ADD COLUMN IF NOT EXISTS "intel_review_required" boolean NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS "intel_blocking_count" integer NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS "idx_carriers_intel_review_required" ON "carriers"("organization_id", "business_unit_id")
WHERE
    "intel_review_required";

COMMENT ON COLUMN "carriers"."intel_risk_level" IS 'Risk level from the current carrier intelligence snapshot. Maintained by the intelligence service only; user edits never write it.';

COMMENT ON COLUMN "carriers"."intel_review_required" IS 'Set when a new blocking intelligence finding appears and cleared when someone marks the snapshot reviewed.';

COMMENT ON COLUMN "carriers"."intel_blocking_count" IS 'Number of blocking, non-overridden findings on the current intelligence snapshot.';

ALTER TABLE "customers"
    ADD COLUMN IF NOT EXISTS "dot_number" varchar(12),
    ADD COLUMN IF NOT EXISTS "mc_number" varchar(12),
    ADD COLUMN IF NOT EXISTS "broker_vetting_enabled" boolean NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS "idx_customers_broker_vetting" ON "customers"("organization_id", "business_unit_id")
WHERE
    "broker_vetting_enabled";

COMMENT ON COLUMN "customers"."broker_vetting_enabled" IS 'For asset carriers: vet this customer as a freight broker (broker authority and bond) through carrier intelligence.';
