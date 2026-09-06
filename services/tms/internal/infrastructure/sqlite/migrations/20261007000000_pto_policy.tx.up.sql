-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261007000000_pto_policy.tx.up.sql

CREATE TABLE IF NOT EXISTS "pto_policies"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "description" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Draft',
    "is_default" INTEGER NOT NULL DEFAULT 0,
    "year_basis" TEXT NOT NULL DEFAULT 'CalendarYear',
    "count_weekends" INTEGER NOT NULL DEFAULT 1,
    "waiting_period_days" INTEGER NOT NULL DEFAULT 0,
    "requires_approval" INTEGER NOT NULL DEFAULT 1,
    "enforce_balance" INTEGER NOT NULL DEFAULT 1,
    "allow_negative" INTEGER NOT NULL DEFAULT 0,
    "negative_floor_days" REAL NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_pto_policies" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_pto_policies_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_pto_policies_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_pto_policies_waiting_period" CHECK ("waiting_period_days" >= 0),
    CONSTRAINT "chk_pto_policies_negative_floor" CHECK ("negative_floor_days" <= 0),
    CONSTRAINT "chk_pto_policies_negative_requires_floor" CHECK (NOT "allow_negative" OR "negative_floor_days" < 0)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_pto_policies_code" ON "pto_policies" ("organization_id", "business_unit_id", LOWER("code"));

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_pto_policies_default" ON "pto_policies" ("organization_id", "business_unit_id")WHERE
    "is_default";

--bun:split

CREATE INDEX IF NOT EXISTS "idx_pto_policies_status" ON "pto_policies" ("organization_id", "business_unit_id", "status");

--bun:split

CREATE TABLE IF NOT EXISTS "pto_policy_rules"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "pto_policy_id" TEXT NOT NULL,
    "pto_type" TEXT NOT NULL,
    "accrual_method" TEXT NOT NULL DEFAULT 'None',
    "accrual_amount_days" REAL NOT NULL DEFAULT 0,
    "max_balance_days" REAL,
    "carryover_cap_days" REAL,
    "carryover_expiry_days" INTEGER NOT NULL DEFAULT 0,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_pto_policy_rules" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_pto_policy_rules_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_pto_policy_rules_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_pto_policy_rules_policy" FOREIGN KEY ("pto_policy_id", "organization_id", "business_unit_id") REFERENCES "pto_policies"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_pto_policy_rules_type" UNIQUE ("pto_policy_id", "organization_id", "business_unit_id", "pto_type"),
    CONSTRAINT "chk_pto_policy_rules_amount" CHECK ("accrual_amount_days" >= 0),
    CONSTRAINT "chk_pto_policy_rules_method_amount" CHECK ("accrual_method" = 'None' OR "accrual_amount_days" > 0),
    CONSTRAINT "chk_pto_policy_rules_max_balance" CHECK ("max_balance_days" IS NULL OR "max_balance_days" > 0),
    CONSTRAINT "chk_pto_policy_rules_carryover_cap" CHECK ("carryover_cap_days" IS NULL OR "carryover_cap_days" >= 0),
    CONSTRAINT "chk_pto_policy_rules_carryover_expiry" CHECK ("carryover_expiry_days" >= 0)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_pto_policy_rules_policy" ON "pto_policy_rules" ("pto_policy_id", "sort_order");

--bun:split

