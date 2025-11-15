-- Drop workflow tables in reverse order

--bun:split
DROP TABLE IF EXISTS "workflow_templates" CASCADE;

--bun:split
DROP TABLE IF EXISTS "workflow_execution_steps" CASCADE;

--bun:split
DROP TABLE IF EXISTS "workflow_executions" CASCADE;

--bun:split
DROP TABLE IF EXISTS "workflow_edges" CASCADE;

--bun:split
DROP TABLE IF EXISTS "workflow_nodes" CASCADE;

--bun:split
DROP TABLE IF EXISTS "workflow_versions" CASCADE;

--bun:split
DROP TABLE IF EXISTS "workflows" CASCADE;

-- Drop enums
--bun:split
DROP TYPE IF EXISTS "workflow_execution_step_status_enum" CASCADE;

--bun:split
DROP TYPE IF EXISTS "workflow_execution_status_enum" CASCADE;

--bun:split
DROP TYPE IF EXISTS "workflow_action_type_enum" CASCADE;

--bun:split
DROP TYPE IF EXISTS "workflow_node_type_enum" CASCADE;

--bun:split
DROP TYPE IF EXISTS "workflow_trigger_type_enum" CASCADE;

--bun:split
DROP TYPE IF EXISTS "workflow_status_enum" CASCADE;
