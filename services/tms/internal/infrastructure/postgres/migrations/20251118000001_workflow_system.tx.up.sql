SET statement_timeout = 0;

-- Create workflow trigger type enum
CREATE TYPE workflow_trigger_type_enum AS ENUM(
    'Manual',
    'Scheduled',
    'Event'
);

--bun:split
-- Create workflow status enum
CREATE TYPE workflow_status_enum AS ENUM(
    'Active',
    'Inactive',
    'Draft'
);

--bun:split
-- Create workflow version status enum
CREATE TYPE workflow_version_status_enum AS ENUM(
    'Draft',
    'Published',
    'Archived'
);

--bun:split
-- Create workflow instance status enum
CREATE TYPE workflow_instance_status_enum AS ENUM(
    'Running',
    'Completed',
    'Failed',
    'Cancelled',
    'Paused'
);

--bun:split
-- Create workflow node type enum
CREATE TYPE workflow_node_type_enum AS ENUM(
    'Trigger',
    'EntityUpdate',
    'Condition'
);

--bun:split
-- Create workflow node execution status enum
CREATE TYPE workflow_node_execution_status_enum AS ENUM(
    'Pending',
    'Running',
    'Completed',
    'Failed',
    'Skipped'
);

--bun:split
-- Create workflow execution mode enum
CREATE TYPE workflow_execution_mode_enum AS ENUM(
    'Manual',
    'Scheduled',
    'Event'
);

--bun:split
-- Create workflow_templates table (stable workflow concept)
CREATE TABLE IF NOT EXISTS "workflow_templates"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "name" varchar(255) NOT NULL,
    "description" text,
    "is_template" boolean NOT NULL DEFAULT FALSE,
    "published_version_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_by_id" varchar(100) NOT NULL,
    "updated_by_id" varchar(100) NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_workflow_templates" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_workflow_templates_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_templates_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_templates_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users"("id") ON DELETE RESTRICT,
    CONSTRAINT "fk_workflow_templates_updated_by" FOREIGN KEY ("updated_by_id") REFERENCES "users"("id") ON DELETE RESTRICT,
    CONSTRAINT "uq_workflow_templates_name" UNIQUE ("organization_id", "business_unit_id", "name")
);

--bun:split
-- Create workflow_versions table (version-specific configuration)
CREATE TABLE IF NOT EXISTS "workflow_versions"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "workflow_template_id" varchar(100) NOT NULL,
    "version_number" integer NOT NULL,
    "name" varchar(255) NOT NULL,
    "description" text,
    "trigger_type" workflow_trigger_type_enum NOT NULL DEFAULT 'Manual',
    "status" workflow_status_enum NOT NULL DEFAULT 'Draft',
    "version_status" workflow_version_status_enum NOT NULL DEFAULT 'Draft',
    "schedule_config" jsonb DEFAULT '{}'::jsonb,
    "trigger_config" jsonb DEFAULT '{}'::jsonb,
    "change_description" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_by_id" varchar(100) NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_workflow_versions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_workflow_versions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_versions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_versions_workflow_template" FOREIGN KEY ("workflow_template_id", "organization_id", "business_unit_id") REFERENCES "workflow_templates"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_versions_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users"("id") ON DELETE RESTRICT,
    CONSTRAINT "uq_workflow_versions_number" UNIQUE ("workflow_template_id", "organization_id", "business_unit_id", "version_number"),
    CONSTRAINT "chk_workflow_versions_positive_number" CHECK ("version_number" > 0)
);

--bun:split
-- Add foreign key from workflow_templates to workflow_versions for published_version_id
-- Note: This is added after workflow_versions table is created to avoid circular dependency
ALTER TABLE "workflow_templates"
    ADD CONSTRAINT "fk_workflow_templates_published_version"
    FOREIGN KEY ("published_version_id", "organization_id", "business_unit_id")
    REFERENCES "workflow_versions"("id", "organization_id", "business_unit_id")
    ON DELETE SET NULL;

--bun:split
-- Create workflow nodes table
CREATE TABLE IF NOT EXISTS "workflow_nodes"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "workflow_version_id" varchar(100) NOT NULL,
    "name" varchar(255) NOT NULL,
    "node_type" workflow_node_type_enum NOT NULL,
    "position_x" integer NOT NULL DEFAULT 0,
    "position_y" integer NOT NULL DEFAULT 0,
    "config" jsonb NOT NULL DEFAULT '{}'::jsonb,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_workflow_nodes" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_workflow_nodes_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_nodes_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_nodes_workflow_version" FOREIGN KEY ("workflow_version_id", "organization_id", "business_unit_id") REFERENCES "workflow_versions"("id", "organization_id", "business_unit_id") ON DELETE CASCADE
);

