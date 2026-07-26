--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
-- A view is a named set of parameter values for one report. A parameterized
-- report is one question asked many ways, and without views every asking means
-- retyping every parameter. Views store only the answers, never the report, so
-- editing a definition reaches every view of it at once.
CREATE TABLE IF NOT EXISTS "report_views"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "definition_id" varchar(100) NOT NULL,
    "owner_id" varchar(100) NOT NULL,
    "name" varchar(120) NOT NULL,
    "description" text,
    "params" jsonb,
    "shared" boolean NOT NULL DEFAULT FALSE,
    "pinned" boolean NOT NULL DEFAULT FALSE,
    "format" varchar(10),
    "last_run_at" bigint,
    "run_count" bigint NOT NULL DEFAULT 0,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_report_views" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_report_views_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_report_views_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_report_views_owner" FOREIGN KEY ("owner_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    -- A view is meaningless without its report, so it dies with it rather than
    -- lingering as a preset nobody can open.
    CONSTRAINT "fk_report_views_definition" FOREIGN KEY ("definition_id", "business_unit_id", "organization_id") REFERENCES "report_definitions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "check_report_views_params_format" CHECK (params IS NULL OR jsonb_typeof(params) = 'object')
);

--bun:split
CREATE INDEX "idx_report_views_tenant" ON "report_views"("organization_id", "business_unit_id");

--bun:split
-- The picker's query: every view of one report the caller may see.
CREATE INDEX "idx_report_views_definition" ON "report_views"("definition_id", "organization_id", "business_unit_id");

--bun:split
CREATE INDEX "idx_report_views_owner" ON "report_views"("owner_id", "organization_id", "business_unit_id");

--bun:split
-- Names are unique per owner per report, not globally: two people may each keep
-- their own "My Region" view of the same report without colliding.
CREATE UNIQUE INDEX "idx_report_views_name" ON "report_views"(lower("name"), "definition_id", "owner_id", "organization_id", "business_unit_id");

--bun:split
COMMENT ON TABLE report_views IS 'Saved views: named parameter presets for a report definition';
