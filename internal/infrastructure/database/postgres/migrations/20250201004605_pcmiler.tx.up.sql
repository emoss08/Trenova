CREATE TABLE IF NOT EXISTS "pcmiler_configurations"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    -- Core fields
    "api_key" varchar(255) NOT NULL,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_pcmiler_configurations" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_pcmiler_configurations_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_pcmiler_configurations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- Ensure the organization has one PCMiler configuration
CREATE UNIQUE INDEX "idx_pcmiler_configurations_organization" ON "pcmiler_configurations"("organization_id");
