--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TYPE "pto_policy_status_enum" AS ENUM(
    'Active',
    'Inactive',
    'Draft'
);

CREATE TYPE "pto_year_basis_enum" AS ENUM(
    'CalendarYear',
    'HireAnniversary'
);

CREATE TYPE "pto_accrual_method_enum" AS ENUM(
    'None',
    'FixedAnnualGrant',
    'Monthly'
);

CREATE TYPE "pto_ledger_entry_type_enum" AS ENUM(
    'OpeningBalance',
    'Accrual',
    'Usage',
    'Reversal',
    'Adjustment',
    'Carryover',
    'Expiry'
);

CREATE TYPE "pto_ledger_actor_type_enum" AS ENUM(
    'User',
    'System'
);

--bun:split
CREATE TABLE IF NOT EXISTS "pto_policies"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "name" varchar(100) NOT NULL,
    "code" varchar(50) NOT NULL,
    "description" text,
    "status" pto_policy_status_enum NOT NULL DEFAULT 'Draft',
    "is_default" boolean NOT NULL DEFAULT FALSE,
    "year_basis" pto_year_basis_enum NOT NULL DEFAULT 'CalendarYear',
    "count_weekends" boolean NOT NULL DEFAULT TRUE,
    "waiting_period_days" integer NOT NULL DEFAULT 0,
    "requires_approval" boolean NOT NULL DEFAULT TRUE,
    "enforce_balance" boolean NOT NULL DEFAULT TRUE,
    "allow_negative" boolean NOT NULL DEFAULT FALSE,
    "negative_floor_days" numeric(6, 2) NOT NULL DEFAULT 0,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_pto_policies" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_pto_policies_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_pto_policies_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_pto_policies_waiting_period" CHECK ("waiting_period_days" >= 0),
    CONSTRAINT "chk_pto_policies_negative_floor" CHECK ("negative_floor_days" <= 0),
    CONSTRAINT "chk_pto_policies_negative_requires_floor" CHECK (NOT "allow_negative" OR "negative_floor_days" < 0)
);

