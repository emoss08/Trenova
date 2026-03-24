--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
CREATE TABLE IF NOT EXISTS "hold_reasons"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "type" hold_type_enum NOT NULL,
    "code" varchar(64) NOT NULL,
    "label" varchar(100) NOT NULL,
    "description" text,
    "default_severity" hold_severity_enum NOT NULL DEFAULT 'Advisory',
    "default_blocks_dispatch" boolean NOT NULL DEFAULT FALSE,
    "default_blocks_delivery" boolean NOT NULL DEFAULT FALSE,
    "default_blocks_billing" boolean NOT NULL DEFAULT FALSE,
    "default_visible_to_customer" boolean NOT NULL DEFAULT FALSE,
    "active" boolean NOT NULL DEFAULT TRUE,
    "sort_order" integer NOT NULL DEFAULT 100,
    "external_map" jsonb,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_hold_reasons" PRIMARY KEY ("id", "organization_id"),
    CONSTRAINT "fk_hr_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_hr_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ux_hr_org_bu_type_code" UNIQUE ("organization_id", "business_unit_id", "type", "code")
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_hr_org_bu_type_active" ON "hold_reasons"("organization_id", "business_unit_id", "type")
WHERE
    active = TRUE;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_hr_code_type_label" ON "hold_reasons"("code", "type", "label");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_hr_status" ON "hold_reasons"("active");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_hr_external_map" ON "hold_reasons" USING GIN("external_map");

--bun:split
COMMENT ON TABLE hold_reasons IS 'Reasons for holds. Each reason can be associated with a specific hold type and severity, with default settings for blocking and visibility.';

--bun:split
ALTER TABLE "hold_reasons"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_hr_search" ON "hold_reasons" USING GIN(search_vector);

--bun:split
CREATE OR REPLACE FUNCTION hold_reasons_search_vector_update()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.search_vector := setweight(to_tsvector('simple', COALESCE(NEW.code, '')), 'A') || setweight(to_tsvector('simple', COALESCE(NEW.label, '')), 'B') || setweight(to_tsvector('english', COALESCE(CAST(NEW.type AS text), '')), 'C') || setweight(to_tsvector('english', COALESCE(CAST(NEW.default_severity AS text), '')), 'D');
    -- Auto-update timestamps
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS hold_reasons_search_vector_trigger ON hold_reasons;

--bun:split
CREATE TRIGGER hold_reasons_search_vector_trigger
    BEFORE INSERT OR UPDATE ON hold_reasons
    FOR EACH ROW
    EXECUTE FUNCTION hold_reasons_search_vector_update();

--bun:split
ALTER TABLE hold_reasons
    ALTER COLUMN active SET STATISTICS 1000;

--bun:split
ALTER TABLE hold_reasons
    ALTER COLUMN organization_id SET STATISTICS 1000;

