--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TABLE IF NOT EXISTS "report_definitions"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "name" varchar(255) NOT NULL,
    "description" text,
    "category" varchar(100),
    "tags" text[],
    "kind" varchar(20) NOT NULL DEFAULT 'custom',
    "canned_key" varchar(100),
    "canned_version" varchar(20),
    "owner_id" varchar(100) NOT NULL,
    "visibility" varchar(20) NOT NULL DEFAULT 'private',
    "status" varchar(20) NOT NULL DEFAULT 'draft',
    "diagnostics" text[],
    "catalog_version" varchar(80) NOT NULL,
    "definition" jsonb NOT NULL,
    "default_format" varchar(10) NOT NULL DEFAULT 'csv',
    "current_revision" bigint NOT NULL DEFAULT 1,
    "last_run_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_report_definitions" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_report_definitions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_report_definitions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_report_definitions_owner" FOREIGN KEY ("owner_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "check_report_definitions_kind" CHECK ("kind" IN ('custom', 'canned_fork')),
    CONSTRAINT "check_report_definitions_visibility" CHECK ("visibility" IN ('private', 'shared')),
    CONSTRAINT "check_report_definitions_status" CHECK ("status" IN ('draft', 'active', 'archived', 'needs_attention')),
    CONSTRAINT "check_report_definitions_default_format" CHECK ("default_format" IN ('csv', 'xlsx', 'pdf', 'json')),
    CONSTRAINT "check_report_definitions_definition_format" CHECK (jsonb_typeof(definition) = 'object'),
    CONSTRAINT "check_report_definitions_canned_fork_key" CHECK ("kind" != 'canned_fork' OR "canned_key" IS NOT NULL)
);

--bun:split
CREATE INDEX "idx_report_definitions_tenant" ON "report_definitions"("organization_id", "business_unit_id");

--bun:split
CREATE INDEX "idx_report_definitions_owner" ON "report_definitions"("owner_id", "organization_id", "business_unit_id");

--bun:split
CREATE INDEX "idx_report_definitions_status" ON "report_definitions"("status", "organization_id", "business_unit_id");

--bun:split
CREATE UNIQUE INDEX "idx_report_definitions_name" ON "report_definitions"(lower("name"), "owner_id", "organization_id", "business_unit_id");

--bun:split
COMMENT ON TABLE report_definitions IS 'Saved report definitions (custom reports and per-tenant forks of canned reports)';

--bun:split
CREATE TABLE IF NOT EXISTS "report_definition_revisions"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "definition_id" varchar(100) NOT NULL,
    "revision_number" bigint NOT NULL,
    "catalog_version" varchar(80) NOT NULL,
    "definition" jsonb NOT NULL,
    "created_by_id" varchar(100) NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_report_definition_revisions" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_report_definition_revisions_definition" FOREIGN KEY ("definition_id", "business_unit_id", "organization_id") REFERENCES "report_definitions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_report_definition_revisions_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "check_report_definition_revisions_definition_format" CHECK (jsonb_typeof(definition) = 'object')
);

--bun:split
CREATE UNIQUE INDEX "idx_report_definition_revisions_number" ON "report_definition_revisions"("definition_id", "revision_number", "organization_id", "business_unit_id");

--bun:split
COMMENT ON TABLE report_definition_revisions IS 'Append-only snapshots of report definitions; runs bind to a revision for reproducibility';

--bun:split
CREATE TABLE IF NOT EXISTS "report_runs"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "definition_id" varchar(100),
    "revision_id" varchar(100),
    "canned_key" varchar(100),
    "canned_version" varchar(20),
    "schedule_id" varchar(100),
    "requested_by_id" varchar(100) NOT NULL,
    "trigger" varchar(20) NOT NULL DEFAULT 'manual',
    "params" jsonb,
    "format" varchar(10) NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'queued',
    "row_count" bigint,
    "byte_size" bigint,
    "duration_ms" bigint,
    "truncated" boolean NOT NULL DEFAULT FALSE,
    "error" jsonb,
    "artifact_key" varchar(512),
    "artifact_expires_at" bigint,
    "cache_hit" boolean NOT NULL DEFAULT FALSE,
    "temporal_workflow_id" varchar(255),
    "temporal_run_id" varchar(255),
    "queued_at" bigint,
    "started_at" bigint,
    "completed_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_report_runs" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_report_runs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_report_runs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_report_runs_requested_by" FOREIGN KEY ("requested_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_report_runs_definition" FOREIGN KEY ("definition_id", "business_unit_id", "organization_id") REFERENCES "report_definitions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "check_report_runs_trigger" CHECK ("trigger" IN ('manual', 'scheduled', 'api')),
    CONSTRAINT "check_report_runs_status" CHECK ("status" IN ('queued', 'running', 'succeeded', 'failed', 'canceled', 'expired')),
    CONSTRAINT "check_report_runs_format" CHECK ("format" IN ('csv', 'xlsx', 'pdf', 'json')),
    CONSTRAINT "check_report_runs_source" CHECK ("definition_id" IS NOT NULL OR "canned_key" IS NOT NULL)
);

--bun:split
CREATE INDEX "idx_report_runs_tenant_status" ON "report_runs"("organization_id", "business_unit_id", "status");

--bun:split
CREATE INDEX "idx_report_runs_definition" ON "report_runs"("definition_id", "organization_id", "business_unit_id");

--bun:split
CREATE INDEX "idx_report_runs_requested_by" ON "report_runs"("requested_by_id", "organization_id", "business_unit_id");

--bun:split
CREATE INDEX "idx_report_runs_expiry" ON "report_runs"("artifact_expires_at") WHERE "artifact_expires_at" IS NOT NULL AND "status" = 'succeeded';

--bun:split
COMMENT ON TABLE report_runs IS 'Individual report generation runs with status, artifact location, and diagnostics';

--bun:split
CREATE TABLE IF NOT EXISTS "report_schedules"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "definition_id" varchar(100) NOT NULL,
    "cron_expression" varchar(100) NOT NULL,
    "timezone" varchar(64) NOT NULL,
    "formats" text[] NOT NULL,
    "delivery" jsonb,
    "enabled" boolean NOT NULL DEFAULT TRUE,
    "run_as_id" varchar(100) NOT NULL,
    "last_run_id" varchar(100),
    "next_run_at" bigint,
    "consecutive_failures" integer NOT NULL DEFAULT 0,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_report_schedules" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_report_schedules_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_report_schedules_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_report_schedules_run_as" FOREIGN KEY ("run_as_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_report_schedules_definition" FOREIGN KEY ("definition_id", "business_unit_id", "organization_id") REFERENCES "report_definitions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE INDEX "idx_report_schedules_tenant_enabled" ON "report_schedules"("organization_id", "business_unit_id", "enabled");

--bun:split
CREATE INDEX "idx_report_schedules_definition" ON "report_schedules"("definition_id", "organization_id", "business_unit_id");

--bun:split
COMMENT ON TABLE report_schedules IS 'Recurring report schedules reconciled into Temporal Schedules';
