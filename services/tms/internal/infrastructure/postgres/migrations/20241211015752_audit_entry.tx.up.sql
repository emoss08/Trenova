CREATE TYPE "audit_entry_category_enum" AS ENUM(
    'System',
    'User'
);

CREATE TABLE IF NOT EXISTS "audit_entries"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "resource" varchar(50) NOT NULL,
    "resource_id" varchar(100) NOT NULL,
    "operation" integer NOT NULL,
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
    CONSTRAINT "fk_audit_entries_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_audit_entries_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_audit_entries_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "check_changes_format" CHECK (jsonb_typeof(changes) = 'object'),
    CONSTRAINT "check_previous_state_format" CHECK (jsonb_typeof(previous_state) = 'object'),
    CONSTRAINT "check_current_state_format" CHECK (jsonb_typeof(current_state) = 'object'),
    CONSTRAINT "check_metadata_format" CHECK (jsonb_typeof(metadata) = 'object')
);

--bun:split
-- Primary time-based index for queries
CREATE INDEX "idx_audit_entries_timestamp" ON "audit_entries"("timestamp");

-- Table and column comments
COMMENT ON TABLE "audit_entries" IS 'Append-only audit log for tracking system changes and user actions';

--bun:split
-- Function to prevent updates/deletes on audit_entries
CREATE OR REPLACE FUNCTION prevent_audit_modification()
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
    EXECUTE FUNCTION prevent_audit_modification();

--bun:split
ALTER TABLE audit_entries
    ADD COLUMN IF NOT EXISTS "category" varchar(50) NOT NULL DEFAULT 'System',
    ADD COLUMN IF NOT EXISTS "critical" boolean NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS "ip_address" varchar(45);

