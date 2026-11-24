--keel:notx
-- A clerk keeps one driver settlement out of the pay run on purpose: the hold is stored beside
-- the status (why, a note, who placed it and when; who released it and when), so a held
-- settlement is still the Draft, Pending approval or Approved settlement it was.
ALTER TABLE "driver_settlements"
    ADD COLUMN IF NOT EXISTS "hold_reason" varchar(50),
    ADD COLUMN IF NOT EXISTS "hold_note" text,
    ADD COLUMN IF NOT EXISTS "held_by_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "held_at" bigint,
    ADD COLUMN IF NOT EXISTS "hold_released_by_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "hold_released_at" bigint;

--keel:split
ALTER TABLE "driver_settlements"
    ADD CONSTRAINT "chk_driver_settlements_hold_reason" CHECK ("hold_reason" IS NULL OR "hold_reason" IN ('RateDispute', 'MissingPaperwork', 'NegativeNet', 'DeductionReview', 'Other')) NOT VALID,
    ADD CONSTRAINT "chk_driver_settlements_hold_whole" CHECK (("hold_reason" IS NULL) = ("held_at" IS NULL)) NOT VALID,
    ADD CONSTRAINT "chk_driver_settlements_hold_note" CHECK ("hold_note" IS NULL OR length("hold_note") <= 500) NOT VALID,
    -- A posted or paid settlement is paid through its journal; it can never be held.
    ADD CONSTRAINT "chk_driver_settlements_hold_unreleased" CHECK ("hold_reason" IS NULL OR "status" IN ('Draft', 'PendingApproval', 'Approved')) NOT VALID,
    ADD CONSTRAINT "fk_driver_settlements_held_by" FOREIGN KEY ("held_by_id") REFERENCES "users"("id") ON DELETE SET NULL NOT VALID,
    ADD CONSTRAINT "fk_driver_settlements_hold_released_by" FOREIGN KEY ("hold_released_by_id") REFERENCES "users"("id") ON DELETE SET NULL NOT VALID;

--keel:split
ALTER TABLE "driver_settlements" VALIDATE CONSTRAINT "chk_driver_settlements_hold_reason";

--keel:split
ALTER TABLE "driver_settlements" VALIDATE CONSTRAINT "chk_driver_settlements_hold_whole";

--keel:split
ALTER TABLE "driver_settlements" VALIDATE CONSTRAINT "chk_driver_settlements_hold_note";

--keel:split
ALTER TABLE "driver_settlements" VALIDATE CONSTRAINT "chk_driver_settlements_hold_unreleased";

--keel:split
ALTER TABLE "driver_settlements" VALIDATE CONSTRAINT "fk_driver_settlements_held_by";

--keel:split
ALTER TABLE "driver_settlements" VALIDATE CONSTRAINT "fk_driver_settlements_hold_released_by";

--keel:split
-- A batch keeps how many of its settlements are held and what they net, beside its other totals.
ALTER TABLE "driver_settlement_batches"
    ADD COLUMN IF NOT EXISTS "held_count" integer NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "held_net_minor" bigint NOT NULL DEFAULT 0;

--keel:split
-- The held settlements of a batch: the release split, the blockers read and the batch view.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_driver_settlements_held_batch" ON "driver_settlements"("organization_id", "business_unit_id", "batch_id") WHERE "hold_reason" IS NOT NULL;

--keel:split
-- The held settlements of a pay period: the settlement workspace's count and filter.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_driver_settlements_held_period" ON "driver_settlements"("organization_id", "business_unit_id", "period_start", "period_end") WHERE "hold_reason" IS NOT NULL;

--keel:split
-- Holding a settlement is its own operation, driver_settlement hold. Every role and API key that
-- approves driver settlements today may hold and release them, so nobody loses a way to keep a
-- settlement out of a pay run; an administrator then grants or removes it on its own.
UPDATE
    "resource_permissions"
SET
    "operations" = array_append("operations", 'hold'),
    "updated_at" = extract(epoch FROM current_timestamp)::bigint
WHERE
    "resource" = 'driver_settlement'
    AND 'approve' = ANY ("operations")
    AND NOT ('hold' = ANY ("operations"));

--keel:split
UPDATE
    "api_key_permissions"
SET
    "operations" = array_append("operations", 'hold'),
    "updated_at" = extract(epoch FROM current_timestamp)::bigint
WHERE
    "resource" = 'driver_settlement'
    AND 'approve' = ANY ("operations")
    AND NOT ('hold' = ANY ("operations"));
