-- Audit entries table with enhanced structure and constraints
CREATE TABLE IF NOT EXISTS "audit_entries" (
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "resource" varchar(50) NOT NULL,
    "resource_id" varchar(100) NOT NULL,
    "action" action_enum NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "timestamp" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "changes" jsonb DEFAULT '{}' ::jsonb,
    "previous_state" jsonb DEFAULT '{}' ::jsonb,
    "current_state" jsonb DEFAULT '{}' ::jsonb,
    "metadata" jsonb DEFAULT '{}' ::jsonb,
    "user_agent" varchar(255),
    "correlation_id" varchar(100),
    "comment" text,
    "sensitive_data" boolean NOT NULL DEFAULT FALSE,
    PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_audit_entries_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_audit_entries_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_audit_entries_user" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "check_changes_format" CHECK (jsonb_typeof(changes) = 'object'),
    CONSTRAINT "check_previous_state_format" CHECK (jsonb_typeof(previous_state) = 'object'),
    CONSTRAINT "check_current_state_format" CHECK (jsonb_typeof(current_state) = 'object'),
    CONSTRAINT "check_metadata_format" CHECK (jsonb_typeof(metadata) = 'object')
);

--bun:split
-- Primary time-based index for queries
CREATE INDEX "idx_audit_entries_timestamp" ON "audit_entries" ("timestamp");

-- Composite indexes for common query patterns
CREATE INDEX "idx_audit_entries_resource_lookup" ON "audit_entries" ("resource", "resource_id", "timestamp");

CREATE INDEX "idx_audit_entries_org_time" ON "audit_entries" ("organization_id", "timestamp");

CREATE INDEX "idx_audit_entries_business_unit_time" ON "audit_entries" ("business_unit_id", "timestamp");

CREATE INDEX "idx_audit_entries_user_time" ON "audit_entries" ("user_id", "timestamp");

-- Additional indexes for filtering and relationships
CREATE INDEX "idx_audit_entries_action" ON "audit_entries" ("action");

CREATE INDEX "idx_audit_entries_correlation" ON "audit_entries" ("correlation_id")
WHERE
    correlation_id IS NOT NULL;

CREATE INDEX "idx_audit_entries_sensitive" ON "audit_entries" ("sensitive_data")
WHERE
    sensitive_data = TRUE;

-- JSONB indexes for efficient querying
CREATE INDEX "idx_audit_entries_changes" ON "audit_entries" USING gin ("changes");

CREATE INDEX "idx_audit_entries_previous_state" ON "audit_entries" USING gin ("previous_state");

CREATE INDEX "idx_audit_entries_current_state" ON "audit_entries" USING gin ("current_state");

CREATE INDEX "idx_audit_entries_metadata" ON "audit_entries" USING gin ("metadata");

-- Table and column comments
COMMENT ON TABLE audit_entries IS 'Append-only audit log for tracking system changes and user actions';

--bun:split
-- Function to prevent updates/deletes on audit_entries
CREATE OR REPLACE FUNCTION prevent_audit_modification ()
    RETURNS TRIGGER
    AS $$
BEGIN
    RAISE EXCEPTION 'Modifications are not allowed on audit_entries (append-only table)';
END;
$$
LANGUAGE plpgsql;

-- Trigger to enforce append-only behavior
CREATE TRIGGER enforce_audit_append_only
    BEFORE UPDATE OR DELETE ON "audit_entries"
    FOR EACH ROW
    EXECUTE FUNCTION prevent_audit_modification ();

--bun:split
ALTER TABLE audit_entries
    ADD COLUMN IF NOT EXISTS category VARCHAR(50) NOT NULL DEFAULT 'system',
    ADD COLUMN IF NOT EXISTS critical BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS ip_address VARCHAR(45);

--bun:split
-- Add indexes to improve query performance
CREATE INDEX IF NOT EXISTS idx_audit_entries_category ON audit_entries (category);

CREATE INDEX IF NOT EXISTS idx_audit_entries_critical ON audit_entries (critical);

CREATE INDEX IF NOT EXISTS idx_audit_entries_timestamp ON audit_entries (timestamp);