CREATE TABLE IF NOT EXISTS "worker_pto_policy_assignments"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "pto_policy_id" TEXT NOT NULL,
    "effective_from" INTEGER NOT NULL,
    "effective_to" INTEGER,
    "assigned_by_id" TEXT,
    "note" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_worker_pto_policy_assignments" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_pto_policy_assignments_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_pto_policy_assignments_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_pto_policy_assignments_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_pto_policy_assignments_policy" FOREIGN KEY ("pto_policy_id", "organization_id", "business_unit_id") REFERENCES "pto_policies"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_worker_pto_policy_assignments_range" CHECK ("effective_to" IS NULL OR "effective_to" > "effective_from")
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_pto_policy_assignments_open" ON "worker_pto_policy_assignments" ("organization_id", "business_unit_id", "worker_id")WHERE
    "effective_to" IS NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_pto_policy_assignments_worker" ON "worker_pto_policy_assignments" ("worker_id", "effective_from" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_pto_policy_assignments_policy_open" ON "worker_pto_policy_assignments" ("pto_policy_id")WHERE
    "effective_to" IS NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "worker_pto_balances"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "pto_type" TEXT NOT NULL,
    "balance_days" REAL NOT NULL DEFAULT 0,
    "accrued_ytd_days" REAL NOT NULL DEFAULT 0,
    "used_ytd_days" REAL NOT NULL DEFAULT 0,
    "carried_days" REAL NOT NULL DEFAULT 0,
    "entry_count" INTEGER NOT NULL DEFAULT 0,
    "last_accrual_period_key" TEXT,
    "last_rollover_key" TEXT,
    "year_started_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_worker_pto_balances" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_pto_balances_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_pto_balances_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_pto_balances_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_worker_pto_balances_worker_type" UNIQUE ("organization_id", "business_unit_id", "worker_id", "pto_type")
);

--bun:split

CREATE TABLE IF NOT EXISTS "worker_pto_ledger"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "pto_type" TEXT NOT NULL,
    "entry_type" TEXT NOT NULL,
    "amount_days" REAL NOT NULL,
    "balance_after_days" REAL NOT NULL,
    "sequence" INTEGER NOT NULL,
    "effective_at" INTEGER NOT NULL,
    "period_key" TEXT,
    "source_pto_id" TEXT,
    "assignment_id" TEXT,
    "pto_policy_id" TEXT,
    "actor_type" TEXT NOT NULL DEFAULT 'User',
    "created_by_id" TEXT,
    "note" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_worker_pto_ledger" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_pto_ledger_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_pto_ledger_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_pto_ledger_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_pto_ledger_source_pto" FOREIGN KEY ("source_pto_id", "worker_id", "organization_id", "business_unit_id") REFERENCES "worker_pto"("id", "worker_id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "uq_worker_pto_ledger_sequence" UNIQUE ("organization_id", "business_unit_id", "worker_id", "pto_type", "sequence"),
    CONSTRAINT "chk_worker_pto_ledger_amount_nonzero" CHECK ("amount_days" <> 0),
    CONSTRAINT "chk_worker_pto_ledger_amount_sign" CHECK (("entry_type" IN ('OpeningBalance', 'Accrual', 'Reversal', 'Carryover') AND "amount_days" > 0) OR ("entry_type" IN ('Usage', 'Expiry') AND "amount_days" < 0) OR "entry_type" = 'Adjustment'),
    CONSTRAINT "chk_worker_pto_ledger_adjustment_note" CHECK ("entry_type" <> 'Adjustment' OR ("note" IS NOT NULL AND LENGTH("note") > 0))
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_pto_ledger_period" ON "worker_pto_ledger" ("organization_id", "business_unit_id", "worker_id", "pto_type", "entry_type", "period_key")WHERE
    "period_key" IS NOT NULL;

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_pto_ledger_pto_usage" ON "worker_pto_ledger" ("source_pto_id", "organization_id", "business_unit_id", "entry_type")WHERE
    "source_pto_id" IS NOT NULL AND "entry_type" IN ('Usage', 'Reversal');

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_pto_ledger_worker_type" ON "worker_pto_ledger" ("worker_id", "pto_type", "effective_at" DESC, "sequence" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_pto_ledger_org_effective" ON "worker_pto_ledger" ("organization_id", "business_unit_id", "effective_at" DESC);

--bun:split

ALTER TABLE "worker_pto" ADD COLUMN "days" REAL NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "worker_pto" ADD COLUMN "balance_after_days" REAL;

--bun:split

ALTER TABLE "worker_pto" ADD COLUMN "auto_approved" INTEGER NOT NULL DEFAULT 0;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_pto_pending" ON "worker_pto" ("organization_id", "business_unit_id", "worker_id", "type")WHERE
    "status" = 'Requested';
