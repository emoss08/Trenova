-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261021000000_compensation_benefits.tx.up.sql

ALTER TABLE "recurring_earnings" ADD COLUMN "kind" TEXT NOT NULL DEFAULT 'Standard';

--bun:split

ALTER TABLE "recurring_deductions" ADD COLUMN "kind" TEXT NOT NULL DEFAULT 'Standard';

--bun:split

ALTER TABLE "recurring_deductions" ADD COLUMN "court_order_number" TEXT;

--bun:split

ALTER TABLE "recurring_deductions" ADD COLUMN "case_number" TEXT;

--bun:split

ALTER TABLE "recurring_deductions" ADD COLUMN "issuing_agency" TEXT;

--bun:split

ALTER TABLE "recurring_deductions" ADD COLUMN "priority" INTEGER NOT NULL DEFAULT 100;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_recurring_deductions_priority" ON "recurring_deductions" ("organization_id", "business_unit_id", "worker_id", "priority");

--bun:split

CREATE TABLE IF NOT EXISTS "benefit_plans"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "plan_type" TEXT NOT NULL DEFAULT 'Medical',
    "carrier" TEXT,
    "policy_number" TEXT,
    "pay_code_id" TEXT NOT NULL,
    "plan_year" INTEGER NOT NULL,
    "employee_cost_minor" INTEGER NOT NULL DEFAULT 0,
    "employer_cost_minor" INTEGER NOT NULL DEFAULT 0,
    "currency_code" TEXT NOT NULL DEFAULT 'USD',
    "waiting_period_days" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_benefit_plans" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_benefit_plans_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_benefit_plans_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_benefit_plans_pay_code" FOREIGN KEY ("pay_code_id", "organization_id", "business_unit_id") REFERENCES "pay_codes"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_benefit_plans_costs" CHECK ("employee_cost_minor" >= 0 AND "employer_cost_minor" >= 0),
    CONSTRAINT "chk_benefit_plans_year" CHECK ("plan_year" >= 2000 AND "plan_year" <= 2200),
    CONSTRAINT "chk_benefit_plans_waiting" CHECK ("waiting_period_days" >= 0 AND "waiting_period_days" <= 365)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_benefit_plans_code_year" ON "benefit_plans" ("organization_id", "business_unit_id", lower("code"), "plan_year");

--bun:split

CREATE TABLE IF NOT EXISTS "worker_benefit_enrollments"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "benefit_plan_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "coverage_tier" TEXT NOT NULL DEFAULT 'Employee',
    "effective_from" INTEGER NOT NULL,
    "effective_to" INTEGER,
    "employee_cost_minor" INTEGER NOT NULL DEFAULT 0,
    "employer_cost_minor" INTEGER NOT NULL DEFAULT 0,
    "recurring_deduction_id" TEXT,
    "waived_reason" TEXT,
    "notes" TEXT,
    "enrolled_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
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

CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_benefit_enrollments_open" ON "worker_benefit_enrollments" ("organization_id", "business_unit_id", "worker_id", "benefit_plan_id")WHERE
    "effective_to" IS NULL AND "status" IN ('Pending', 'Active');

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_benefit_enrollments_worker" ON "worker_benefit_enrollments" ("organization_id", "business_unit_id", "worker_id", "status");
