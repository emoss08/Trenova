-- Workflow Automation System Migration
-- Workflow Status Enum
CREATE TYPE "workflow_status_enum" AS ENUM(
    'draft', -- Being edited, not published
    'active', -- Published and can be triggered
    'inactive', -- Published but disabled
    'archived' -- Archived, cannot be activated
);

--bun:split
-- Workflow Trigger Type Enum
CREATE TYPE "workflow_trigger_type_enum" AS ENUM(
    'manual', -- Manually triggered
    'scheduled', -- Time-based (cron)
    'shipment_status', -- Shipment status change
    'document_uploaded', -- Document upload event
    'entity_created', -- Entity creation event
    'entity_updated', -- Entity update event
    'webhook' -- External webhook
);

--bun:split
-- Workflow Node Type Enum
CREATE TYPE "workflow_node_type_enum" AS ENUM(
    'trigger', -- Trigger node (start)
    'action', -- Action node
    'condition', -- If/else condition
    'loop', -- Loop/iteration
    'delay', -- Delay/wait
    'end' -- End node
);

--bun:split
-- Workflow Action Type Enum
CREATE TYPE "workflow_action_type_enum" AS ENUM(
    -- Shipment actions
    'shipment_update_status',
    'shipment_assign_carrier',
    'shipment_assign_driver',
    'shipment_update_field',
    -- Billing actions
    'billing_validate_requirements',
    'billing_transfer_to_queue',
    'billing_generate_invoice',
    'billing_send_invoice',
    -- Document actions
    'document_validate_completeness',
    'document_request_missing',
    'document_generate',
    -- Notification actions
    'notification_send_email',
    'notification_send_sms',
    'notification_send_webhook',
    'notification_send_push',
    -- Data actions
    'data_transform',
    'data_api_call',
    'data_database_query',
    -- Flow control
    'flow_approval_request',
    'flow_wait_for_event',
    'flow_parallel_execution'
);

--bun:split
-- Workflow Execution Status Enum
CREATE TYPE "workflow_execution_status_enum" AS ENUM(
    'pending', -- Queued for execution
    'running', -- Currently executing
    'paused', -- Paused by user
    'completed', -- Completed successfully
    'failed', -- Failed with error
    'canceled', -- Canceled by user
    'timeout' -- Execution timeout
);

--bun:split
-- Workflow Execution Step Status Enum
CREATE TYPE "workflow_execution_step_status_enum" AS ENUM(
    'pending',
    'running',
    'completed',
    'failed',
    'skipped',
    'retrying'
);

--bun:split
-- Main Workflows Table
CREATE TABLE IF NOT EXISTS "workflows"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    -- Basic Info
    "name" varchar(255) NOT NULL,
    "description" text,
    "status" workflow_status_enum NOT NULL DEFAULT 'draft',
    -- Trigger Configuration
    "trigger_type" workflow_trigger_type_enum NOT NULL,
    "trigger_config" jsonb NOT NULL DEFAULT '{}',
    -- Versioning
    "current_version_id" varchar(100),
    "published_version_id" varchar(100),
    -- Execution Settings
    "timeout_seconds" integer DEFAULT 300,
    "max_retries" integer DEFAULT 3,
    "retry_delay_seconds" integer DEFAULT 60,
    "enable_logging" boolean DEFAULT TRUE,
    "enable_notifications" boolean DEFAULT FALSE,
    -- Permissions
    "created_by" varchar(100) NOT NULL,
    "updated_by" varchar(100),
    -- Tags and Categories
    "tags" text[] DEFAULT '{}',
    "category" varchar(100),
    -- Metadata
    "version" integer NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL,
    "updated_at" bigint NOT NULL,
    CONSTRAINT "pk_workflows" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_workflows_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT "fk_workflows_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE CASCADE ON DELETE CASCADE
);

