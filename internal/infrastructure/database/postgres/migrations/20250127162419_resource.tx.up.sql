CREATE TABLE IF NOT EXISTS "resource_definitions"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "resource_type" varchar(100) NOT NULL,
    "display_name" varchar(100) NOT NULL,
    "table_name" varchar(100) NOT NULL,
    "description" text NOT NULL,
    "allow_custom_fields" boolean NOT NULL DEFAULT false,
    "allow_automations" boolean NOT NULL DEFAULT false,
    "allow_notifications" boolean NOT NULL DEFAULT false,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint
);

CREATE UNIQUE INDEX "idx_resource_definitions_resource_type" ON "resource_definitions"("resource_type");

CREATE INDEX "idx_resource_definitions_created_updated" ON "resource_definitions"("created_at", "updated_at");

COMMENT ON TABLE "resource_definitions" IS 'Stores information about resource definitions';