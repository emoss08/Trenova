-- Migration to add pro number sequences table
CREATE TABLE "pro_number_sequences" (
    "id" varchar(100) PRIMARY KEY,
    "organization_id" varchar(100) NOT NULL,
    "year" smallint NOT NULL,
    "month" smallint NOT NULL,
    "current_sequence" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(epoch FROM current_timestamp) ::bigint,
    "version" bigint NOT NULL DEFAULT 0,
    CONSTRAINT "fk_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id"),
    CONSTRAINT "uq_sequence_period" UNIQUE ("organization_id", "year", "month")
);

-- Create index for quick lookups
CREATE INDEX "idx_pro_number_sequences_lookup" ON "pro_number_sequences" ("organization_id", "year", "month");

-- Add pro_number column to shipments table if it doesn't exist
ALTER TABLE "shipments"
    ADD COLUMN IF NOT EXISTS "pro_number" VARCHAR(50) NOT NULL,
    ADD CONSTRAINT "uq_pro_number" UNIQUE ("pro_number", "organization_id");