--bun:split
-- Workflow Versions Table (for version history)
CREATE TABLE IF NOT EXISTS "workflow_versions"(
    "id" varchar(100) NOT NULL,
    "workflow_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    -- Version Info
    "version_number" integer NOT NULL,
    "version_name" varchar(255),
    "changelog" text,
    -- Workflow Definition (stored as JSON)
    "workflow_definition" jsonb NOT NULL,
    -- Status
    "is_published" boolean DEFAULT FALSE,
    "published_at" bigint,
    "published_by" varchar(100),
    -- Metadata
    "created_by" varchar(100) NOT NULL,
    "created_at" bigint NOT NULL,
    CONSTRAINT "pk_workflow_versions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_workflow_versions_workflow" FOREIGN KEY ("workflow_id", "organization_id", "business_unit_id") REFERENCES workflows(id, organization_id, business_unit_id) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT "uq_workflow_version" UNIQUE ("workflow_id", "version_number", "organization_id", "business_unit_id")
);

--bun:split
-- Workflow Nodes Table
CREATE TABLE IF NOT EXISTS "workflow_nodes"(
    "id" varchar(100) NOT NULL,
    "workflow_version_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    -- Node Info
    "node_key" varchar(100) NOT NULL, -- Unique key within workflow (e.g., "node_1", "node_2")
    "node_type" workflow_node_type_enum NOT NULL,
    "action_type" workflow_action_type_enum,
    -- Display
    "label" varchar(255) NOT NULL,
    "description" text,
    -- Configuration
    "config" jsonb NOT NULL DEFAULT '{}',
    -- Position (for UI canvas)
    "position_x" double precision DEFAULT 0,
    "position_y" double precision DEFAULT 0,
    -- Metadata
    "created_at" bigint NOT NULL,
    CONSTRAINT "pk_workflow_nodes" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_workflow_nodes_version" FOREIGN KEY ("workflow_version_id", "organization_id", "business_unit_id") REFERENCES workflow_versions(id, organization_id, business_unit_id) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT "uq_workflow_node_key" UNIQUE ("workflow_version_id", "node_key", "organization_id", "business_unit_id")
);

--bun:split
-- Workflow Edges Table (connections between nodes)
CREATE TABLE IF NOT EXISTS "workflow_edges"(
    "id" varchar(100) NOT NULL,
    "workflow_version_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    -- Edge Info
    "source_node_id" varchar(100) NOT NULL,
    "target_node_id" varchar(100) NOT NULL,
    "source_handle" varchar(100), -- For multiple outputs (e.g., "true", "false" for conditions)
    "target_handle" varchar(100),
    -- Condition (for conditional edges)
    "condition" jsonb,
    -- Display
    "label" varchar(255),
    -- Metadata
    "created_at" bigint NOT NULL,
    CONSTRAINT "pk_workflow_edges" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_workflow_edges_version" FOREIGN KEY ("workflow_version_id", "organization_id", "business_unit_id") REFERENCES workflow_versions(id, organization_id, business_unit_id) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_edges_source" FOREIGN KEY ("source_node_id", "organization_id", "business_unit_id") REFERENCES workflow_nodes(id, organization_id, business_unit_id) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT fk_workflow_edges_target FOREIGN KEY ("target_node_id", "organization_id", "business_unit_id") REFERENCES workflow_nodes(id, organization_id, business_unit_id) ON UPDATE CASCADE ON DELETE CASCADE
);

