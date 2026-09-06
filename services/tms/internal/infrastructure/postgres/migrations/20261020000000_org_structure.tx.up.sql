--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
CREATE TYPE "job_department_enum" AS ENUM(
    'Operations',
    'Safety',
    'Maintenance',
    'Billing',
    'Administration',
    'Sales',
    'HumanResources',
    'Executive',
    'Other'
);

--bun:split
CREATE TYPE "approval_scope_enum" AS ENUM(
    'All',
    'TimeOff',
    'Expenses'
);

--bun:split
CREATE TABLE IF NOT EXISTS "job_positions"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "status" status_enum NOT NULL DEFAULT 'Active',
    "code" varchar(20) NOT NULL,
    "title" varchar(100) NOT NULL,
    "description" text,
    "department" job_department_enum NOT NULL DEFAULT 'Operations',
    "flsa_exempt" boolean NOT NULL DEFAULT FALSE,
    "is_driving_position" boolean NOT NULL DEFAULT TRUE,
    "reports_to_position_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_job_positions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_job_positions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_job_positions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_job_positions_reports_to" FOREIGN KEY ("reports_to_position_id", "organization_id", "business_unit_id") REFERENCES "job_positions"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_job_positions_reports_to_self" CHECK ("reports_to_position_id" IS NULL OR "reports_to_position_id" <> "id")
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_job_positions_code" ON "job_positions"("organization_id", "business_unit_id", lower("code"));

--bun:split
CREATE INDEX IF NOT EXISTS "idx_job_positions_department" ON "job_positions"("organization_id", "business_unit_id", "department");

--bun:split
COMMENT ON TABLE job_positions IS 'Job titles the roster is counted by. Separate from fleet codes: a terminal is where somebody works, a position is what they do.';

--bun:split
COMMENT ON COLUMN job_positions.reports_to_position_id IS 'The position this one reports into, which is the shape of the org chart. A person''s own manager is on the worker, because two people in the same position can report to different managers.';

--bun:split
COMMENT ON COLUMN job_positions.flsa_exempt IS 'Whether the position is exempt from overtime under the Fair Labor Standards Act. Recorded per position because that is where the duties test is applied.';

--bun:split
ALTER TABLE "workers"
    ADD COLUMN IF NOT EXISTS "position_id" varchar(100);

--bun:split
ALTER TABLE "workers"
    ADD CONSTRAINT "fk_workers_position" FOREIGN KEY ("position_id", "organization_id", "business_unit_id") REFERENCES "job_positions"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_workers_manager" ON "workers"("organization_id", "business_unit_id", "manager_id")
WHERE
    "manager_id" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_workers_position" ON "workers"("organization_id", "business_unit_id", "position_id")
WHERE
    "position_id" IS NOT NULL;

--bun:split
COMMENT ON COLUMN workers.manager_id IS 'The person who approves this worker''s time off and reads their file. A user rather than a worker: approving is something a signed-in account does, and not every manager is on the driver roster. The column has existed since the worker table; the index is what makes reading a manager''s team cheap.';

--bun:split
CREATE TABLE IF NOT EXISTS "approval_delegations"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "delegator_id" varchar(100) NOT NULL,
    "delegate_id" varchar(100) NOT NULL,
    "scope" approval_scope_enum NOT NULL DEFAULT 'All',
    "starts_at" bigint NOT NULL,
    "ends_at" bigint,
    "reason" varchar(255),
    "revoked_at" bigint,
    "revoked_by_id" varchar(100),
    "created_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_approval_delegations" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_approval_delegations_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_approval_delegations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_approval_delegations_delegator" FOREIGN KEY ("delegator_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_approval_delegations_delegate" FOREIGN KEY ("delegate_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_approval_delegations_distinct" CHECK ("delegator_id" <> "delegate_id"),
    CONSTRAINT "chk_approval_delegations_window" CHECK ("ends_at" IS NULL OR "ends_at" >= "starts_at")
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_approval_delegations_delegate" ON "approval_delegations"("organization_id", "business_unit_id", "delegate_id", "starts_at");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_approval_delegations_delegator" ON "approval_delegations"("organization_id", "business_unit_id", "delegator_id", "starts_at");

--bun:split
COMMENT ON TABLE approval_delegations IS 'Who may approve in somebody else''s place while they are away. A delegation widens what the delegate can act on; it never widens what the delegator could approve in the first place.';

--bun:split
COMMENT ON COLUMN approval_delegations.revoked_at IS 'Set when a delegation is called back before its window ends. Kept rather than deleted so an approval made under it can still be explained.';
