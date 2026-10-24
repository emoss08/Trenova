ALTER TABLE "shipments"
    ADD COLUMN IF NOT EXISTS "auto_rated" boolean NOT NULL DEFAULT false;

ALTER TABLE "shipments"
    ADD COLUMN IF NOT EXISTS "auto_rated_at" bigint;

UPDATE "shipments"
SET "auto_rated" = true,
    "auto_rated_at" = "updated_at"
WHERE "rate_agreement_id" IS NOT NULL
    AND "rate_override_amount" IS NULL;

CREATE INDEX IF NOT EXISTS "idx_shipments_auto_rated"
    ON "shipments" ("organization_id", "auto_rated")
    WHERE "auto_rated";