--bun:split
-- Workflow Executions Table
CREATE TABLE IF NOT EXISTS "workflow_executions"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    -- Workflow Reference
    "workflow_id" varchar(100) NOT NULL,
    "workflow_version_id" varchar(100) NOT NULL,
    -- Execution Info
    "status" workflow_execution_status_enum NOT NULL DEFAULT 'pending',
    "trigger_type" workflow_trigger_type_enum NOT NULL,
    -- Trigger Context
    "trigger_data" jsonb NOT NULL DEFAULT '{}',
    "triggered_by" varchar(100), -- User ID if manual
    -- Temporal Workflow Info
    "temporal_workflow_id" varchar(255),
    "temporal_run_id" varchar(255),
    -- Execution Results
    "input_data" jsonb,
    "output_data" jsonb,
    "error_message" text,
    "error_stack" text,
    -- Timing
    "started_at" bigint,
    "completed_at" bigint,
    "duration_ms" bigint,
    -- Retry Info
    "retry_count" integer DEFAULT 0,
    "max_retries" integer DEFAULT 3,
    -- Metadata
    "version" integer NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL,
    "updated_at" bigint NOT NULL,
    CONSTRAINT "pk_workflow_executions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_workflow_executions_workflow" FOREIGN KEY ("workflow_id", "organization_id", "business_unit_id") REFERENCES workflows(id, organization_id, business_unit_id) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_executions_version" FOREIGN KEY ("workflow_version_id", "organization_id", "business_unit_id") REFERENCES workflow_versions(id, organization_id, business_unit_id) ON UPDATE CASCADE ON DELETE CASCADE
);

--bun:split
-- Workflow Execution Steps Table (detailed step logs)
CREATE TABLE IF NOT EXISTS "workflow_execution_steps"(
    "id" varchar(100) NOT NULL,
    "execution_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    -- Node Reference
    "node_id" varchar(100) NOT NULL,
    "node_key" varchar(100) NOT NULL,
    "node_type" workflow_node_type_enum NOT NULL,
    "action_type" workflow_action_type_enum,
    -- Step Info
    "step_number" integer NOT NULL,
    "status" workflow_execution_step_status_enum NOT NULL DEFAULT 'pending',
    -- Execution Data
    "input_data" jsonb,
    "output_data" jsonb,
    "error_message" text,
    "error_stack" text,
    -- Timing
    "started_at" bigint,
    "completed_at" bigint,
    "duration_ms" bigint,
    -- Retry Info
    "retry_count" integer DEFAULT 0,
    -- Metadata
    "created_at" bigint NOT NULL,
    "updated_at" bigint NOT NULL,
    CONSTRAINT "pk_workflow_execution_steps" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_workflow_execution_steps_execution" FOREIGN KEY ("execution_id", "organization_id", "business_unit_id") REFERENCES workflow_executions(id, organization_id, business_unit_id) ON UPDATE CASCADE ON DELETE CASCADE
);

--bun:split
-- Workflow Templates Table
CREATE TABLE IF NOT EXISTS "workflow_templates"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    -- Template Info
    "name" varchar(255) NOT NULL,
    "description" text,
    "category" varchar(100),
    "tags" text[] DEFAULT '{}',
    -- Template Definition
    "template_definition" jsonb NOT NULL,
    -- Visibility
    "is_system_template" boolean DEFAULT FALSE, -- System-wide templates
    "is_public" boolean DEFAULT FALSE,
    -- Usage Stats
    "usage_count" integer DEFAULT 0,
    -- Metadata
    "created_by" varchar(100) NOT NULL,
    "version" integer NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL,
    "updated_at" bigint NOT NULL,
    CONSTRAINT "pk_workflow_templates" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_workflow_templates_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT "fk_workflow_templates_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE CASCADE ON DELETE CASCADE
);

--bun:split
-- Indexes for workflows
CREATE INDEX idx_workflows_org_bu ON workflows(organization_id, business_unit_id);

CREATE INDEX idx_workflows_status ON workflows(status);

CREATE INDEX idx_workflows_trigger_type ON workflows(trigger_type);

CREATE INDEX idx_workflows_created_by ON workflows(created_by);

CREATE INDEX idx_workflows_tags ON workflows USING GIN(tags);

CREATE INDEX idx_workflows_created_at ON workflows(created_at DESC);

--bun:split
-- Indexes for workflow_versions
CREATE INDEX idx_workflow_versions_workflow ON workflow_versions(workflow_id, organization_id, business_unit_id);

