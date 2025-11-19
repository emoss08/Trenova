SET statement_timeout = 0;

-- Drop triggers
DROP TRIGGER IF EXISTS workflow_templates_search_update ON "workflow_templates";

--bun:split
-- Drop functions
DROP FUNCTION IF EXISTS workflow_templates_search_trigger();

--bun:split
-- Drop tables in reverse order of creation
DROP TABLE IF EXISTS "workflow_node_executions";

--bun:split
DROP TABLE IF EXISTS "workflow_instances";

--bun:split
DROP TABLE IF EXISTS "workflow_connections";

--bun:split
DROP TABLE IF EXISTS "workflow_nodes";

--bun:split
-- Remove FK constraint before dropping workflow_versions
ALTER TABLE "workflow_templates"
    DROP CONSTRAINT IF EXISTS "fk_workflow_templates_published_version";

--bun:split
DROP TABLE IF EXISTS "workflow_versions";

--bun:split
DROP TABLE IF EXISTS "workflow_templates";

--bun:split
-- Drop enums
DROP TYPE IF EXISTS workflow_execution_mode_enum;

--bun:split
DROP TYPE IF EXISTS workflow_node_execution_status_enum;

--bun:split
DROP TYPE IF EXISTS workflow_node_type_enum;

--bun:split
DROP TYPE IF EXISTS workflow_instance_status_enum;

--bun:split
DROP TYPE IF EXISTS workflow_version_status_enum;

--bun:split
DROP TYPE IF EXISTS workflow_status_enum;

--bun:split
DROP TYPE IF EXISTS workflow_trigger_type_enum;
