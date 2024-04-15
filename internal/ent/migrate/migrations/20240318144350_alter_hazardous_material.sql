-- Modify "revenue_codes" table
ALTER TABLE "revenue_codes"
ADD COLUMN "status" character varying NOT NULL DEFAULT 'A';

-- Create "charge_types" table
CREATE TABLE
    "charge_types" (
        "id" uuid NOT NULL,
        "status" character varying NOT NULL DEFAULT 'A',
        PRIMARY KEY ("id")
    );