-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20251118000001_workflow_system.tx.up.sql

CREATE TABLE IF NOT EXISTS "workflow_templates"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "is_template" INTEGER NOT NULL DEFAULT 0,
    "published_version_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_by_id" TEXT NOT NULL,
    "updated_by_id" TEXT NOT NULL,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_workflow_templates" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_workflow_templates_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_templates_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_templates_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users"("id") ON DELETE RESTRICT,
    CONSTRAINT "fk_workflow_templates_updated_by" FOREIGN KEY ("updated_by_id") REFERENCES "users"("id") ON DELETE RESTRICT,
    CONSTRAINT "uq_workflow_templates_name" UNIQUE ("organization_id", "business_unit_id", "name")
);

--bun:split

CREATE TABLE IF NOT EXISTS "workflow_versions"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "workflow_template_id" TEXT NOT NULL,
    "version_number" INTEGER NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "trigger_type" TEXT NOT NULL DEFAULT 'Manual',
    "status" TEXT NOT NULL DEFAULT 'Draft',
    "version_status" TEXT NOT NULL DEFAULT 'Draft',
    "schedule_config" TEXT,
    "trigger_config" TEXT,
    "change_description" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_by_id" TEXT NOT NULL,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_workflow_versions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_workflow_versions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_versions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_versions_workflow_template" FOREIGN KEY ("workflow_template_id", "organization_id", "business_unit_id") REFERENCES "workflow_templates"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_versions_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users"("id") ON DELETE RESTRICT,
    CONSTRAINT "uq_workflow_versions_number" UNIQUE ("workflow_template_id", "organization_id", "business_unit_id", "version_number"),
    CONSTRAINT "chk_workflow_versions_positive_number" CHECK ("version_number" > 0)
);

--bun:split

CREATE TABLE IF NOT EXISTS "workflow_nodes"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "workflow_version_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "node_type" TEXT NOT NULL,
    "position_x" INTEGER NOT NULL DEFAULT 0,
    "position_y" INTEGER NOT NULL DEFAULT 0,
    "config" TEXT NOT NULL DEFAULT '{}',
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_workflow_nodes" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_workflow_nodes_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_nodes_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_nodes_workflow_version" FOREIGN KEY ("workflow_version_id", "organization_id", "business_unit_id") REFERENCES "workflow_versions"("id", "organization_id", "business_unit_id") ON DELETE CASCADE
);

--bun:split

CREATE TABLE IF NOT EXISTS "workflow_connections"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "workflow_version_id" TEXT NOT NULL,
    "source_node_id" TEXT NOT NULL,
    "target_node_id" TEXT NOT NULL,
    "condition" TEXT,
    "is_default_branch" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_workflow_connections" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_workflow_connections_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_connections_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_connections_workflow_version" FOREIGN KEY ("workflow_version_id", "organization_id", "business_unit_id") REFERENCES "workflow_versions"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_connections_source_node" FOREIGN KEY ("source_node_id", "organization_id", "business_unit_id") REFERENCES "workflow_nodes"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_connections_target_node" FOREIGN KEY ("target_node_id", "organization_id", "business_unit_id") REFERENCES "workflow_nodes"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "chk_workflow_connections_no_self_link" CHECK ("source_node_id" != "target_node_id")
);

--bun:split

CREATE TABLE IF NOT EXISTS "workflow_instances"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "workflow_template_id" TEXT NOT NULL,
    "workflow_version_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Running',
    "execution_mode" TEXT NOT NULL DEFAULT 'Manual',
    "trigger_payload" TEXT,
    "workflow_variables" TEXT,
    "execution_context" TEXT,
    "error_message" TEXT,
    "started_by_id" TEXT,
    "started_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "completed_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_workflow_instances" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_workflow_instances_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_instances_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_instances_workflow_template" FOREIGN KEY ("workflow_template_id", "organization_id", "business_unit_id") REFERENCES "workflow_templates"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_instances_workflow_version" FOREIGN KEY ("workflow_version_id", "organization_id", "business_unit_id") REFERENCES "workflow_versions"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_workflow_instances_started_by" FOREIGN KEY ("started_by_id") REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "chk_workflow_instances_completed_at" CHECK (("status" IN ('Completed', 'Failed', 'Cancelled') AND "completed_at" IS NOT NULL) OR ("status" IN ('Running', 'Paused') AND "completed_at" IS NULL))
);