--bun:split
-- Create workflow connections table (edges)
CREATE TABLE IF NOT EXISTS "workflow_connections"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "workflow_version_id" varchar(100) NOT NULL,
    "source_node_id" varchar(100) NOT NULL,
    "target_node_id" varchar(100) NOT NULL,
    "condition" jsonb DEFAULT NULL,
    "is_default_branch" boolean NOT NULL DEFAULT FALSE,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_workflow_connections" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_workflow_connections_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_connections_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_connections_workflow_version" FOREIGN KEY ("workflow_version_id", "organization_id", "business_unit_id") REFERENCES "workflow_versions"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_connections_source_node" FOREIGN KEY ("source_node_id", "organization_id", "business_unit_id") REFERENCES "workflow_nodes"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_connections_target_node" FOREIGN KEY ("target_node_id", "organization_id", "business_unit_id") REFERENCES "workflow_nodes"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "chk_workflow_connections_no_self_link" CHECK ("source_node_id" != "target_node_id")
);

--bun:split
-- Create workflow instances table (execution state)
CREATE TABLE IF NOT EXISTS "workflow_instances"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "workflow_template_id" varchar(100) NOT NULL,
    "workflow_version_id" varchar(100) NOT NULL,
    "status" workflow_instance_status_enum NOT NULL DEFAULT 'Running',
    "execution_mode" workflow_execution_mode_enum NOT NULL DEFAULT 'Manual',
    "trigger_payload" jsonb DEFAULT '{}'::jsonb,
    "workflow_variables" jsonb DEFAULT '{}'::jsonb,
    "execution_context" jsonb DEFAULT '{}'::jsonb,
    "error_message" text,
    "started_by_id" varchar(100),
    "started_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "completed_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_workflow_instances" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_workflow_instances_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_instances_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_instances_workflow_template" FOREIGN KEY ("workflow_template_id", "organization_id", "business_unit_id") REFERENCES "workflow_templates"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_instances_workflow_version" FOREIGN KEY ("workflow_version_id", "organization_id", "business_unit_id") REFERENCES "workflow_versions"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_workflow_instances_started_by" FOREIGN KEY ("started_by_id") REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "chk_workflow_instances_completed_at" CHECK (("status" IN ('Completed', 'Failed', 'Cancelled') AND "completed_at" IS NOT NULL) OR ("status" IN ('Running', 'Paused') AND "completed_at" IS NULL))
);

--bun:split
-- Create workflow node executions table (node execution state)
CREATE TABLE IF NOT EXISTS "workflow_node_executions"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "workflow_instance_id" varchar(100) NOT NULL,
    "workflow_node_id" varchar(100) NOT NULL,
    "status" workflow_node_execution_status_enum NOT NULL DEFAULT 'Pending',
    "attempt_count" smallint NOT NULL DEFAULT 0,
    "input_data" jsonb DEFAULT '{}'::jsonb,
    "output_data" jsonb DEFAULT '{}'::jsonb,
    "error_details" jsonb DEFAULT NULL,
    "started_at" bigint,
    "completed_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_workflow_node_executions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_workflow_node_executions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_node_executions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_node_executions_workflow_instance" FOREIGN KEY ("workflow_instance_id", "organization_id", "business_unit_id") REFERENCES "workflow_instances"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_node_executions_workflow_node" FOREIGN KEY ("workflow_node_id", "organization_id", "business_unit_id") REFERENCES "workflow_nodes"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "chk_workflow_node_executions_started_at" CHECK (("status" != 'Pending' AND "started_at" IS NOT NULL) OR ("status" = 'Pending')),
    CONSTRAINT "chk_workflow_node_executions_completed_at" CHECK (("status" IN ('Completed', 'Failed', 'Skipped') AND "completed_at" IS NOT NULL) OR ("status" IN ('Pending', 'Running')))
);

--bun:split
-- Indexes for workflow_templates
CREATE INDEX IF NOT EXISTS idx_workflow_templates_bu_org ON "workflow_templates"("business_unit_id", "organization_id");

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_templates_created_updated ON "workflow_templates"("created_at", "updated_at");

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_templates_published_version ON "workflow_templates"("published_version_id")
WHERE
    "published_version_id" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_templates_is_template ON "workflow_templates"("is_template")
WHERE
    "is_template" = TRUE;

--bun:split
-- Indexes for workflow_versions
CREATE INDEX IF NOT EXISTS idx_workflow_versions_bu_org ON "workflow_versions"("business_unit_id", "organization_id");

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_versions_created_updated ON "workflow_versions"("created_at", "updated_at");

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_versions_template ON "workflow_versions"("workflow_template_id");

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_versions_status ON "workflow_versions"("status");

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_versions_version_status ON "workflow_versions"("version_status");

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_versions_published ON "workflow_versions"("workflow_template_id", "version_status")
WHERE
    "version_status" = 'Published';

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_versions_number ON "workflow_versions"("workflow_template_id", "version_number");