--bun:split
ALTER TABLE "pto_policies"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (setweight(immutable_to_tsvector('simple', COALESCE("code", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("name", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("description", '')), 'B')) STORED;

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_pto_policies_code" ON "pto_policies"("organization_id", "business_unit_id", LOWER("code"));

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_pto_policies_default" ON "pto_policies"("organization_id", "business_unit_id")
WHERE
    "is_default";

--bun:split
CREATE INDEX IF NOT EXISTS "idx_pto_policies_status" ON "pto_policies"("organization_id", "business_unit_id", "status");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_pto_policies_search" ON "pto_policies" USING GIN(search_vector);

--bun:split
COMMENT ON TABLE pto_policies IS 'Paid-time-off accrual policies. A worker is assigned at most one open policy; each policy carries per-type accrual rules.';

--bun:split
CREATE TABLE IF NOT EXISTS "pto_policy_rules"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "pto_policy_id" varchar(100) NOT NULL,
    "pto_type" worker_pto_type_enum NOT NULL,
    "accrual_method" pto_accrual_method_enum NOT NULL DEFAULT 'None',
    "accrual_amount_days" numeric(6, 2) NOT NULL DEFAULT 0,
    "max_balance_days" numeric(6, 2),
    "carryover_cap_days" numeric(6, 2),
    "carryover_expiry_days" integer NOT NULL DEFAULT 0,
    "sort_order" integer NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
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
CREATE INDEX IF NOT EXISTS "idx_pto_policy_rules_policy" ON "pto_policy_rules"("pto_policy_id", "sort_order");

--bun:split
COMMENT ON TABLE pto_policy_rules IS 'Per-PTO-type accrual rule for a policy. Types without a rule are untracked: no ledger, no balance enforcement.';

--bun:split
CREATE TABLE IF NOT EXISTS "worker_pto_policy_assignments"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "pto_policy_id" varchar(100) NOT NULL,
    "effective_from" bigint NOT NULL,
    "effective_to" bigint,
    "assigned_by_id" varchar(100),
    "note" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_worker_pto_policy_assignments" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_pto_policy_assignments_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_pto_policy_assignments_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_pto_policy_assignments_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_pto_policy_assignments_policy" FOREIGN KEY ("pto_policy_id", "organization_id", "business_unit_id") REFERENCES "pto_policies"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_worker_pto_policy_assignments_range" CHECK ("effective_to" IS NULL OR "effective_to" > "effective_from")
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_pto_policy_assignments_open" ON "worker_pto_policy_assignments"("organization_id", "business_unit_id", "worker_id")
WHERE
    "effective_to" IS NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_pto_policy_assignments_worker" ON "worker_pto_policy_assignments"("worker_id", "effective_from" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_pto_policy_assignments_policy_open" ON "worker_pto_policy_assignments"("pto_policy_id")
WHERE
    "effective_to" IS NULL;

--bun:split
COMMENT ON TABLE worker_pto_policy_assignments IS 'History of which PTO policy governs a worker. At most one row per worker is open (effective_to IS NULL).';

--bun:split
CREATE TABLE IF NOT EXISTS "worker_pto_balances"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "pto_type" worker_pto_type_enum NOT NULL,
    "balance_days" numeric(8, 2) NOT NULL DEFAULT 0,
    "accrued_ytd_days" numeric(8, 2) NOT NULL DEFAULT 0,
    "used_ytd_days" numeric(8, 2) NOT NULL DEFAULT 0,
    "carried_days" numeric(8, 2) NOT NULL DEFAULT 0,
    "entry_count" bigint NOT NULL DEFAULT 0,
    "last_accrual_period_key" varchar(64),
    "last_rollover_key" varchar(64),
    "year_started_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_worker_pto_balances" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_pto_balances_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_pto_balances_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_pto_balances_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_worker_pto_balances_worker_type" UNIQUE ("organization_id", "business_unit_id", "worker_id", "pto_type")
);

--bun:split
COMMENT ON TABLE worker_pto_balances IS 'Materialised running balance per worker and PTO type. The ledger is the source of truth; this row is the lock target that serialises accrual against approvals.';

--bun:split
CREATE TABLE IF NOT EXISTS "worker_pto_ledger"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "worker_id" varchar(100) NOT NULL,
    "pto_type" worker_pto_type_enum NOT NULL,
    "entry_type" pto_ledger_entry_type_enum NOT NULL,
    "amount_days" numeric(8, 2) NOT NULL,
    "balance_after_days" numeric(8, 2) NOT NULL,
    "sequence" bigint NOT NULL,
    "effective_at" bigint NOT NULL,
    "period_key" varchar(64),
    "source_pto_id" varchar(100),
    "assignment_id" varchar(100),
    "pto_policy_id" varchar(100),
    "actor_type" pto_ledger_actor_type_enum NOT NULL DEFAULT 'User',
    "created_by_id" varchar(100),
    "note" varchar(255),
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
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
CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_pto_ledger_period" ON "worker_pto_ledger"("organization_id", "business_unit_id", "worker_id", "pto_type", "entry_type", "period_key")
WHERE
    "period_key" IS NOT NULL;

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_pto_ledger_pto_usage" ON "worker_pto_ledger"("source_pto_id", "organization_id", "business_unit_id", "entry_type")
WHERE
    "source_pto_id" IS NOT NULL AND "entry_type" IN ('Usage', 'Reversal');

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_pto_ledger_worker_type" ON "worker_pto_ledger"("worker_id", "pto_type", "effective_at" DESC, "sequence" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_pto_ledger_org_effective" ON "worker_pto_ledger"("organization_id", "business_unit_id", "effective_at" DESC);

--bun:split
COMMENT ON TABLE worker_pto_ledger IS 'Append-only PTO ledger in days. Accruals are keyed by period so a re-run never double-posts; Usage/Reversal are keyed by the PTO request.';

--bun:split
ALTER TABLE "worker_pto"
    ADD COLUMN IF NOT EXISTS "days" numeric(6, 2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "balance_after_days" numeric(8, 2),
    ADD COLUMN IF NOT EXISTS "auto_approved" boolean NOT NULL DEFAULT FALSE;

--bun:split
UPDATE
    "worker_pto"
SET
    "days" = GREATEST(1, FLOOR("end_date" / 86400) - FLOOR("start_date" / 86400) + 1)
WHERE
    "days" = 0;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_worker_pto_pending" ON "worker_pto"("organization_id", "business_unit_id", "worker_id", "type")
WHERE
    "status" = 'Requested';
