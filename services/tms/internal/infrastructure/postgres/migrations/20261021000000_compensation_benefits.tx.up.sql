--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
--
-- Compensation history and benefits.
--
-- Deliberately built on the recurring earning and deduction machinery rather
-- than beside it: a benefit contribution and a garnishment are both money that
-- comes off a settlement, and the settlement builder already knows how to take
-- money off a settlement. A second mechanism would have to be taught the cap,
-- the pause, the reversal on a voided settlement and the escrow interaction all
-- over again, and would drift from the first one within a release.
CREATE TYPE "benefit_plan_type_enum" AS ENUM(
    'Medical',
    'Dental',
    'Vision',
    'Life',
    'Disability',
    'Retirement',
    'Other'
);

--bun:split
CREATE TYPE "benefit_enrollment_status_enum" AS ENUM(
    'Pending',
    'Active',
    'Waived',
    'Ended'
);

--bun:split
CREATE TYPE "benefit_coverage_tier_enum" AS ENUM(
    'Employee',
    'EmployeeSpouse',
    'EmployeeChildren',
    'Family'
);

--bun:split
-- Kind is VARCHAR to match the status and frequency columns already on these
-- tables, which are VARCHAR rather than Postgres enums.
ALTER TABLE "recurring_earnings"
    ADD COLUMN IF NOT EXISTS "kind" varchar(50) NOT NULL DEFAULT 'Standard';

--bun:split
COMMENT ON COLUMN recurring_earnings.kind IS 'Standard for ongoing pay, Bonus for a one-off. A bonus is an ordinary earning whose cap equals its amount, so it pays once and completes itself through the same path everything else uses.';

--bun:split
ALTER TABLE "recurring_deductions"
    ADD COLUMN IF NOT EXISTS "kind" varchar(50) NOT NULL DEFAULT 'Standard';

--bun:split
ALTER TABLE "recurring_deductions"
    ADD COLUMN IF NOT EXISTS "court_order_number" varchar(100);

--bun:split
ALTER TABLE "recurring_deductions"
    ADD COLUMN IF NOT EXISTS "case_number" varchar(100);

--bun:split
ALTER TABLE "recurring_deductions"
    ADD COLUMN IF NOT EXISTS "issuing_agency" varchar(150);

--bun:split
ALTER TABLE "recurring_deductions"
    ADD COLUMN IF NOT EXISTS "priority" smallint NOT NULL DEFAULT 100;

--bun:split
ALTER TABLE "recurring_deductions"
    ADD CONSTRAINT "chk_recurring_deductions_priority" CHECK ("priority" >= 0 AND "priority" <= 999);

--bun:split
COMMENT ON COLUMN recurring_deductions.priority IS 'The order deductions are taken in when pay will not cover them all. Lower goes first; garnishments default below everything else because a court order outranks a voluntary deduction.';

--bun:split
COMMENT ON COLUMN recurring_deductions.court_order_number IS 'The order a garnishment is taken under. Required for a garnishment: money taken from somebody''s pay without a reference to the order authorising it cannot be defended.';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_recurring_deductions_priority" ON "recurring_deductions"("organization_id", "business_unit_id", "worker_id", "priority");