--bun:split
-- Indexes for workflow_nodes
CREATE INDEX IF NOT EXISTS idx_workflow_nodes_bu_org ON "workflow_nodes"("business_unit_id", "organization_id");

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_nodes_created_updated ON "workflow_nodes"("created_at", "updated_at");

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_nodes_workflow_version ON "workflow_nodes"("workflow_version_id");

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_nodes_type ON "workflow_nodes"("node_type");

--bun:split
-- Indexes for workflow_connections
CREATE INDEX IF NOT EXISTS idx_workflow_connections_bu_org ON "workflow_connections"("business_unit_id", "organization_id");

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_connections_created_updated ON "workflow_connections"("created_at", "updated_at");

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_connections_workflow_version ON "workflow_connections"("workflow_version_id");

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_connections_source ON "workflow_connections"("source_node_id");

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_connections_target ON "workflow_connections"("target_node_id");

--bun:split
-- Indexes for workflow_instances
CREATE INDEX IF NOT EXISTS idx_workflow_instances_bu_org ON "workflow_instances"("business_unit_id", "organization_id");

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_instances_created_updated ON "workflow_instances"("created_at", "updated_at");

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_instances_workflow_template ON "workflow_instances"("workflow_template_id");

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_instances_workflow_version ON "workflow_instances"("workflow_version_id");

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_instances_status ON "workflow_instances"("status");

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_instances_execution_mode ON "workflow_instances"("execution_mode");

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_instances_started_at ON "workflow_instances"("started_at");

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_instances_completed_at ON "workflow_instances"("completed_at")
WHERE
    "completed_at" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_instances_running ON "workflow_instances"("status")
WHERE
    "status" = 'Running';

--bun:split
-- Indexes for workflow_node_executions
CREATE INDEX IF NOT EXISTS idx_workflow_node_executions_bu_org ON "workflow_node_executions"("business_unit_id", "organization_id");

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_node_executions_created_updated ON "workflow_node_executions"("created_at", "updated_at");

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_node_executions_instance ON "workflow_node_executions"("workflow_instance_id");

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_node_executions_node ON "workflow_node_executions"("workflow_node_id");

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_node_executions_status ON "workflow_node_executions"("status");

--bun:split
-- Comments for workflow_templates
COMMENT ON TABLE "workflow_templates" IS 'Stores workflow template definitions (stable workflow concepts)';

--bun:split
COMMENT ON COLUMN "workflow_templates"."published_version_id" IS 'Currently active/published version of this workflow';

--bun:split
COMMENT ON COLUMN "workflow_templates"."is_template" IS 'Whether this workflow is shareable as a template';

--bun:split
-- Comments for workflow_versions
COMMENT ON TABLE "workflow_versions" IS 'Stores version-specific workflow configurations';

--bun:split
COMMENT ON COLUMN "workflow_versions"."version_number" IS 'Auto-incrementing version number per template (1, 2, 3...)';

--bun:split
COMMENT ON COLUMN "workflow_versions"."version_status" IS 'Version lifecycle status: Draft, Published, or Archived';

--bun:split
COMMENT ON COLUMN "workflow_versions"."change_description" IS 'User-provided description of what changed in this version';

--bun:split
COMMENT ON COLUMN "workflow_versions"."trigger_type" IS 'How the workflow is triggered: Manual, Scheduled, or Event-based';

--bun:split
COMMENT ON COLUMN "workflow_versions"."schedule_config" IS 'Cron expression and schedule settings for scheduled workflows';

--bun:split
COMMENT ON COLUMN "workflow_versions"."trigger_config" IS 'Event filters and trigger conditions for event-based workflows';

--bun:split
-- Comments for workflow_nodes
COMMENT ON TABLE "workflow_nodes" IS 'Stores individual nodes (steps) within a workflow version';

--bun:split
COMMENT ON COLUMN "workflow_nodes"."workflow_version_id" IS 'References the specific workflow version this node belongs to';

--bun:split
COMMENT ON COLUMN "workflow_nodes"."node_type" IS 'Type of action: Trigger, EntityUpdate, or Condition';

--bun:split
COMMENT ON COLUMN "workflow_nodes"."position_x" IS 'X coordinate for visual workflow editor';

--bun:split
COMMENT ON COLUMN "workflow_nodes"."position_y" IS 'Y coordinate for visual workflow editor';

