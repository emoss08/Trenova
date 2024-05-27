-- Modify "rates" table
ALTER TABLE "rates" ADD COLUMN "approved_by" uuid NULL, ADD COLUMN "approved_date" date NULL, ADD COLUMN "usage_count" bigint NULL DEFAULT 0, ADD COLUMN "minimum_charge" numeric(19,4) NULL, ADD COLUMN "maximum_charge" numeric(19,4) NULL;
