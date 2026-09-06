--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TYPE "pto_termination_action_enum" AS ENUM(
    'Forfeit',
    'PayOut'
);

CREATE TYPE "worker_leave_type_enum" AS ENUM(
    'FMLA',
    'Medical',
    'Military',
    'Parental',
    'Personal',
    'Other'
);

CREATE TYPE "org_holiday_kind_enum" AS ENUM(
    'Holiday',
    'Blackout'
);

--bun:split
ALTER TABLE "pto_policy_rules"
    ADD COLUMN IF NOT EXISTS "tiers" jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS "on_termination" pto_termination_action_enum NOT NULL DEFAULT 'Forfeit';
--bun:split
COMMENT ON COLUMN pto_policy_rules.tiers IS 'Tenure tiers [{minMonths, accrualAmountDays, maxBalanceDays}] applied by months of service at each accrual; the base amount covers tenure below the first tier.';
--bun:split
ALTER TABLE "worker_pto_ledger"
    DROP CONSTRAINT IF EXISTS "chk_worker_pto_ledger_amount_sign";
--bun:split
ALTER TABLE "worker_pto_ledger"
    ADD CONSTRAINT "chk_worker_pto_ledger_amount_sign" CHECK (("entry_type" IN ('OpeningBalance', 'Accrual', 'Reversal', 'Carryover') AND "amount_days" > 0) OR ("entry_type" IN ('Usage', 'Expiry', 'Payout', 'Forfeiture') AND "amount_days" < 0) OR "entry_type" = 'Adjustment');
--bun:split
ALTER TABLE "workers"
    ADD COLUMN IF NOT EXISTS "leave_type" worker_leave_type_enum;
--bun:split
CREATE TABLE IF NOT EXISTS "org_holidays"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "name" varchar(100) NOT NULL,
    "holiday_date" bigint NOT NULL,
    "kind" org_holiday_kind_enum NOT NULL DEFAULT 'Holiday',
    "recurs_annually" boolean NOT NULL DEFAULT FALSE,
    "description" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_org_holidays" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_org_holidays_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_org_holidays_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_org_holidays_date" CHECK ("holiday_date" > 0)
);
--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_org_holidays_date_kind" ON "org_holidays"("organization_id", "business_unit_id", "holiday_date", "kind");
--bun:split
CREATE INDEX IF NOT EXISTS "idx_org_holidays_date" ON "org_holidays"("organization_id", "business_unit_id", "holiday_date");
--bun:split
COMMENT ON TABLE org_holidays IS 'Organisation holiday calendar. Holidays are skipped when a PTO policy does not count weekends; Blackout dates cannot be requested off at all. holiday_date is midnight UTC of the calendar date; recurs_annually repeats month/day every year.';
