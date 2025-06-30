-- Create enum for sequence types
CREATE TYPE "sequence_type_enum" AS ENUM(
    'pro_number',
    'consolidation',
    'invoice',
    'work_order'
);

-- Create the sequences table
CREATE TABLE IF NOT EXISTS "sequences"(
    "id" varchar(100) PRIMARY KEY,
    "sequence_type" sequence_type_enum NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "year" int2 NOT NULL CHECK (year >= 1900 AND year <= 2100),
    "month" int2 NOT NULL CHECK (month >= 1 AND month <= 12),
    "current_sequence" bigint NOT NULL DEFAULT 0 CHECK (current_sequence >= 0),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint,
    -- Foreign keys
    CONSTRAINT "fk_sequences_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_sequences_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    -- Unique constraint to ensure one sequence per type/org/bu/year/month
    CONSTRAINT "uk_sequences_unique_key" UNIQUE ("sequence_type", "organization_id", "business_unit_id", "year", "month")
);

-- Create indexes for better performance
--bun:split
CREATE INDEX "idx_sequences_org_type" ON sequences("organization_id", "sequence_type");

--bun:split
CREATE INDEX "idx_sequences_org_bu_type" ON sequences("organization_id", "business_unit_id", "sequence_type")
WHERE
    "business_unit_id" IS NOT NULL;

--bun:split
CREATE INDEX "idx_sequences_year_month" ON sequences("year", "month");

-- Migrate existing pro_number_sequences data if the table exists
--bun:split
DO $$
BEGIN
    IF EXISTS(
        SELECT
            1
        FROM
            information_schema.tables
        WHERE
            table_name = 'pro_number_sequences') THEN
    INSERT INTO sequences("id", "sequence_type", "organization_id", "business_unit_id", "year", "month", "current_sequence", "version", "created_at", "updated_at")
    SELECT
        "id",
        'pro_number'::sequence_type_enum,
        "organization_id",
        "business_unit_id",
        "year",
        "month",
        "current_sequence",
        "version",
        "created_at",
        "updated_at"
    FROM
        "pro_number_sequences"
    ON CONFLICT("sequence_type",
        "organization_id",
        "business_unit_id",
        "year",
        "month")
        DO NOTHING;
END IF;
END
$$;

-- Add replica identity for CDC if needed
ALTER TABLE "sequences" REPLICA IDENTITY
    FULL;

