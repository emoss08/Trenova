-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260716120000_reports.tx.up.sql

CREATE TABLE IF NOT EXISTS "report_definitions"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "category" TEXT,
    "tags" TEXT,
    "kind" TEXT NOT NULL DEFAULT 'custom',
    "canned_key" TEXT,
    "canned_version" TEXT,
    "owner_id" TEXT NOT NULL,
    "visibility" TEXT NOT NULL DEFAULT 'private',
    "status" TEXT NOT NULL DEFAULT 'draft',
    "diagnostics" TEXT,
    "catalog_version" TEXT NOT NULL,
    "definition" TEXT NOT NULL,
    "default_format" TEXT NOT NULL DEFAULT 'csv',
    "current_revision" INTEGER NOT NULL DEFAULT 1,
    "last_run_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_report_definitions" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_report_definitions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_report_definitions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_report_definitions_owner" FOREIGN KEY ("owner_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "check_report_definitions_kind" CHECK ("kind" IN ('custom', 'canned_fork')),
    CONSTRAINT "check_report_definitions_visibility" CHECK ("visibility" IN ('private', 'shared')),
    CONSTRAINT "check_report_definitions_status" CHECK ("status" IN ('draft', 'active', 'archived', 'needs_attention')),
    CONSTRAINT "check_report_definitions_default_format" CHECK ("default_format" IN ('csv', 'xlsx', 'pdf', 'json')),
    CONSTRAINT "check_report_definitions_canned_fork_key" CHECK ("kind" != 'canned_fork' OR "canned_key" IS NOT NULL)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_report_definitions_tenant" ON "report_definitions" ("organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_report_definitions_owner" ON "report_definitions" ("owner_id", "organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_report_definitions_status" ON "report_definitions" ("status", "organization_id", "business_unit_id");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_report_definitions_name" ON "report_definitions" (lower("name"), "owner_id", "organization_id", "business_unit_id");

--bun:split

CREATE TABLE IF NOT EXISTS "report_definition_revisions"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "definition_id" TEXT NOT NULL,
    "revision_number" INTEGER NOT NULL,
    "catalog_version" TEXT NOT NULL,
    "definition" TEXT NOT NULL,
    "created_by_id" TEXT NOT NULL,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_report_definition_revisions" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_report_definition_revisions_definition" FOREIGN KEY ("definition_id", "business_unit_id", "organization_id") REFERENCES "report_definitions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_report_definition_revisions_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_report_definition_revisions_number" ON "report_definition_revisions" ("definition_id", "revision_number", "organization_id", "business_unit_id");

--bun:split

CREATE TABLE IF NOT EXISTS "report_runs"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "definition_id" TEXT,
    "revision_id" TEXT,
    "canned_key" TEXT,
    "canned_version" TEXT,
    "schedule_id" TEXT,
    "requested_by_id" TEXT NOT NULL,
    "trigger" TEXT NOT NULL DEFAULT 'manual',
    "params" TEXT,
    "format" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'queued',
    "row_count" INTEGER,
    "byte_size" INTEGER,
    "duration_ms" INTEGER,
    "truncated" INTEGER NOT NULL DEFAULT 0,
    "error" TEXT,
    "artifact_key" TEXT,
    "artifact_expires_at" INTEGER,
    "cache_hit" INTEGER NOT NULL DEFAULT 0,
    "temporal_workflow_id" TEXT,
    "temporal_run_id" TEXT,
    "queued_at" INTEGER,
    "started_at" INTEGER,
    "completed_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
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

CREATE INDEX IF NOT EXISTS "idx_report_runs_tenant_status" ON "report_runs" ("organization_id", "business_unit_id", "status");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_report_runs_definition" ON "report_runs" ("definition_id", "organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_report_runs_requested_by" ON "report_runs" ("requested_by_id", "organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_report_runs_expiry" ON "report_runs" ("artifact_expires_at")WHERE "artifact_expires_at" IS NOT NULL AND "status" = 'succeeded';

--bun:split

CREATE TABLE IF NOT EXISTS "report_schedules"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "definition_id" TEXT NOT NULL,
    "cron_expression" TEXT NOT NULL,
    "timezone" TEXT NOT NULL,
    "formats" TEXT NOT NULL,
    "delivery" TEXT,
    "enabled" INTEGER NOT NULL DEFAULT 1,
    "run_as_id" TEXT NOT NULL,
    "last_run_id" TEXT,
    "next_run_at" INTEGER,
    "consecutive_failures" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_report_schedules" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_report_schedules_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_report_schedules_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_report_schedules_run_as" FOREIGN KEY ("run_as_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_report_schedules_definition" FOREIGN KEY ("definition_id", "business_unit_id", "organization_id") REFERENCES "report_definitions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_report_schedules_tenant_enabled" ON "report_schedules" ("organization_id", "business_unit_id", "enabled");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_report_schedules_definition" ON "report_schedules" ("definition_id", "organization_id", "business_unit_id");