--bun:split

CREATE TABLE IF NOT EXISTS "workflow_node_executions"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "workflow_instance_id" TEXT NOT NULL,
    "workflow_node_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "attempt_count" INTEGER NOT NULL DEFAULT 0,
    "input_data" TEXT,
    "output_data" TEXT,
    "error_details" TEXT,
    "started_at" INTEGER,
    "completed_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_workflow_node_executions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_workflow_node_executions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_node_executions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_node_executions_workflow_instance" FOREIGN KEY ("workflow_instance_id", "organization_id", "business_unit_id") REFERENCES "workflow_instances"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_node_executions_workflow_node" FOREIGN KEY ("workflow_node_id", "organization_id", "business_unit_id") REFERENCES "workflow_nodes"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "chk_workflow_node_executions_started_at" CHECK (("status" != 'Pending' AND "started_at" IS NOT NULL) OR ("status" = 'Pending')),
    CONSTRAINT "chk_workflow_node_executions_completed_at" CHECK (("status" IN ('Completed', 'Failed', 'Skipped') AND "completed_at" IS NOT NULL) OR ("status" IN ('Pending', 'Running')))
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_templates_bu_org ON "workflow_templates" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_templates_created_updated ON "workflow_templates" ("created_at", "updated_at");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_templates_published_version ON "workflow_templates" ("published_version_id")WHERE
    "published_version_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_templates_is_template ON "workflow_templates" ("is_template")WHERE
    "is_template" = TRUE;

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_versions_bu_org ON "workflow_versions" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_versions_created_updated ON "workflow_versions" ("created_at", "updated_at");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_versions_template ON "workflow_versions" ("workflow_template_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_versions_status ON "workflow_versions" ("status");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_versions_version_status ON "workflow_versions" ("version_status");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_versions_published ON "workflow_versions" ("workflow_template_id", "version_status")WHERE
    "version_status" = 'Published';

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_versions_number ON "workflow_versions" ("workflow_template_id", "version_number");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_nodes_bu_org ON "workflow_nodes" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_nodes_created_updated ON "workflow_nodes" ("created_at", "updated_at");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_nodes_workflow_version ON "workflow_nodes" ("workflow_version_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_nodes_type ON "workflow_nodes" ("node_type");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_connections_bu_org ON "workflow_connections" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_connections_created_updated ON "workflow_connections" ("created_at", "updated_at");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_connections_workflow_version ON "workflow_connections" ("workflow_version_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_connections_source ON "workflow_connections" ("source_node_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_connections_target ON "workflow_connections" ("target_node_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_instances_bu_org ON "workflow_instances" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_instances_created_updated ON "workflow_instances" ("created_at", "updated_at");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_instances_workflow_template ON "workflow_instances" ("workflow_template_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_instances_workflow_version ON "workflow_instances" ("workflow_version_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_instances_status ON "workflow_instances" ("status");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_instances_execution_mode ON "workflow_instances" ("execution_mode");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_instances_started_at ON "workflow_instances" ("started_at");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_instances_completed_at ON "workflow_instances" ("completed_at")WHERE
    "completed_at" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_instances_running ON "workflow_instances" ("status")WHERE
    "status" = 'Running';

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_node_executions_bu_org ON "workflow_node_executions" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_node_executions_created_updated ON "workflow_node_executions" ("created_at", "updated_at");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_node_executions_instance ON "workflow_node_executions" ("workflow_instance_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_node_executions_node ON "workflow_node_executions" ("workflow_node_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_workflow_node_executions_status ON "workflow_node_executions" ("status");
