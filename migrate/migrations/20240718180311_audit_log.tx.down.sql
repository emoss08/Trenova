DROP TYPE IF EXISTS audit_log_status_enum CASCADE;

-- bun:split

DROP TABLE IF EXISTS audit_logs;

-- bun:split

DROP INDEX IF EXISTS idx_audit_logs_table_name;

-- bun:split

DROP INDEX IF EXISTS idx_audit_logs_entity_id;

-- bun:split

DROP INDEX IF EXISTS idx_audit_logs_user_id;

-- bun:split

DROP INDEX IF EXISTS idx_audit_logs_username;