CREATE INDEX idx_workflow_versions_published ON workflow_versions(is_published)
WHERE
    is_published = TRUE;

CREATE INDEX idx_workflow_versions_number ON workflow_versions(workflow_id, version_number DESC);

--bun:split
-- Indexes for workflow_nodes
CREATE INDEX idx_workflow_nodes_version ON workflow_nodes(workflow_version_id, organization_id, business_unit_id);

CREATE INDEX idx_workflow_nodes_type ON workflow_nodes(node_type);

CREATE INDEX idx_workflow_nodes_action_type ON workflow_nodes(action_type);

--bun:split
-- Indexes for workflow_edges
CREATE INDEX idx_workflow_edges_version ON workflow_edges(workflow_version_id, organization_id, business_unit_id);

CREATE INDEX idx_workflow_edges_source ON workflow_edges(source_node_id);

CREATE INDEX idx_workflow_edges_target ON workflow_edges(target_node_id);

--bun:split
-- Indexes for workflow_executions
CREATE INDEX idx_workflow_executions_workflow ON workflow_executions(workflow_id, organization_id, business_unit_id);

CREATE INDEX idx_workflow_executions_status ON workflow_executions(status);

CREATE INDEX idx_workflow_executions_temporal ON workflow_executions(temporal_workflow_id)
WHERE
    temporal_workflow_id IS NOT NULL;

CREATE INDEX idx_workflow_executions_created_at ON workflow_executions(created_at DESC);

CREATE INDEX idx_workflow_executions_completed_at ON workflow_executions(completed_at DESC)
WHERE
    completed_at IS NOT NULL;

CREATE INDEX idx_workflow_executions_trigger_type ON workflow_executions(trigger_type);

CREATE INDEX idx_workflow_executions_triggered_by ON workflow_executions(triggered_by)
WHERE
    triggered_by IS NOT NULL;

-- BRIN index for time-series queries
CREATE INDEX idx_workflow_executions_dates_brin ON workflow_executions USING BRIN(created_at, completed_at);

--bun:split
-- Indexes for workflow_execution_steps
CREATE INDEX idx_workflow_execution_steps_execution ON workflow_execution_steps(execution_id, organization_id, business_unit_id);

CREATE INDEX idx_workflow_execution_steps_status ON workflow_execution_steps(status);

CREATE INDEX idx_workflow_execution_steps_step_number ON workflow_execution_steps(execution_id, step_number);

CREATE INDEX idx_workflow_execution_steps_node ON workflow_execution_steps(node_id);

--bun:split
-- Indexes for workflow_templates
CREATE INDEX idx_workflow_templates_org_bu ON workflow_templates(organization_id, business_unit_id);

CREATE INDEX idx_workflow_templates_category ON workflow_templates(category);

CREATE INDEX idx_workflow_templates_system ON workflow_templates(is_system_template)
WHERE
    is_system_template = TRUE;

CREATE INDEX idx_workflow_templates_public ON workflow_templates(is_public)
WHERE
    is_public = TRUE;

CREATE INDEX idx_workflow_templates_tags ON workflow_templates USING GIN(tags);

CREATE INDEX idx_workflow_templates_usage ON workflow_templates(usage_count DESC);

--bun:split
-- Statistics collection
ALTER TABLE workflows SET (autovacuum_vacuum_scale_factor = 0.0);

ALTER TABLE workflows SET (autovacuum_vacuum_threshold = 5000);

ALTER TABLE workflows SET (autovacuum_analyze_scale_factor = 0.0);

ALTER TABLE workflows SET (autovacuum_analyze_threshold = 5000);

ALTER TABLE workflow_executions SET (autovacuum_vacuum_scale_factor = 0.0);

ALTER TABLE workflow_executions SET (autovacuum_vacuum_threshold = 5000);

ALTER TABLE workflow_executions SET (autovacuum_analyze_scale_factor = 0.0);

ALTER TABLE workflow_executions SET (autovacuum_analyze_threshold = 5000);

