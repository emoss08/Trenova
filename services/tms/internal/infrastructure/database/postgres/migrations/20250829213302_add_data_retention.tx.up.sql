-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
CREATE TABLE IF NOT EXISTS "data_retention"(
  "id" varchar(100) NOT NULL,
  "organization_id" varchar(100) NOT NULL,
  "business_unit_id" varchar(100) NOT NULL,
  "audit_retention_period" integer NOT NULL DEFAULT 120,
  "version" bigint NOT NULL DEFAULT 0,
  "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
  "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
  CONSTRAINT "pk_data_retention" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
  CONSTRAINT "fk_dr_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "fk_dr_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "uq_data_retention_organization" UNIQUE ("organization_id")
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_data_retention_bu_org" ON "data_retention"("business_unit_id", "organization_id");

CREATE INDEX IF NOT EXISTS "idx_data_rention_timestamps" ON "data_retention"("created_at", "updated_at");

--bun:split
COMMENT ON TABLE "data_retention" IS 'Stores configuration for data retention policy';

--bun:split
CREATE OR REPLACE FUNCTION "data_retention_update_timestamp"()
  RETURNS TRIGGER
  AS $$
BEGIN
  NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
  RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS "data_retention_update_timestamp_trigger" ON "data_retention";

--bun:split
CREATE TRIGGER "data_retention_update_timestamp_trigger"
  BEFORE UPDATE ON "data_retention"
  FOR EACH ROW
  EXECUTE FUNCTION "data_retention_update_timestamp"();

--bun:split
ALTER TABLE "data_retention"
  ALTER COLUMN "organization_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "data_retention"
  ALTER COLUMN "business_unit_id" SET STATISTICS 1000;