--bun:split
COMMENT ON COLUMN "workflow_nodes"."config" IS 'Node-specific configuration (entity type, fields, mappings, etc.)';

--bun:split
-- Comments for workflow_connections
COMMENT ON TABLE "workflow_connections" IS 'Stores connections (edges) between workflow nodes';

--bun:split
COMMENT ON COLUMN "workflow_connections"."condition" IS 'Conditional logic for branching (field, operator, value)';

--bun:split
COMMENT ON COLUMN "workflow_connections"."is_default_branch" IS 'Whether this is the default/else branch when condition fails';

--bun:split
-- Comments for workflow_instances
COMMENT ON TABLE "workflow_instances" IS 'Stores workflow execution instances with runtime state';

--bun:split
COMMENT ON COLUMN "workflow_instances"."workflow_template_id" IS 'Which workflow template was executed';

--bun:split
COMMENT ON COLUMN "workflow_instances"."workflow_version_id" IS 'Which specific version was executed (audit trail)';

--bun:split
COMMENT ON COLUMN "workflow_instances"."trigger_payload" IS 'Initial data that triggered the workflow execution';

--bun:split
COMMENT ON COLUMN "workflow_instances"."workflow_variables" IS 'Global variables accessible by all nodes during execution';

--bun:split
COMMENT ON COLUMN "workflow_instances"."execution_context" IS 'Runtime context (user, timestamp, metadata, etc.)';

--bun:split
-- Comments for workflow_node_executions
COMMENT ON TABLE "workflow_node_executions" IS 'Stores individual node execution state within workflow instances';

--bun:split
COMMENT ON COLUMN "workflow_node_executions"."attempt_count" IS 'Number of execution attempts (for retry logic)';

--bun:split
COMMENT ON COLUMN "workflow_node_executions"."input_data" IS 'Input data provided to this node';

--bun:split
COMMENT ON COLUMN "workflow_node_executions"."output_data" IS 'Output data produced by this node (passed to subsequent nodes)';

--bun:split
COMMENT ON COLUMN "workflow_node_executions"."error_details" IS 'Error information if execution failed';

--bun:split
-- Search Vector for workflow_templates
ALTER TABLE "workflow_templates"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

--bun:split
CREATE INDEX IF NOT EXISTS idx_workflow_templates_search_vector ON "workflow_templates" USING GIN(search_vector);

--bun:split
CREATE OR REPLACE FUNCTION workflow_templates_search_trigger()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.search_vector := setweight(to_tsvector('english', COALESCE(NEW.name, '')), 'A') || setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B');
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS workflow_templates_search_update ON "workflow_templates";

--bun:split
CREATE TRIGGER workflow_templates_search_update
    BEFORE INSERT OR UPDATE ON "workflow_templates"
    FOR EACH ROW
    EXECUTE FUNCTION workflow_templates_search_trigger();

--bun:split
UPDATE
    "workflow_templates"
SET
    search_vector = setweight(to_tsvector('english', COALESCE(name, '')), 'A') || setweight(to_tsvector('english', COALESCE(description, '')), 'B');

--bun:split
-- Statistics for query optimization
ALTER TABLE "workflow_templates"
    ALTER COLUMN "organization_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "workflow_templates"
    ALTER COLUMN "business_unit_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "workflow_templates"
    ALTER COLUMN "published_version_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "workflow_versions"
    ALTER COLUMN "organization_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "workflow_versions"
    ALTER COLUMN "business_unit_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "workflow_versions"
    ALTER COLUMN "workflow_template_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "workflow_versions"
    ALTER COLUMN "version_status" SET STATISTICS 1000;

--bun:split
ALTER TABLE "workflow_nodes"
    ALTER COLUMN "organization_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "workflow_nodes"
    ALTER COLUMN "business_unit_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "workflow_nodes"
    ALTER COLUMN "workflow_version_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "workflow_connections"
    ALTER COLUMN "organization_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "workflow_connections"
    ALTER COLUMN "business_unit_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "workflow_connections"
    ALTER COLUMN "workflow_version_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "workflow_instances"
    ALTER COLUMN "organization_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "workflow_instances"
    ALTER COLUMN "business_unit_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "workflow_instances"
    ALTER COLUMN "workflow_template_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "workflow_instances"
    ALTER COLUMN "workflow_version_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "workflow_instances"
    ALTER COLUMN "status" SET STATISTICS 1000;

--bun:split
ALTER TABLE "workflow_node_executions"
    ALTER COLUMN "organization_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "workflow_node_executions"
    ALTER COLUMN "business_unit_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "workflow_node_executions"
    ALTER COLUMN "workflow_instance_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "workflow_node_executions"
    ALTER COLUMN "status" SET STATISTICS 1000;