--bun:split
CREATE TABLE IF NOT EXISTS "benefit_plans"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "status" status_enum NOT NULL DEFAULT 'Active',
    "code" varchar(20) NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" text,
    "plan_type" benefit_plan_type_enum NOT NULL DEFAULT 'Medical',
    "carrier" varchar(150),
    "policy_number" varchar(100),
    "pay_code_id" varchar(100) NOT NULL,
    "plan_year" smallint NOT NULL,
    "employee_cost_minor" bigint NOT NULL DEFAULT 0,
    "employer_cost_minor" bigint NOT NULL DEFAULT 0,
    "currency_code" varchar(3) NOT NULL DEFAULT 'USD',
    "waiting_period_days" integer NOT NULL DEFAULT 0,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_benefit_plans" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_benefit_plans_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_benefit_plans_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_benefit_plans_pay_code" FOREIGN KEY ("pay_code_id", "organization_id", "business_unit_id") REFERENCES "pay_codes"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_benefit_plans_costs" CHECK ("employee_cost_minor" >= 0 AND "employer_cost_minor" >= 0),
    CONSTRAINT "chk_benefit_plans_year" CHECK ("plan_year" >= 2000 AND "plan_year" <= 2200),
    CONSTRAINT "chk_benefit_plans_waiting" CHECK ("waiting_period_days" >= 0 AND "waiting_period_days" <= 365)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_benefit_plans_code_year" ON "benefit_plans"("organization_id", "business_unit_id", lower("code"), "plan_year");

--bun:split
COMMENT ON TABLE benefit_plans IS 'What a carrier offers, per plan year. The pay code is what a contribution shows up as on a settlement, which is why it is required rather than optional: a deduction nobody can categorise is a deduction nobody can explain.';

--bun:split
COMMENT ON COLUMN benefit_plans.employer_cost_minor IS 'What the employer pays. Never deducted — it is carried so a total-compensation statement can show what the job is actually worth.';

--bun:split
CREATE TABLE IF NOT EXISTS "worker_benefit_enrollments"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "benefit_plan_id" varchar(100) NOT NULL,
    "status" benefit_enrollment_status_enum NOT NULL DEFAULT 'Pending',
    "coverage_tier" benefit_coverage_tier_enum NOT NULL DEFAULT 'Employee',
    "effective_from" bigint NOT NULL,
    "effective_to" bigint,
    "employee_cost_minor" bigint NOT NULL DEFAULT 0,
    "employer_cost_minor" bigint NOT NULL DEFAULT 0,
    "recurring_deduction_id" varchar(100),
    "waived_reason" varchar(255),
    "notes" text,
    "enrolled_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_worker_benefit_enrollments" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_benefit_enrollments_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_benefit_enrollments_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_benefit_enrollments_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_benefit_enrollments_plan" FOREIGN KEY ("benefit_plan_id", "organization_id", "business_unit_id") REFERENCES "benefit_plans"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_worker_benefit_enrollments_deduction" FOREIGN KEY ("recurring_deduction_id", "organization_id", "business_unit_id") REFERENCES "recurring_deductions"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_worker_benefit_enrollments_costs" CHECK ("employee_cost_minor" >= 0 AND "employer_cost_minor" >= 0),
    CONSTRAINT "chk_worker_benefit_enrollments_window" CHECK ("effective_to" IS NULL OR "effective_to" >= "effective_from"),
    CONSTRAINT "chk_worker_benefit_enrollments_waived" CHECK ("status" <> 'Waived' OR "waived_reason" IS NOT NULL)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_benefit_enrollments_open" ON "worker_benefit_enrollments"("organization_id", "business_unit_id", "worker_id", "benefit_plan_id")
WHERE
    "effective_to" IS NULL AND "status" IN ('Pending', 'Active');

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_benefit_enrollments_worker" ON "worker_benefit_enrollments"("organization_id", "business_unit_id", "worker_id", "status");

--bun:split
COMMENT ON TABLE worker_benefit_enrollments IS 'Who is on which plan. The employee contribution is taken through a recurring deduction rather than a second mechanism, so it inherits the cap, the pause, and the reversal on a voided settlement that the deduction path already has.';

--bun:split
COMMENT ON COLUMN worker_benefit_enrollments.recurring_deduction_id IS 'The deduction this enrollment created. Nulled rather than cascaded if the deduction is removed, so an enrollment records that somebody was covered even after the money stops.';

--bun:split
COMMENT ON COLUMN worker_benefit_enrollments.employee_cost_minor IS 'Copied from the plan and the coverage tier when the enrollment is made. Copied rather than read through, because a plan repriced next year must not silently restate what somebody was charged this year.';
